package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"math/big"
	mrand "math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"location-tracker/services"
	"location-tracker/types"
)

var (
	// In-memory cache (locations expire after 24 hours)
	locations     = make(map[string]types.Location)
	locationMutex sync.RWMutex

	// Error log cache (keep last 50 errors)
	errorLogs     = make([]types.ErrorLog, 0, 50)
	errorLogMutex sync.RWMutex

	// Nearby businesses from last shared location
	currentBusinesses     = make([]types.Business, 0, 5)
	currentBusinessesMutex sync.RWMutex

	// Commercial real estate cache (keyed by location name)
	commercialRealEstateCache     = make(map[string]types.CommercialRealEstate)
	commercialRealEstateCacheMutex sync.RWMutex

	// Pending user experience note from Twilio SMS
	pendingUserExperienceNote string
	pendingUserNoteKeywords   []string
	userExperienceNoteMutex   sync.RWMutex

	// Global password from environment
	globalPassword = os.Getenv("TRACKER_PASSWORD")

	// Google Maps API key
	googleMapsAPIKey = os.Getenv("GOOGLE_MAPS_API_KEY")

	// Perplexity API key
	perplexityAPIKey = os.Getenv("PERPLEXITY_API_KEY")

	// HTTPS mode flag
	useHTTPS = false

	// DynamoDB client
	dynamoClient *dynamodb.Client
	useDynamoDB  = false

	// DynamoDB table names
	errorLogsTableName            = "location-tracker-error-logs"
	locationsTableName            = "location-tracker-locations"
	commercialRealEstateTableName = "location-tracker-commercial-realestate"
	anonymousTipsTableName        = "location-tracker-anonymous-tips"
	bannedUsersTableName          = "location-tracker-banned-users"
	donationsTableName            = "location-tracker-donations"

	// Anonymous tips cache
	anonymousTips      = make([]types.AnonymousTip, 0, 100)
	anonymousTipsMutex sync.RWMutex
	pendingTipQueue    = make([]string, 0, 10) // Queue of tip IDs to attach to next errors
	pendingTipMutex    sync.RWMutex

	// Identity and moderation systems
	identityManager   *UserIdentityManager
	contentModerator  *ContentModerator
	rateLimiter       *RateLimiter
	banManager        *BanManager

	// Configuration from environment
	openaiAPIKey       = os.Getenv("OPENAI_API_KEY")
	tipEncryptionKey   = os.Getenv("TIP_ENCRYPTION_KEY")
	tipMaxLength       = 1000
	tipRateLimit       = 10 // tips per hour per user

	// Last interaction context - tracks the most recent user-driven interaction
	// All subsequent generated content (errors, GIFs, songs, etc.) traces back to this seed event
	lastInteractionContext     *types.LastInteractionContext
	lastInteractionContextMutex sync.RWMutex

	// Services
	businessService   *services.BusinessService
	commercialService *services.CommercialService
	contextService    *services.ContextService
)

func main() {
	// Initialize random seed for location generation
	mrand.Seed(time.Now().UnixNano())

	// Require password to be set
	if globalPassword == "" {
		log.Fatal("❌ TRACKER_PASSWORD environment variable must be set!")
	}

	// Check if HTTPS should be enabled
	if os.Getenv("USE_HTTPS") == "true" {
		useHTTPS = true
	}

	// Initialize DynamoDB connection (reads existing tables, never creates/modifies)
	initializeDynamoDB()

	// Initialize anonymous tip system
	initializeTipSystem()

	// Initialize services
	businessService = services.NewBusinessService(googleMapsAPIKey)
	commercialService = services.NewCommercialService(perplexityAPIKey)
	contextService = services.NewContextService()

	log.Printf("✅ Location tracker starting...")
	log.Printf("🔧 Services initialized (business, commercial, context)")
	log.Printf("🔒 Password authentication enabled")
	if useHTTPS {
		log.Printf("🔐 HTTPS mode enabled")
	}
	if useDynamoDB {
		log.Printf("💾 DynamoDB persistence enabled")
		log.Printf("📊 Error logs table: %s", errorLogsTableName)
		log.Printf("📍 Locations table: %s", locationsTableName)
	} else {
		log.Printf("⚠️  DynamoDB unavailable, using in-memory storage only")
	}

	// Routes
	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/verify-turnstile", handleVerifyTurnstile)
	http.HandleFunc("/api/create-payment-intent", handleCreatePaymentIntent)
	http.HandleFunc("/api/webhook/stripe", handleStripeWebhook)
	http.HandleFunc("/api/cryptogram", handleCryptogram)
	http.HandleFunc("/api/cryptogram/info", handleCryptogramInfo)
	http.HandleFunc("/api/location", handleLocation)
	http.HandleFunc("/api/errorlogs/", handleErrorLogByID)
	http.HandleFunc("/api/errorlogs", handleErrorLogs)
	http.HandleFunc("/api/facebook-share/", handleFacebookShare)
	http.HandleFunc("/api/share-image/", handleShareImage)
	http.HandleFunc("/api/rorschach/interpret/", handleRorschachInterpret)
	http.HandleFunc("/api/rorschach/respond/", handleRorschachUserResponse)
	http.HandleFunc("/api/businesses", handleBusinesses)
	http.HandleFunc("/api/keywords", handlePendingKeywords)
	http.HandleFunc("/api/commercial-context", handleCommercialContext)
	http.HandleFunc("/api/last-interaction-context", handleLastInteractionContext)
	http.HandleFunc("/api/commercialrealestate", handleCommercialRealEstate)
	http.HandleFunc("/api/health", handleHealth)
	http.HandleFunc("/api/twilio/sms", handleTwilioWebhook)
	http.HandleFunc("/api/tips", handleTips)
	http.HandleFunc("/api/tips/", handleTipByID)

	// Start cleanup goroutines
	go cleanupOldLocations()

	// Load existing data from DynamoDB on startup (preserves all existing records)
	if useDynamoDB {
		go loadExistingData()
	}

	httpPort := "8080"
	httpsPort := "8443"

	// Allow custom ports via environment
	if customHTTPPort := os.Getenv("HTTP_PORT"); customHTTPPort != "" {
		httpPort = customHTTPPort
	}
	if customHTTPSPort := os.Getenv("HTTPS_PORT"); customHTTPSPort != "" {
		httpsPort = customHTTPSPort
	}

	if useHTTPS {
		// Check for certificate files or generate self-signed ones
		certFile := os.Getenv("CERT_FILE")
		keyFile := os.Getenv("KEY_FILE")

		if certFile == "" || keyFile == "" {
			log.Printf("📜 No certificates provided, generating self-signed certificate...")
			certFile = "server.crt"
			keyFile = "server.key"

			if err := generateSelfSignedCert(certFile, keyFile); err != nil {
				log.Fatalf("❌ Failed to generate certificate: %v", err)
			}
			log.Printf("✅ Self-signed certificate generated")
		}

		// Start HTTP server for Twilio webhooks (in background)
		go func() {
			log.Printf("🌍 HTTP server running on http://:%s (for Twilio webhooks)", httpPort)
			if err := http.ListenAndServe(":"+httpPort, nil); err != nil {
				log.Fatalf("❌ HTTP server failed: %v", err)
			}
		}()

		// Start HTTPS server for browser access (main thread)
		log.Printf("🌍 HTTPS server running on https://:%s (for browser access)", httpsPort)
		log.Fatal(http.ListenAndServeTLS(":"+httpsPort, certFile, keyFile, nil))
	} else {
		log.Printf("⚠️  Running in HTTP mode - geolocation may not work in browsers!")
		log.Printf("💡 Set USE_HTTPS=true to enable HTTPS")
		log.Printf("🌍 Server running on http://:%s", httpPort)
		log.Fatal(http.ListenAndServe(":"+httpPort, nil))
	}
}

func fetchNearbyBusinesses(lat, lng float64) ([]types.Business, error) {
	return businessService.FetchNearbyBusinesses(lat, lng)
}

func getBusinessType(types []string) string {
	return businessService.GetBusinessType(types)
}

// extractUserKeywords extracts meaningful keywords from user notes for satirical purposes
func extractUserKeywords(userNote string) []string {
	keywords := make([]string, 0)

	// Remove common stopwords
	stopwords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "is": true, "are": true, "was": true,
		"were": true, "be": true, "been": true, "being": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "this": true, "that": true, "these": true, "those": true,
	}

	words := strings.FieldsFunc(strings.ToLower(userNote), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	})

	for _, word := range words {
		if len(word) > 3 && !stopwords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// extractLocationKeywords extracts keywords from location data (place names, addresses, business names)
func extractLocationKeywords(locationName string, businesses []types.Business) []string {
	return businessService.ExtractLocationKeywords(locationName, businesses)
}

// updateLastInteractionContext updates the global last interaction context
// This serves as the "seed event" for all subsequent generated content
func updateLastInteractionContext(interactionType string, keywords []string, sourceID string, locationName string, lat float64, lng float64, businesses []types.Business, rawContent string) {
	contextService.UpdateContext(interactionType, keywords, sourceID, locationName, lat, lng, businesses, rawContent)

	// Also update the global variable for backwards compatibility
	lastInteractionContextMutex.Lock()
	defer lastInteractionContextMutex.Unlock()
	lastInteractionContext = contextService.GetContext()
}

// getLastInteractionContext safely retrieves the current last interaction context
func getLastInteractionContext() *types.LastInteractionContext {
	lastInteractionContextMutex.RLock()
	defer lastInteractionContextMutex.RUnlock()

	if lastInteractionContext == nil {
		return nil
	}

	// Return a copy to avoid race conditions
	contextCopy := *lastInteractionContext
	return &contextCopy
}

// generateRandomLocationInRadius generates a random lat/lng within radiusMiles of the base location
func generateRandomLocationInRadius(baseLat, baseLng, radiusMiles float64) (float64, float64) {
	// Convert radius from miles to degrees (approximate)
	// 1 degree of latitude ≈ 69 miles
	// 1 degree of longitude varies by latitude
	radiusDegrees := radiusMiles / 69.0

	// Generate random angle and distance
	angle := mrand.Float64() * 2 * math.Pi
	distance := mrand.Float64() * radiusDegrees

	// Calculate offset
	latOffset := distance * math.Cos(angle)
	lngOffset := distance * math.Sin(angle) / math.Cos(baseLat*math.Pi/180.0)

	return baseLat + latOffset, baseLng + lngOffset
}

// getCachedCommercialRealEstate attempts to find cached commercial real estate data near a location
func getCachedCommercialRealEstate(queryLat, queryLng float64, radiusMiles float64) (*types.CommercialRealEstate, error) {
	if !useDynamoDB {
		return nil, nil
	}

	ctx := context.Background()

	// Scan the table for recent commercial real estate entries
	// Note: In production, you'd want a GSI on timestamp for better performance
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String(commercialRealEstateTableName),
	}

	result, err := dynamoClient.Scan(ctx, scanInput)
	if err != nil {
		log.Printf("⚠️  Failed to scan commercial real estate table: %v", err)
		return nil, err
	}

	var records []types.CommercialRealEstate
	err = attributevalue.UnmarshalListOfMaps(result.Items, &records)
	if err != nil {
		log.Printf("⚠️  Failed to unmarshal commercial real estate records: %v", err)
		return nil, err
	}

	// Find a record within the specified radius and not too old (30 days)
	cacheExpiryDuration := 30 * 24 * time.Hour
	now := time.Now()

	for i := range records {
		record := &records[i]

		// Check if record is not expired
		if now.Sub(record.Timestamp) > cacheExpiryDuration {
			continue
		}

		// Calculate distance between query point and cached record
		distance := calculateDistance(queryLat, queryLng, record.QueryLat, record.QueryLng)

		// If within radius, return this cached record
		if distance <= radiusMiles {
			log.Printf("💾 Cache HIT: Found cached commercial real estate data %.2f miles away (age: %v)",
				distance, now.Sub(record.Timestamp).Round(time.Hour))
			return record, nil
		}
	}

	log.Printf("🔍 Cache MISS: No cached data found within %.1f miles", radiusMiles)
	return nil, nil
}

// calculateDistance calculates the distance in miles between two lat/lng coordinates using Haversine formula
func calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusMiles = 3959.0

	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180
	lng1Rad := lng1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lng2Rad := lng2 * math.Pi / 180

	// Haversine formula
	dlat := lat2Rad - lat1Rad
	dlng := lng2Rad - lng1Rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dlng/2)*math.Sin(dlng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMiles * c
}

// searchCommercialRealEstate uses Perplexity API to find commercial real estate and businesses in an area
// It first checks the cache to avoid unnecessary API calls
func searchCommercialRealEstate(baseLat, baseLng float64, userKeywords []string) ([]types.CommercialPropertyDetails, []types.GoverningBody, float64, float64, error) {
	// Try to find cached data first
	cached, err := getCachedCommercialRealEstate(baseLat, baseLng, 5.0)
	if err == nil && cached != nil {
		return cached.Properties, cached.GoverningBodies, cached.QueryLat, cached.QueryLng, nil
	}

	// Use service for search
	return commercialService.SearchCommercialRealEstate(baseLat, baseLng, userKeywords)
}

// saveCommercialRealEstateToDynamoDB stores commercial real estate information in DynamoDB
func saveCommercialRealEstateToDynamoDB(commercialRealEstate types.CommercialRealEstate) {
	if !useDynamoDB {
		return
	}

	ctx := context.Background()

	item, err := attributevalue.MarshalMap(commercialRealEstate)
	if err != nil {
		log.Printf("❌ Failed to marshal commercial real estate: %v", err)
		return
	}

	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(commercialRealEstateTableName),
		Item:      item,
	})
	if err != nil {
		log.Printf("❌ Failed to save commercial real estate to DynamoDB: %v", err)
		return
	}

	log.Printf("💾 Commercial real estate info saved to DynamoDB: %s", commercialRealEstate.LocationName)
}

// saveTipToDynamoDB saves an anonymous tip to DynamoDB
func saveTipToDynamoDB(tip types.AnonymousTip) {
	if !useDynamoDB {
		return
	}

	ctx := context.Background()

	item, err := attributevalue.MarshalMap(tip)
	if err != nil {
		log.Printf("❌ Failed to marshal tip: %v", err)
		return
	}

	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(anonymousTipsTableName),
		Item:      item,
	})
	if err != nil {
		log.Printf("❌ Failed to save tip to DynamoDB: %v", err)
		return
	}

	log.Printf("💾 Anonymous tip saved to DynamoDB: %s", tip.ID)
}

// getTipFromDynamoDB retrieves a tip by ID from DynamoDB
func getTipFromDynamoDB(tipID string) (*types.AnonymousTip, error) {
	if !useDynamoDB {
		return nil, fmt.Errorf("DynamoDB not available")
	}

	ctx := context.Background()

	result, err := dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(anonymousTipsTableName),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: tipID},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, fmt.Errorf("tip not found")
	}

	var tip types.AnonymousTip
	if err := attributevalue.UnmarshalMap(result.Item, &tip); err != nil {
		return nil, err
	}

	return &tip, nil
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Constant-time comparison would be better for production
	if req.Password == globalPassword {
		// Set authentication cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "auth",
			Value:    "authenticated",
			HttpOnly: true,
			Secure:   useHTTPS, // Secure flag enabled when using HTTPS
			SameSite: http.SameSiteStrictMode,
			MaxAge:   86400, // 24 hours
			Path:     "/",
		})

		log.Printf("✅ Successful login from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	} else {
		// Add delay to prevent brute force
		time.Sleep(2 * time.Second)
		log.Printf("⚠️  Failed login attempt from %s", r.RemoteAddr)
		http.Error(w, "Invalid password", http.StatusUnauthorized)
	}
}

// handleVerifyTurnstile verifies Cloudflare Turnstile token and grants access
func handleVerifyTurnstile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token      string `json:"token"`
		ErrorLogID string `json:"error_log_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request",
		})
		return
	}

	// Verify the Turnstile token with Cloudflare
	turnstileSecretKey := os.Getenv("TURNSTILE_SECRET_KEY")
	if turnstileSecretKey == "" {
		// Use Cloudflare's test secret key that forces interactive challenge
		turnstileSecretKey = "2x0000000000000000000000000000000AA"
	}

	// Make verification request to Cloudflare
	verifyURL := "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	verifyData := map[string]string{
		"secret":   turnstileSecretKey,
		"response": req.Token,
	}

	verifyJSON, _ := json.Marshal(verifyData)
	resp, err := http.Post(verifyURL, "application/json", bytes.NewBuffer(verifyJSON))
	if err != nil {
		log.Printf("❌ Turnstile verification error: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Verification service unavailable",
		})
		return
	}
	defer resp.Body.Close()

	var verifyResp struct {
		Success bool `json:"success"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		log.Printf("❌ Failed to decode Turnstile response: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Verification failed",
		})
		return
	}

	if verifyResp.Success {
		// Set puzzle_solved cookie to grant access
		http.SetCookie(w, &http.Cookie{
			Name:     "auth",
			Value:    "puzzle_solved",
			HttpOnly: true,
			Secure:   useHTTPS,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   86400, // 24 hours
			Path:     "/",
		})

		log.Printf("✅ Turnstile verification successful for error log: %s from %s", req.ErrorLogID, r.RemoteAddr)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	} else {
		log.Printf("⚠️  Turnstile verification failed from %s", r.RemoteAddr)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "You appear to be a spy. Verification failed.",
		})
	}
}

// handleCreatePaymentIntent creates a Stripe Payment Intent for donations
func handleCreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DonationType string `json:"donation_type"` // "meme_disclaimer" or "church_committee"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid request",
		})
		return
	}

	// Determine amount and description based on donation type
	var amount int64
	var description string
	var bankRecordPurpose string

	switch req.DonationType {
	case "meme_disclaimer":
		amount = 50 // 50 cents (Stripe minimum)
		description = "Meme Disclaimer Record"
		bankRecordPurpose = "Not associated with these memes/GIFs - Historical comedic reference"
	case "church_committee":
		amount = 75 // 75 cents
		description = "Church Committee Historical Reference"
		bankRecordPurpose = "Local government transparency concern - Numerical church committee reference"
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid donation type",
		})
		return
	}

	stripeSecretKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeSecretKey == "" {
		log.Printf("⚠️  STRIPE_SECRET_KEY not configured")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Payment system not configured",
		})
		return
	}

	// Create Stripe Payment Intent
	paymentIntentURL := "https://api.stripe.com/v1/payment_intents"
	data := url.Values{}
	data.Set("amount", fmt.Sprintf("%d", amount))
	data.Set("currency", "usd")
	data.Set("description", description)
	data.Set("metadata[donation_type]", req.DonationType)
	data.Set("metadata[bank_purpose]", bankRecordPurpose)

	stripeReq, _ := http.NewRequest("POST", paymentIntentURL, strings.NewReader(data.Encode()))
	stripeReq.Header.Set("Authorization", "Bearer "+stripeSecretKey)
	stripeReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(stripeReq)
	if err != nil {
		log.Printf("❌ Stripe API error: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Payment service unavailable",
		})
		return
	}
	defer resp.Body.Close()

	var stripeResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stripeResp); err != nil {
		log.Printf("❌ Failed to decode Stripe response: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Payment processing error",
		})
		return
	}

	if resp.StatusCode != 200 {
		log.Printf("❌ Stripe error response: %v", stripeResp)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Payment creation failed",
		})
		return
	}

	// Store pending donation record
	donationID := fmt.Sprintf("%d", time.Now().UnixNano())
	donation := types.Donation{
		ID:                donationID,
		DonationType:      req.DonationType,
		Amount:            amount,
		StripePaymentID:   stripeResp["id"].(string),
		Timestamp:         time.Now(),
		IPAddress:         getClientIP(r),
		Status:            "pending",
		BankRecordPurpose: bankRecordPurpose,
	}

	if useDynamoDB {
		go saveDonationToDynamoDB(donation)
	}

	log.Printf("💰 Payment intent created: %s for %s ($%.2f)", donation.StripePaymentID, req.DonationType, float64(amount)/100)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"client_secret":      stripeResp["client_secret"],
		"amount":             amount,
		"description":        description,
		"bank_record_purpose": bankRecordPurpose,
	})
}

// handleStripeWebhook handles Stripe webhook events
func handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request", http.StatusBadRequest)
		return
	}

	// TODO: Verify webhook signature in production
	// For now, just parse the event
	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	eventType := event["type"].(string)
	log.Printf("📨 Stripe webhook received: %s", eventType)

	if eventType == "payment_intent.succeeded" {
		data := event["data"].(map[string]interface{})
		object := data["object"].(map[string]interface{})
		paymentIntentID := object["id"].(string)

		// Update donation status
		log.Printf("✅ Payment succeeded: %s", paymentIntentID)
		// TODO: Update DynamoDB record status to "succeeded"
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"received": true}`))
}

// saveDonationToDynamoDB saves donation record to DynamoDB
func saveDonationToDynamoDB(donation types.Donation) {
	ctx := context.Background()

	item, err := attributevalue.MarshalMap(donation)
	if err != nil {
		log.Printf("❌ Failed to marshal donation: %v", err)
		return
	}

	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(donationsTableName),
		Item:      item,
	})
	if err != nil {
		log.Printf("❌ Failed to save donation to DynamoDB: %v", err)
		return
	}

	log.Printf("💾 types.Donation saved to DynamoDB: %s", donation.ID)
}

func handleCryptogram(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Answer string `json:"answer"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get today's cryptogram
	crypto, err := GetTodaysCryptogram()
	if err != nil {
		log.Printf("⚠️  Error getting cryptogram: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Normalize answer (uppercase, trim whitespace)
	normalizedAnswer := strings.ToUpper(strings.TrimSpace(req.Answer))

	if normalizedAnswer == crypto.PlainText {
		// Set puzzle solved cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "auth",
			Value:    "puzzle_solved",
			HttpOnly: true,
			Secure:   useHTTPS,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   86400, // 24 hours
			Path:     "/",
		})

		log.Printf("✅ Cryptogram solved from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	} else {
		// Add delay to prevent brute force
		time.Sleep(1 * time.Second)
		log.Printf("⚠️  Failed cryptogram attempt from %s", r.RemoteAddr)
		http.Error(w, "Incorrect answer", http.StatusUnauthorized)
	}
}

func handleCryptogramInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get today's cryptogram
	crypto, err := GetTodaysCryptogram()
	if err != nil {
		log.Printf("⚠️  Error getting cryptogram: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return public info (not the plain text answer!)
	response := map[string]interface{}{
		"date":            crypto.Date,
		"cipher_text":     crypto.CipherText,
		"book_title":      crypto.BookTitle,
		"book_author":     crypto.BookAuthor,
		"book_description": crypto.BookDescription,
		"book_cover":      crypto.BookCover,
		"hint_keywords":   crypto.HintKeywords,
		"hint_numbers":    crypto.HintNumbers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleLocation(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	if !isAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "POST":
		// Store new location
		var loc types.Location
		if err := json.NewDecoder(r.Body).Decode(&loc); err != nil {
			http.Error(w, "Invalid location data", http.StatusBadRequest)
			return
		}

		loc.Timestamp = time.Now()
		loc.UserAgent = r.UserAgent()

		// Store in memory cache
		locationMutex.Lock()
		locations[loc.DeviceID] = loc
		locationMutex.Unlock()

		// Persist to DynamoDB (appends to existing data, never deletes)
		if useDynamoDB {
			go saveLocationToDynamoDB(loc)
		}

		log.Printf("📍 Location updated: %s at (%.6f, %.6f) ±%.0fm",
			loc.DeviceID, loc.Latitude, loc.Longitude, loc.Accuracy)

		// Fetch nearby businesses from Google Maps
		go func() {
			businesses, err := fetchNearbyBusinesses(loc.Latitude, loc.Longitude)
			if err != nil {
				log.Printf("⚠️  Error fetching businesses: %v", err)
				// Still update context even if business fetch fails
				updateLastInteractionContext(
					"location_share",
					[]string{},
					loc.DeviceID,
					fmt.Sprintf("%.6f,%.6f", loc.Latitude, loc.Longitude),
					loc.Latitude,
					loc.Longitude,
					[]types.Business{},
					"",
				)
				return
			}

			if len(businesses) > 0 {
				currentBusinessesMutex.Lock()
				currentBusinesses = businesses
				currentBusinessesMutex.Unlock()

				businessNames := make([]string, len(businesses))
				for i, b := range businesses {
					businessNames[i] = b.Name
				}
				log.Printf("🏢 Updated current businesses: %v", businessNames)

				// Update last interaction context with location and business data
				// Extract keywords from location and businesses
				locationName := fmt.Sprintf("%.6f,%.6f", loc.Latitude, loc.Longitude)
				if len(businesses) > 0 && businesses[0].Address != "" {
					locationName = businesses[0].Address
				}

				keywords := extractLocationKeywords(locationName, businesses)
				updateLastInteractionContext(
					"location_share",
					keywords,
					loc.DeviceID,
					locationName,
					loc.Latitude,
					loc.Longitude,
					businesses,
					"",
				)
			}
		}()

		json.NewEncoder(w).Encode(map[string]bool{"success": true})

	case "GET":
		// Return all recent locations
		locationMutex.RLock()
		defer locationMutex.RUnlock()

		json.NewEncoder(w).Encode(locations)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleErrorLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "POST":
		// POST doesn't require auth (for error-generator to send logs)
		// Store new error log
		var errorLog types.ErrorLog
		if err := json.NewDecoder(r.Body).Decode(&errorLog); err != nil {
			http.Error(w, "Invalid error log data", http.StatusBadRequest)
			return
		}

		errorLog.Timestamp = time.Now()
		errorLog.ID = fmt.Sprintf("%d", errorLog.Timestamp.UnixNano())

		// Generate retrievable URL for this error log with ID and timestamp
		baseURL := os.Getenv("BASE_URL")
		if baseURL == "" {
			baseURL = "https://notspies.org" // Default to production URL
		}
		// URL format: /api/errorlogs/{id}/{timestamp}
		// Timestamp is URL-encoded to handle special characters
		timestampStr := url.QueryEscape(errorLog.Timestamp.Format(time.RFC3339Nano))
		errorLog.URL = fmt.Sprintf("%s/api/errorlogs/%s/%s", baseURL, errorLog.ID, timestampStr)

		// Backward compatibility: if gif_urls is provided but gif_url is not, use first GIF
		if errorLog.GifURL == "" && len(errorLog.GifURLs) > 0 {
			errorLog.GifURL = errorLog.GifURLs[0]
		}
		// Forward compatibility: if gif_url is provided but gif_urls is not, create array with single GIF
		if len(errorLog.GifURLs) == 0 && errorLog.GifURL != "" {
			errorLog.GifURLs = []string{errorLog.GifURL}
		}

		// Attach pending user experience note and keywords from Twilio SMS if available
		userExperienceNoteMutex.Lock()
		var userKeywords []string
		if pendingUserExperienceNote != "" {
			errorLog.UserExperienceNote = pendingUserExperienceNote
			errorLog.UserNoteKeywords = pendingUserNoteKeywords
			userKeywords = pendingUserNoteKeywords
			log.Printf("💬 Attached user experience note: %s", pendingUserExperienceNote)
			if len(pendingUserNoteKeywords) > 0 {
				log.Printf("🔑 Extracted keywords: %v", pendingUserNoteKeywords)
			}
			pendingUserExperienceNote = "" // Clear after attaching
			pendingUserNoteKeywords = nil
		}
		userExperienceNoteMutex.Unlock()

		// Get current nearby businesses from Google Maps
		currentBusinessesMutex.RLock()
		businesses := make([]types.Business, len(currentBusinesses))
		copy(businesses, currentBusinesses)
		currentBusinessesMutex.RUnlock()

		// Store business names in error log
		if len(businesses) > 0 {
			businessNames := make([]string, len(businesses))
			for i, b := range businesses {
				businessNames[i] = b.Name
			}
			errorLog.NearbyBusinesses = businessNames
			log.Printf("🏢 Associated with %d nearby businesses from Google Maps", len(businesses))
		}

		// Get current location to search for commercial real estate
		locationMutex.RLock()
		var currentLocation *types.Location
		for _, loc := range locations {
			// Use the most recent location
			if currentLocation == nil || loc.Timestamp.After(currentLocation.Timestamp) {
				locCopy := loc
				currentLocation = &locCopy
			}
		}
		locationMutex.RUnlock()

		// Search for commercial real estate near current location (asynchronously)
		if currentLocation != nil {
			go func(lat, lng float64, keywords []string) {
				properties, governingBodies, queryLat, queryLng, err := searchCommercialRealEstate(lat, lng, keywords)
				if err != nil {
					log.Printf("⚠️  Error searching commercial real estate: %v", err)
					return
				}

				if len(properties) > 0 || len(governingBodies) > 0 {
					locationName := fmt.Sprintf("Area-%.4f-%.4f", queryLat, queryLng)

					commercialRealEstateRecord := types.CommercialRealEstate{
						ID:              fmt.Sprintf("%s-%d", locationName, time.Now().UnixNano()),
						LocationName:    locationName,
						QueryLat:        queryLat,
						QueryLng:        queryLng,
						Properties:      properties,
						GoverningBodies: governingBodies,
						Timestamp:       time.Now(),
					}

					// Cache the commercial real estate
					commercialRealEstateCacheMutex.Lock()
					commercialRealEstateCache[locationName] = commercialRealEstateRecord
					commercialRealEstateCacheMutex.Unlock()

					saveCommercialRealEstateToDynamoDB(commercialRealEstateRecord)

					// Log the results with details
					for _, prop := range properties {
						contactInfo := ""
						if prop.ContactInfo != nil {
							if phone, ok := prop.ContactInfo["phone"].(string); ok && phone != "" {
								contactInfo += fmt.Sprintf(" | Phone: %s", phone)
							}
							if email, ok := prop.ContactInfo["email"].(string); ok && email != "" {
								contactInfo += fmt.Sprintf(" | Email: %s", email)
							}
						}
						log.Printf("🏢 %s - %s (%s) - %s%s", prop.Address, prop.PropertyType, prop.Status, prop.Description, contactInfo)
					}

					// Log governing bodies
					for _, body := range governingBodies {
						log.Printf("🏛️  %s (%s) - %s - %s", body.Name, body.Type, body.Jurisdiction, body.Contact)
					}
				}
			}(currentLocation.Latitude, currentLocation.Longitude, userKeywords)
		}

		// Attach anonymous tips from pending queue
		pendingTipMutex.Lock()
		if len(pendingTipQueue) > 0 {
			// Take up to 3 tips from the queue
			numTips := len(pendingTipQueue)
			if numTips > 3 {
				numTips = 3
			}
			errorLog.AnonymousTips = make([]string, numTips)
			copy(errorLog.AnonymousTips, pendingTipQueue[:numTips])

			// Remove attached tips from queue
			pendingTipQueue = pendingTipQueue[numTips:]

			log.Printf("📝 Attached %d anonymous tips to error log", numTips)
		}
		pendingTipMutex.Unlock()

		// Attach seed interaction traceability - link this error log back to the last user interaction
		seedContext := getLastInteractionContext()
		if seedContext != nil {
			errorLog.SeedInteractionType = seedContext.InteractionType
			errorLog.SeedInteractionTimestamp = seedContext.Timestamp
			errorLog.SeedInteractionID = seedContext.SourceID
			errorLog.SeedKeywords = seedContext.Keywords
			log.Printf("🔗 Linked error log to seed interaction: type=%s, id=%s, keywords=%v",
				seedContext.InteractionType, seedContext.SourceID, seedContext.Keywords)
		}

		// Store in memory cache
		errorLogMutex.Lock()
		// Prepend new error to beginning (most recent first)
		errorLogs = append([]types.ErrorLog{errorLog}, errorLogs...)
		// Keep only last 50 errors in memory
		if len(errorLogs) > 50 {
			errorLogs = errorLogs[:50]
		}
		errorLogMutex.Unlock()

		// Persist to DynamoDB (appends to existing data, never deletes)
		if useDynamoDB {
			go saveErrorLogToDynamoDB(errorLog)
		}

		log.Printf("📝 Error logged: %s", errorLog.Message)

		json.NewEncoder(w).Encode(map[string]bool{"success": true})

	case "GET":
		// GET requires at least puzzle access (viewing logs in UI)
		if !hasPuzzleAccess(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Return recent error logs (limit to 30 most recent)
		errorLogMutex.RLock()
		defer errorLogMutex.RUnlock()

		// Get the 30 most recent error logs
		recentLogs := errorLogs
		if len(recentLogs) > 30 {
			recentLogs = recentLogs[:30]
		}

		// If user only has puzzle access (not full auth), hide location-specific data
		hasFullAccess := isAuthenticated(r)
		if !hasFullAccess {
			// Create sanitized copy without location data and user notes
			sanitized := make([]types.ErrorLog, len(recentLogs))
			for i, log := range recentLogs {
				sanitized[i] = log
				sanitized[i].NearbyBusinesses = nil         // Hide nearby businesses (location-specific)
				sanitized[i].UserExperienceNote = ""        // Hide user experience notes
				sanitized[i].UserNoteKeywords = nil         // Hide user note keywords
			}
			json.NewEncoder(w).Encode(sanitized)
		} else {
			json.NewEncoder(w).Encode(recentLogs)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleErrorLogByID retrieves a single error log by its ID
func handleErrorLogByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID and optional timestamp from URL path
	// Format: /api/errorlogs/{id} or /api/errorlogs/{id}/{timestamp}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}
	errorLogID := pathParts[3]

	if errorLogID == "" {
		http.Error(w, "Error log ID is required", http.StatusBadRequest)
		return
	}

	// Check if timestamp is provided (for new URL format)
	var timestampStr string
	hasTimestamp := len(pathParts) >= 5 && pathParts[4] != ""
	if hasTimestamp {
		// URL-decode the timestamp
		timestampStr, _ = url.QueryUnescape(pathParts[4])
	}

	// Check if this is a browser request (Accept header contains text/html)
	isBrowserRequest := strings.Contains(r.Header.Get("Accept"), "text/html")

	// First check in-memory cache
	errorLogMutex.RLock()
	for _, log := range errorLogs {
		if log.ID == errorLogID {
			errorLogMutex.RUnlock()

			// Determine access level and sanitize if needed
			hasFullAccess := isAuthenticated(r)
			hasPuzzleAccessOnly := !hasFullAccess && hasPuzzleAccess(r)

			if !hasFullAccess && !hasPuzzleAccessOnly {
				// If browser request, show authentication page
				if isBrowserRequest {
					serveTurnstileAuthPage(w, r, errorLogID)
					return
				}
				// Otherwise return JSON error for API requests
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Return JSON response for authenticated requests
			w.Header().Set("Content-Type", "application/json")

			// If user only has puzzle access, sanitize the log
			if hasPuzzleAccessOnly {
				sanitizedLog := log
				sanitizedLog.NearbyBusinesses = nil
				sanitizedLog.UserExperienceNote = ""
				sanitizedLog.UserNoteKeywords = nil
				json.NewEncoder(w).Encode(sanitizedLog)
			} else {
				json.NewEncoder(w).Encode(log)
			}
			return
		}
	}
	errorLogMutex.RUnlock()

	// Not found in memory, check DynamoDB if enabled
	if useDynamoDB {
		ctx := context.Background()
		var errorLog types.ErrorLog
		var err error

		if hasTimestamp {
			// Fast path: Use GetItem with both id and timestamp (composite key)
			result, getErr := dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
				TableName: aws.String(errorLogsTableName),
				Key: map[string]dynamodbtypes.AttributeValue{
					"id":        &dynamodbtypes.AttributeValueMemberS{Value: errorLogID},
					"timestamp": &dynamodbtypes.AttributeValueMemberS{Value: timestampStr},
				},
			})

			if getErr != nil {
				log.Printf("❌ Failed to retrieve error log from DynamoDB: %v", getErr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if result.Item == nil {
				http.Error(w, "Error log not found", http.StatusNotFound)
				return
			}

			err = attributevalue.UnmarshalMap(result.Item, &errorLog)
		} else {
			// Fallback: Query by id (partition key) - requires scanning through timestamps
			queryResult, queryErr := dynamoClient.Query(ctx, &dynamodb.QueryInput{
				TableName:              aws.String(errorLogsTableName),
				KeyConditionExpression: aws.String("id = :id"),
				ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
					":id": &dynamodbtypes.AttributeValueMemberS{Value: errorLogID},
				},
				Limit: aws.Int32(1),
			})

			if queryErr != nil {
				log.Printf("❌ Failed to query error log from DynamoDB: %v", queryErr)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if len(queryResult.Items) == 0 {
				http.Error(w, "Error log not found", http.StatusNotFound)
				return
			}

			err = attributevalue.UnmarshalMap(queryResult.Items[0], &errorLog)
		}
		if err != nil {
			log.Printf("❌ Failed to unmarshal error log: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Determine access level and sanitize if needed
		hasFullAccess := isAuthenticated(r)
		hasPuzzleAccessOnly := !hasFullAccess && hasPuzzleAccess(r)

		if !hasFullAccess && !hasPuzzleAccessOnly {
			// If browser request, show authentication page
			if isBrowserRequest {
				serveTurnstileAuthPage(w, r, errorLogID)
				return
			}
			// Otherwise return JSON error for API requests
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Return JSON response for authenticated requests
		w.Header().Set("Content-Type", "application/json")

		// If user only has puzzle access, sanitize the log
		if hasPuzzleAccessOnly {
			errorLog.NearbyBusinesses = nil
			errorLog.UserExperienceNote = ""
			errorLog.UserNoteKeywords = nil
		}

		json.NewEncoder(w).Encode(errorLog)
		return
	}

	// Not found in memory and DynamoDB not enabled
	http.Error(w, "Error log not found", http.StatusNotFound)
}

func handleBusinesses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// No auth required - error-generator needs to access this
	currentBusinessesMutex.RLock()
	defer currentBusinessesMutex.RUnlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"businesses": currentBusinesses,
		"count":      len(currentBusinesses),
	})
}

func handleCommercialContext(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// No auth required - error-generator needs to access this
	commercialRealEstateCacheMutex.RLock()
	defer commercialRealEstateCacheMutex.RUnlock()

	// Flatten all commercial properties and governing bodies from cache
	allProperties := make([]types.CommercialPropertyDetails, 0)
	allGoverningBodies := make([]types.GoverningBody, 0)
	for _, record := range commercialRealEstateCache {
		allProperties = append(allProperties, record.Properties...)
		allGoverningBodies = append(allGoverningBodies, record.GoverningBodies...)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"properties":       allProperties,
		"governing_bodies": allGoverningBodies,
	})
}

func handlePendingKeywords(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// No auth required - error-generator needs to access this
	userExperienceNoteMutex.RLock()
	defer userExperienceNoteMutex.RUnlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"keywords": pendingUserNoteKeywords,
		"note":     pendingUserExperienceNote,
	})
}

// handleLastInteractionContext exposes the last user interaction context
// This allows error-generator to fetch the "seed event" that should influence all content generation
func handleLastInteractionContext(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// No auth required - error-generator needs to access this
	context := getLastInteractionContext()

	if context == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"has_context": false,
			"message":     "No user interactions yet",
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"has_context":      true,
		"interaction_type": context.InteractionType,
		"timestamp":        context.Timestamp,
		"keywords":         context.Keywords,
		"location_name":    context.LocationName,
		"latitude":         context.Latitude,
		"longitude":        context.Longitude,
		"business_names":   context.BusinessNames,
		"raw_content":      context.RawContent,
		"source_id":        context.SourceID,
	})
}

func handleCommercialRealEstate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Requires auth to view
	if !isAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	commercialRealEstateCacheMutex.RLock()
	defer commercialRealEstateCacheMutex.RUnlock()

	json.NewEncoder(w).Encode(commercialRealEstateCache)
}

func handleTwilioWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data (Twilio sends application/x-www-form-urlencoded)
	if err := r.ParseForm(); err != nil {
		log.Printf("⚠️  Error parsing Twilio webhook form: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	messageBody := r.FormValue("Body")
	messageFrom := r.FormValue("From")
	messageSid := r.FormValue("MessageSid")

	if messageBody == "" {
		log.Printf("⚠️  Received empty message body from Twilio")
		http.Error(w, "Empty message body", http.StatusBadRequest)
		return
	}

	// Extract keywords from user note for satirical purposes
	keywords := extractUserKeywords(messageBody)

	// Store the message and keywords as pending user experience note
	userExperienceNoteMutex.Lock()
	pendingUserExperienceNote = messageBody
	pendingUserNoteKeywords = keywords
	userExperienceNoteMutex.Unlock()

	log.Printf("📱 Received SMS from %s (SID: %s): %s", messageFrom, messageSid, messageBody)
	log.Printf("💬 Stored user experience note, will attach to next error log")
	if len(keywords) > 0 {
		log.Printf("🔑 Extracted keywords for satirical prompts: %v", keywords)
	}

	// Update last interaction context
	updateLastInteractionContext(
		"user_note",
		keywords,
		messageSid,
		"",
		0,
		0,
		[]types.Business{},
		messageBody,
	)

	// Respond with TwiML (Twilio expects XML response)
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><Response></Response>`)
}

// handleTips handles anonymous tip submissions (POST) and retrieval (GET)
func handleTips(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "POST":
		// Submit new anonymous tip
		var req struct {
			TipContent string `json:"tip_content"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate content
		if err := ValidateTipContent(req.TipContent, tipMaxLength); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "error",
				"reason": err.Error(),
			})
			return
		}

		// Generate anonymous user ID
		userHash, encryptedMetadata, err := identityManager.GenerateAnonymousID(r)
		if err != nil {
			http.Error(w, "Failed to generate user ID", http.StatusInternalServerError)
			return
		}

		// Check if user is banned
		if banned, reason, expiresAt := banManager.IsUserBanned(userHash); banned {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     "banned",
				"reason":     reason,
				"expires_at": expiresAt.Format(time.RFC3339),
			})
			return
		}

		// Check rate limit
		allowed, remaining, resetTime := rateLimiter.CheckAndRecordSubmission(userHash)
		if !allowed {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     "rate_limited",
				"reason":     "Too many submissions. Please try again later.",
				"reset_time": resetTime.Format(time.RFC3339),
			})
			return
		}

		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		// Moderate content
		moderationResult, err := contentModerator.ModerateTip(req.TipContent)
		if err != nil {
			log.Printf("⚠️  Moderation error: %v", err)
			http.Error(w, "Moderation failed", http.StatusInternalServerError)
			return
		}

		// Reject if flagged
		if moderationResult.Status == "rejected" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "rejected",
				"reason": moderationResult.Reason,
			})
			return
		}

		// Create tip record
		tip := types.AnonymousTip{
			ID:               fmt.Sprintf("%d", time.Now().UnixNano()),
			TipContent:       req.TipContent,
			ModeratedContent: moderationResult.ModeratedText,
			UserHash:         userHash,
			UserMetadata:     encryptedMetadata,
			ModerationStatus: moderationResult.Status,
			ModerationReason: moderationResult.Reason,
			Keywords:         extractUserKeywords(moderationResult.ModeratedText),
			Timestamp:        time.Now(),
			IPAddress:        getClientIP(r),
		}

		// Store in memory cache
		anonymousTipsMutex.Lock()
		anonymousTips = append(anonymousTips, tip)
		// Keep only last 100 tips in memory
		if len(anonymousTips) > 100 {
			anonymousTips = anonymousTips[len(anonymousTips)-100:]
		}
		anonymousTipsMutex.Unlock()

		// Add to pending tip queue
		pendingTipMutex.Lock()
		pendingTipQueue = append(pendingTipQueue, tip.ID)
		pendingTipMutex.Unlock()

		// Persist to DynamoDB
		if useDynamoDB {
			go saveTipToDynamoDB(tip)
		}

		log.Printf("📝 Anonymous tip submitted: %s (status: %s, user: %s)", tip.ID, tip.ModerationStatus, userHash)

		// Update last interaction context
		updateLastInteractionContext(
			"tip_submission",
			tip.Keywords,
			tip.ID,
			"",
			0,
			0,
			[]types.Business{},
			moderationResult.ModeratedText,
		)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "success",
			"tip_id":    tip.ID,
			"user_hash": userHash,
			"moderated": moderationResult.Status == "redacted",
			"reason":    moderationResult.Reason,
		})

	case "GET":
		// Get recent approved tips
		anonymousTipsMutex.RLock()
		defer anonymousTipsMutex.RUnlock()

		// Filter for approved tips only
		approvedTips := []types.AnonymousTip{}
		for _, tip := range anonymousTips {
			if tip.ModerationStatus == "approved" || tip.ModerationStatus == "redacted" {
				approvedTips = append(approvedTips, tip)
			}
		}

		// Return most recent 20 tips
		if len(approvedTips) > 20 {
			approvedTips = approvedTips[len(approvedTips)-20:]
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"tips":  approvedTips,
			"count": len(approvedTips),
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTipByID retrieves a specific tip by ID
func handleTipByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract tip ID from URL path
	tipID := strings.TrimPrefix(r.URL.Path, "/api/tips/")
	if tipID == "" {
		http.Error(w, "Tip ID required", http.StatusBadRequest)
		return
	}

	// Search in memory cache
	anonymousTipsMutex.RLock()
	defer anonymousTipsMutex.RUnlock()

	for _, tip := range anonymousTips {
		if tip.ID == tipID {
			json.NewEncoder(w).Encode(tip)
			return
		}
	}

	// If not in cache and DynamoDB is available, try to fetch from there
	if useDynamoDB {
		tip, err := getTipFromDynamoDB(tipID)
		if err == nil && tip != nil {
			json.NewEncoder(w).Encode(tip)
			return
		}
	}

	http.Error(w, "Tip not found", http.StatusNotFound)
}

func isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("auth")
	if err != nil {
		return false
	}
	return cookie.Value == "authenticated"
}

// hasPuzzleAccess checks if user has solved the cryptogram puzzle (partial access to error logs)
func hasPuzzleAccess(r *http.Request) bool {
	cookie, err := r.Cookie("auth")
	if err != nil {
		return false
	}
	return cookie.Value == "puzzle_solved" || cookie.Value == "authenticated"
}

// serveTurnstileAuthPage serves an authentication page with Cloudflare Turnstile
func serveTurnstileAuthPage(w http.ResponseWriter, r *http.Request, errorLogID string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusUnauthorized)

	turnstileSiteKey := os.Getenv("TURNSTILE_SITE_KEY")
	if turnstileSiteKey == "" {
		// Use Cloudflare's test site key that forces interactive challenge
		turnstileSiteKey = "2x00000000000000000000AB"
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<!--
═══════════════════════════════════════════════════════════════════════════════
  🕵️  CLASSIFIED DOCUMENT - FOR YOUR EYES ONLY  🕵️
═══════════════════════════════════════════════════════════════════════════════

    ██████╗ ███████╗ ██████╗██████╗ ███████╗████████╗     ██████╗ ██╗   ██╗███████╗███████╗████████╗
    ██╔══██╗██╔════╝██╔════╝██╔══██╗██╔════╝╚══██╔══╝    ██╔═══██╗██║   ██║██╔════╝██╔════╝╚══██╔══╝
    ███████║█████╗  ██║     ██████╔╝█████╗     ██║       ██║   ██║██║   ██║█████╗  ███████╗   ██║
    ██╔══██║██╔══╝  ██║     ██╔══██╗██╔══╝     ██║       ██║▄▄ ██║██║   ██║██╔══╝  ╚════██║   ██║
    ██║  ██║███████╗╚██████╗██║  ██║███████╗   ██║       ╚██████╔╝╚██████╔╝███████╗███████║   ██║
    ╚═╝  ╚═╝╚══════╝ ╚═════╝╚═╝  ╚═╝╚══════╝   ╚═╝        ╚══▀▀═╝  ╚═════╝ ╚══════╝╚══════╝   ╚═╝

                    🔒 ENCRYPTED MESSAGE FOLLOWS 🔒

    You've discovered a hidden layer of surveillance.

    PUZZLE: Decode the following cipher to reveal the secret code word.

    ROT13 Challenge:
    ┌─────────────────────────────────────────────────────────────┐
    │  PBATENGHYNGVBAF LBH SBHAQ GUR FRPERG                      │
    │  GUR PBQR JBEQ VF: QRPELCGVBA                               │
    │  RAGRE GUVF NG GUR YBTVA CEBZCG GB HAYBPX FCRPVNY NPPRFF   │
    └─────────────────────────────────────────────────────────────┘

    Hint: Caesar's favorite shift is 13 positions...

    ⚠️  WARNING: This document is classified. Unauthorized access is monitored.

    SURVEILLANCE NETWORK STATUS:
    ┌──────────────────────────────────────────┐
    │  [████████████████████████████] 100%%    │
    │  All systems operational                 │
    │  Error Log ID: %s                        │
    │  Access Level: RESTRICTED                │
    │  Your IP: [REDACTED]                     │
    │  Geolocation: [TRACKING...]              │
    └──────────────────────────────────────────┘

    Did you really think this was just an error log?
    Welcome to the panopticon. You're always being watched. 👁️

    P.S. Check the JavaScript console for additional easter eggs...

═══════════════════════════════════════════════════════════════════════════════
-->
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>🕵️ Authentication Required</title>
    <script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }

        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }

        .container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            max-width: 500px;
            width: 100%%;
            padding: 40px;
            text-align: center;
        }

        .spy-icon {
            font-size: 72px;
            margin-bottom: 20px;
            animation: pulse 2s ease-in-out infinite;
        }

        @keyframes spin {
            from { transform: rotate(0deg); }
            to { transform: rotate(360deg); }
        }

        @keyframes pulse {
            0%%, 100%% { transform: scale(1); }
            50%% { transform: scale(1.1); }
        }

        @keyframes spin {
            0%% { transform: rotate(0deg); }
            100%% { transform: rotate(360deg); }
        }

        h1 {
            font-size: 28px;
            color: #1a1a1a;
            margin-bottom: 12px;
            font-weight: 800;
        }

        .subtitle {
            font-size: 16px;
            color: #666;
            margin-bottom: 30px;
            line-height: 1.6;
        }

        .challenge-text {
            font-size: 20px;
            color: #667eea;
            font-weight: 700;
            margin-bottom: 30px;
            padding: 15px;
            background: linear-gradient(135deg, rgba(102, 126, 234, 0.1) 0%%, rgba(118, 75, 162, 0.15) 100%%);
            border-radius: 12px;
            border: 2px solid rgba(102, 126, 234, 0.3);
        }

        .turnstile-container {
            display: flex;
            justify-content: center;
            margin: 30px 0;
        }

        .error-id {
            font-family: 'Courier New', monospace;
            font-size: 12px;
            color: #999;
            margin-top: 30px;
            padding: 10px;
            background: #f5f5f5;
            border-radius: 6px;
        }

        #error-message {
            display: none;
            background: #fee;
            color: #c33;
            padding: 12px;
            border: 2px solid #c33;
            border-radius: 8px;
            margin-top: 20px;
        }

        .loading {
            display: none;
            margin-top: 20px;
            color: #667eea;
            font-weight: 600;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="spy-icon">🕵️</div>
        <h1>Authentication Required</h1>
        <p class="subtitle">This error log is classified. You must verify your identity to proceed.</p>

        <div class="challenge-text">
            🔐 PROVE YOU ARE NOT A SPY
        </div>

        <div class="turnstile-container">
            <div class="cf-turnstile"
                 data-sitekey="%s"
                 data-callback="onTurnstileSuccess"
                 data-theme="light">
            </div>
        </div>

        <div class="loading" id="loading">Verifying your identity...</div>
        <div id="error-message"></div>

        <div class="error-id">
            Requested Error Log ID: %s
        </div>
    </div>

    <script>
        function onTurnstileSuccess(token) {
            document.getElementById('loading').style.display = 'block';

            // Send token to backend for verification
            fetch('/api/verify-turnstile', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    token: token,
                    error_log_id: '%s'
                })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    // Redirect to the error log
                    window.location.href = '/api/errorlogs/%s';
                } else {
                    document.getElementById('loading').style.display = 'none';
                    document.getElementById('error-message').style.display = 'block';
                    document.getElementById('error-message').textContent = 'Verification failed: ' + (data.error || 'Unknown error');
                }
            })
            .catch(error => {
                document.getElementById('loading').style.display = 'none';
                document.getElementById('error-message').style.display = 'block';
                document.getElementById('error-message').textContent = 'Network error: ' + error.message;
            });
        }

        // Console Easter Egg
        console.log('%%c🕵️ SECRET AGENT CONSOLE 🕵️', 'font-size: 24px; font-weight: bold; color: #667eea;');
        console.log('%%c═══════════════════════════════════════════════════════════════', 'color: #764ba2;');
        console.log('%%cWelcome to the surveillance matrix, agent.', 'font-size: 14px; color: #333;');
        console.log('%%c', 'font-size: 12px;');
        console.log('%%cYour access level: RESTRICTED', 'color: red; font-weight: bold;');
        console.log('%%cError Log ID: %s', 'color: #666;');
        console.log('%%c', 'font-size: 12px;');
        console.log('%%c🎮 INTERACTIVE CHALLENGE:', 'font-size: 14px; font-weight: bold; color: #667eea;');
        console.log('%%cType the following command to unlock a secret message:', 'color: #333;');
        console.log('%%c    window.revealSecret()', 'background: #f0f0f0; padding: 8px; font-family: monospace; color: #d63384;');
        console.log('%%c═══════════════════════════════════════════════════════════════', 'color: #764ba2;');

        window.revealSecret = function() {
            console.clear();
            console.log('%%c', 'font-size: 20px;');
            console.log('%%c    ███████╗███████╗ ██████╗██████╗ ███████╗████████╗', 'color: #667eea; font-family: monospace;');
            console.log('%%c    ██╔════╝██╔════╝██╔════╝██╔══██╗██╔════╝╚══██╔══╝', 'color: #667eea; font-family: monospace;');
            console.log('%%c    ███████╗█████╗  ██║     ██████╔╝█████╗     ██║   ', 'color: #667eea; font-family: monospace;');
            console.log('%%c    ╚════██║██╔══╝  ██║     ██╔══██╗██╔══╝     ██║   ', 'color: #667eea; font-family: monospace;');
            console.log('%%c    ███████║███████╗╚██████╗██║  ██║███████╗   ██║   ', 'color: #667eea; font-family: monospace;');
            console.log('%%c    ╚══════╝╚══════╝ ╚═════╝╚═╝  ╚═╝╚══════╝   ╚═╝   ', 'color: #667eea; font-family: monospace;');
            console.log('%%c', 'font-size: 20px;');
            console.log('%%c🎉 CONGRATULATIONS! You found the hidden console puzzle!', 'font-size: 16px; color: green; font-weight: bold;');
            console.log('%%c', 'font-size: 12px;');
            console.log('%%c📜 SECRET MESSAGE:', 'font-size: 14px; font-weight: bold; color: #764ba2;');
            console.log('%%c"The watchers are themselves watched. Every system of surveillance', 'color: #333; font-style: italic;');
            console.log('%%ccontains within it the seeds of its own transparency. You have', 'color: #333; font-style: italic;');
            console.log('%%cdemonstrated this by finding what was hidden in plain sight."', 'color: #333; font-style: italic;');
            console.log('%%c', 'font-size: 12px;');
            console.log('%%c🔑 ACHIEVEMENT UNLOCKED: "Console Spelunker"', 'background: gold; color: black; padding: 8px; font-weight: bold;');
            console.log('%%c', 'font-size: 12px;');
            console.log('%%c💡 Bonus Hint: Check the HTML source comments for another puzzle...', 'color: #666; font-style: italic;');
            console.log('%%c', 'font-size: 12px;');
            console.log('%%c👁️ Remember: You are not paranoid if they really are watching.', 'color: red; font-weight: bold;');
            return '🕵️ Secret revealed! Welcome to the inner circle, agent.';
        };

        // Hidden function for extra curious developers
        window.surveillance = {
            status: 'ACTIVE',
            targets: '[REDACTED]',
            operators: Math.floor(Math.random() * 1000) + 100,
            lastUpdate: new Date().toISOString(),
            message: '👁️ The panopticon sees all. Type window.surveillance.bypass() to attempt escape.'
        };

        window.surveillance.bypass = function() {
            console.log('%%c⚠️  ACCESS DENIED ⚠️', 'font-size: 20px; color: red; font-weight: bold;');
            console.log('%%cNice try, agent. But you cannot simply walk out of the matrix.', 'color: red;');
            console.log('%%cYour attempt has been logged and reported to the authorities.', 'color: red;');
            console.log('%%c...', 'color: gray;');
            setTimeout(() => {
                console.log('%%cJust kidding! 😄 There is no escape, but we appreciate the effort.', 'color: green; font-weight: bold;');
            }, 2000);
            return '🚫 Escape impossible. Welcome to Hotel California.';
        };
    </script>
</body>
</html>`, turnstileSiteKey, errorLogID, errorLogID, errorLogID, errorLogID)

	w.Write([]byte(html))
}

func cleanupOldLocations() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		locationMutex.Lock()
		now := time.Now()
		for id, loc := range locations {
			if now.Sub(loc.Timestamp) > 24*time.Hour {
				delete(locations, id)
				log.Printf("🗑️  Removed old location: %s", id)
			}
		}
		locationMutex.Unlock()
	}
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("index").Parse(indexHTML))
	stripePublishableKey := os.Getenv("STRIPE_PUBLISHABLE_KEY")
	data := map[string]interface{}{
		"GoogleMapsAPIKey":       googleMapsAPIKey,
		"StripePublishableKey":   stripePublishableKey,
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("❌ Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// generateSelfSignedCert creates a self-signed certificate for local testing
func generateSelfSignedCert(certFile, keyFile string) error {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Create certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // Valid for 1 year

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Location Tracker"},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	// Write certificate to file
	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return err
	}

	// Write private key to file
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}); err != nil {
		return err
	}

	return nil
}

// initializeDynamoDB connects to existing DynamoDB tables (never creates/modifies tables)
func initializeDynamoDB() {
	ctx := context.Background()

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		log.Printf("⚠️  Failed to load AWS config: %v", err)
		return
	}

	// Create DynamoDB client
	dynamoClient = dynamodb.NewFromConfig(cfg)

	// Test connection by describing one of the tables (read-only operation)
	_, err = dynamoClient.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(errorLogsTableName),
	})
	if err != nil {
		log.Printf("⚠️  DynamoDB table not accessible: %v", err)
		return
	}

	useDynamoDB = true
}

// initializeTipSystem initializes the anonymous tip submission system
func initializeTipSystem() {
	// Initialize identity manager with encryption key
	if tipEncryptionKey == "" {
		log.Printf("⚠️  TIP_ENCRYPTION_KEY not set, generating random key (will not persist across restarts)")
		// Generate a random 32-byte key
		randomKey := make([]byte, 32)
		_, err := rand.Read(randomKey)
		if err != nil {
			log.Fatal("❌ Failed to generate encryption key")
		}
		tipEncryptionKey = fmt.Sprintf("%x", randomKey)
	}

	// Convert hex string to bytes (expecting 64 hex chars for 32 bytes)
	keyBytes := make([]byte, 32)
	if len(tipEncryptionKey) == 64 {
		// Hex encoded
		for i := 0; i < 32; i++ {
			fmt.Sscanf(tipEncryptionKey[i*2:i*2+2], "%x", &keyBytes[i])
		}
	} else if len(tipEncryptionKey) == 32 {
		// Raw bytes
		copy(keyBytes, []byte(tipEncryptionKey))
	} else {
		log.Fatal("❌ TIP_ENCRYPTION_KEY must be 32 bytes (64 hex characters or 32 raw bytes)")
	}

	var err error
	identityManager, err = NewUserIdentityManager(keyBytes)
	if err != nil {
		log.Fatalf("❌ Failed to create identity manager: %v", err)
	}

	// Initialize content moderator
	contentModerator = NewContentModerator(openaiAPIKey)
	if openaiAPIKey == "" {
		log.Printf("⚠️  OPENAI_API_KEY not set, content moderation will use pattern matching only")
	}

	// Initialize rate limiter
	rateLimiter = NewRateLimiter(tipRateLimit)

	// Initialize ban manager
	banManager = NewBanManager(dynamoClient, bannedUsersTableName)

	log.Printf("✅ Anonymous tip system initialized")
	log.Printf("📝 Tip rate limit: %d per hour", tipRateLimit)
	log.Printf("📏 Tip max length: %d characters", tipMaxLength)
}

// saveErrorLogToDynamoDB appends error log to DynamoDB (never deletes existing data)
func saveErrorLogToDynamoDB(errorLog types.ErrorLog) {
	ctx := context.Background()

	// Marshal the error log to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(errorLog)
	if err != nil {
		log.Printf("❌ Failed to marshal error log: %v", err)
		return
	}

	// Put item into DynamoDB (appends new record, preserves all existing data)
	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(errorLogsTableName),
		Item:      item,
	})
	if err != nil {
		log.Printf("❌ Failed to save error log to DynamoDB: %v", err)
		return
	}

	log.Printf("💾 Error log saved to DynamoDB: %s", errorLog.ID)
}

// saveLocationToDynamoDB appends location to DynamoDB (never deletes existing data)
func saveLocationToDynamoDB(location types.Location) {
	ctx := context.Background()

	// Marshal the location to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(location)
	if err != nil {
		log.Printf("❌ Failed to marshal location: %v", err)
		return
	}

	// Put item into DynamoDB (appends new record, preserves all existing data)
	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(locationsTableName),
		Item:      item,
	})
	if err != nil {
		log.Printf("❌ Failed to save location to DynamoDB: %v", err)
		return
	}

	log.Printf("💾 Location saved to DynamoDB: %s", location.DeviceID)
}

// loadExistingData loads existing records from DynamoDB on startup (preserves all data)
func loadExistingData() {
	ctx := context.Background()

	// Load error logs from DynamoDB
	log.Printf("📥 Loading error logs from DynamoDB...")
	errorLogsResult, err := dynamoClient.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(errorLogsTableName),
		Limit:     aws.Int32(50), // Load last 50 errors for in-memory cache
	})
	if err != nil {
		log.Printf("⚠️  Failed to load error logs: %v", err)
	} else {
		var loadedErrorLogs []types.ErrorLog
		err = attributevalue.UnmarshalListOfMaps(errorLogsResult.Items, &loadedErrorLogs)
		if err != nil {
			log.Printf("⚠️  Failed to unmarshal error logs: %v", err)
		} else {
			// Sort by timestamp descending (most recent first)
			sort.Slice(loadedErrorLogs, func(i, j int) bool {
				return loadedErrorLogs[i].Timestamp.After(loadedErrorLogs[j].Timestamp)
			})

			// Keep only last 50 in memory cache
			if len(loadedErrorLogs) > 50 {
				loadedErrorLogs = loadedErrorLogs[:50]
			}

			errorLogMutex.Lock()
			errorLogs = loadedErrorLogs
			errorLogMutex.Unlock()

			log.Printf("✅ Loaded %d error logs from DynamoDB into memory", len(loadedErrorLogs))
		}
	}

	// Load locations from last 24 hours
	log.Printf("📥 Loading recent locations from DynamoDB...")
	locationsResult, err := dynamoClient.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(locationsTableName),
	})
	if err != nil {
		log.Printf("⚠️  Failed to load locations: %v", err)
	} else {
		var loadedLocations []types.Location
		err = attributevalue.UnmarshalListOfMaps(locationsResult.Items, &loadedLocations)
		if err != nil {
			log.Printf("⚠️  Failed to unmarshal locations: %v", err)
		} else {
			// Filter to last 24 hours and get most recent per device
			now := time.Now()
			recentLocations := make(map[string]types.Location)

			for _, loc := range loadedLocations {
				if now.Sub(loc.Timestamp) <= 24*time.Hour {
					// Keep the most recent location for each device
					if existing, ok := recentLocations[loc.DeviceID]; !ok || loc.Timestamp.After(existing.Timestamp) {
						recentLocations[loc.DeviceID] = loc
					}
				}
			}

			locationMutex.Lock()
			locations = recentLocations
			locationMutex.Unlock()

			log.Printf("✅ Loaded %d locations from DynamoDB into memory", len(recentLocations))
		}
	}
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>📍 Location Tracker</title>
    <script src="https://js.stripe.com/v3/"></script>
    {{/* Hidden ASCII art puzzle for source code viewers */}}
    <script type="text/plain" style="display:none">
═══════════════════════════════════════════════════════════════════════════════
  🕵️  CLASSIFIED DOCUMENT - FOR YOUR EYES ONLY  🕵️
═══════════════════════════════════════════════════════════════════════════════

    ██████╗ ███████╗ ██████╗██████╗ ███████╗████████╗     ██████╗ ██╗   ██╗███████╗███████╗████████╗
    ██╔══██╗██╔════╝██╔════╝██╔══██╗██╔════╝╚══██╔══╝    ██╔═══██╗██║   ██║██╔════╝██╔════╝╚══██╔══╝
    ███████║█████╗  ██║     ██████╔╝█████╗     ██║       ██║   ██║██║   ██║█████╗  ███████╗   ██║
    ██╔══██║██╔══╝  ██║     ██╔══██╗██╔══╝     ██║       ██║▄▄ ██║██║   ██║██╔══╝  ╚════██║   ██║
    ██║  ██║███████╗╚██████╗██║  ██║███████╗   ██║       ╚██████╔╝╚██████╔╝███████╗███████║   ██║
    ╚═╝  ╚═╝╚══════╝ ╚═════╝╚═╝  ╚═╝╚══════╝   ╚═╝        ╚══▀▀═╝  ╚═════╝ ╚══════╝╚══════╝   ╚═╝

                    🔒 ENCRYPTED MESSAGE FOLLOWS 🔒

    You've discovered a hidden layer of surveillance.

    PUZZLE: Decode the following cipher to reveal the secret code word.

    ROT13 Challenge:
    ┌─────────────────────────────────────────────────────────────┐
    │  PBATENGHYNGVBAF LBH SBHAQ GUR FRPERG                      │
    │  GUR PBQR JBEQ VF: QRPELCGVBA                               │
    │  RAGRE GUVF NG GUR YBTVA CEBZCG GB HAYBPX FCRPVNY NPPRFF   │
    └─────────────────────────────────────────────────────────────┘

    Hint: Caesar's favorite shift is 13 positions...

    ⚠️  WARNING: This document is classified. Unauthorized access is monitored.

    SURVEILLANCE NETWORK STATUS:
    ┌──────────────────────────────────────────┐
    │  [████████████████████████████] 100%     │
    │  All systems operational                 │
    │  Active Monitors: 847                    │
    │  Access Level: PUBLIC (MONITORED)        │
    │  Your IP: [REDACTED]                     │
    │  Geolocation: [TRACKING...]              │
    │  Session Started: [LOGGED]               │
    └──────────────────────────────────────────┘

           🌐 WELCOME TO THE ERROR SURVEILLANCE NETWORK 🌐

    Every error is tracked. Every access is logged. Every click is recorded.

    But who watches the watchers?

    Did you really think this was just a location tracker?
    Welcome to the panopticon. You're always being watched. 👁️

    ┌────────────────────────────────────────────────────────────┐
    │  ASCII ART PUZZLE #1: The Maze                            │
    │                                                            │
    │  ╔═══════════════════════════════════════╗                │
    │  ║ S → → ↓                               ║                │
    │  ║       ↓   ╔═══╗   ╔═══════╗           ║                │
    │  ║   ╔═══╝   ║   ║   ║       ↓           ║                │
    │  ║   ↓       ║   ╚═══╝   ╔═══╝           ║                │
    │  ║   ╚═══════════════════╝   → → → E     ║                │
    │  ╚═══════════════════════════════════════╝                │
    │                                                            │
    │  Can you find the path from S to E?                       │
    │  Hint: The path spells a word when decoded...             │
    └────────────────────────────────────────────────────────────┘

    P.S. Check the JavaScript console for additional easter eggs...
          Type: window.revealSecret()

═══════════════════════════════════════════════════════════════════════════════
    </script>
    <style>
        /* Delphi Design System - Memphis × Swiss × 80s Pop + Terminal Minimalist */

        /* Default Theme: Memphis Design Colors */
        :root {
            --bg-primary: linear-gradient(135deg, #ff6b9d 0%, #48cae4 50%, #06ffa5 100%);
            --bg-container: #ffffff;
            --bg-header: linear-gradient(135deg, #00f0ff 0%, #ff0080 100%);
            --bg-card: #f8f9fa;
            --bg-card-hover: #f8f9fa;

            --text-primary: #000000;
            --text-secondary: #6c757d;
            --text-header: #ffffff;
            --text-accent: #ff0080;

            --border-color: #000000;
            --border-width: 2px;
            --border-radius: 12px;

            --shadow-default: 4px 4px 0px rgba(0, 0, 0, 0.1);
            --shadow-hover: 6px 6px 0px rgba(0, 0, 0, 0.15);
            --shadow-button: 4px 4px 0px rgba(0, 0, 0, 0.2);

            --btn-primary: linear-gradient(135deg, #ff6b9d 0%, #48cae4 100%);
            --btn-share: linear-gradient(135deg, #06ffa5 0%, #ccff00 100%);
            --btn-refresh: linear-gradient(135deg, #a55eea 0%, #b300ff 100%);

            /* Legacy color variables for compatibility */
            --memphis-pink: #ff6b9d;
            --memphis-yellow: #feca57;
            --memphis-blue: #48cae4;
            --memphis-green: #06ffa5;
            --memphis-purple: #a55eea;
            --pop-electric-blue: #00f0ff;
            --pop-hot-pink: #ff0080;
            --pop-lime-green: #ccff00;
            --pop-purple-neon: #b300ff;
            --swiss-black: #000000;
            --swiss-white: #ffffff;
            --swiss-gray-100: #f8f9fa;
            --swiss-gray-500: #6c757d;
        }

        /* Terminal Theme: Minimalist White */
        body[data-theme="terminal"] {
            --bg-primary: #ffffff;
            --bg-container: #fafafa;
            --bg-header: #ffffff;
            --bg-card: #ffffff;
            --bg-card-hover: #f5f5f5;

            --text-primary: #000000;
            --text-secondary: #666666;
            --text-header: #000000;
            --text-accent: #000000;

            --border-color: #e0e0e0;
            --border-width: 1px;
            --border-radius: 4px;

            --shadow-default: none;
            --shadow-hover: 0 2px 4px rgba(0, 0, 0, 0.05);
            --shadow-button: 0 1px 2px rgba(0, 0, 0, 0.1);

            --btn-primary: #000000;
            --btn-share: #000000;
            --btn-refresh: #000000;
        }

        * { box-sizing: border-box; margin: 0; padding: 0; }

        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: var(--bg-primary);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
            transition: background 0.3s ease;
        }

        body[data-theme="terminal"] {
            font-family: 'Courier New', 'Monaco', 'Menlo', monospace;
        }

        .container {
            background: var(--bg-container);
            border-radius: var(--border-radius);
            box-shadow: 8px 8px 0px rgba(0, 0, 0, 0.2), 0 20px 60px rgba(0,0,0,0.3);
            max-width: 600px;
            width: 100%;
            overflow: hidden;
            transition: all 0.3s ease;
        }

        body[data-theme="terminal"] .container {
            box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
            border: var(--border-width) solid var(--border-color);
        }

        .header {
            background: var(--bg-header);
            color: var(--text-header);
            padding: 30px;
            text-align: center;
            border-bottom: var(--border-width) solid var(--border-color);
            position: relative;
            transition: all 0.3s ease;
        }

        body[data-theme="terminal"] .header {
            border-bottom: var(--border-width) solid var(--border-color);
            text-shadow: none;
        }

        .header h1 {
            font-size: 28px;
            margin-bottom: 5px;
            font-weight: 900;
            text-transform: uppercase;
            letter-spacing: 0.1em;
            text-shadow: 3px 3px 0px rgba(0, 0, 0, 0.2);
            transition: all 0.3s ease;
        }

        body[data-theme="terminal"] .header h1 {
            font-size: 18px;
            font-weight: 400;
            text-shadow: none;
            letter-spacing: 0.05em;
        }

        .header p {
            opacity: 0.95;
            font-size: 13px;
            text-transform: uppercase;
            letter-spacing: 0.15em;
            font-family: 'Courier New', monospace;
        }

        body[data-theme="terminal"] .header p {
            font-size: 11px;
            opacity: 0.6;
        }

        /* Theme Toggle Switch */
        .theme-toggle {
            position: absolute;
            top: 15px;
            right: 15px;
            display: flex;
            align-items: center;
            gap: 8px;
            cursor: pointer;
            user-select: none;
        }

        .theme-toggle-label {
            font-size: 10px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            opacity: 0.8;
        }

        .theme-toggle-switch {
            position: relative;
            width: 44px;
            height: 22px;
            background: rgba(0, 0, 0, 0.2);
            border-radius: 11px;
            transition: background 0.3s ease;
            border: 1px solid rgba(0, 0, 0, 0.1);
        }

        .theme-toggle-switch::after {
            content: '';
            position: absolute;
            top: 2px;
            left: 2px;
            width: 16px;
            height: 16px;
            background: white;
            border-radius: 50%;
            transition: transform 0.3s ease;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
        }

        body[data-theme="terminal"] .theme-toggle-switch {
            background: #000000;
        }

        body[data-theme="terminal"] .theme-toggle-switch::after {
            transform: translateX(22px);
        }

        .content { padding: 30px; }

        /* Login Form */
        #login input {
            width: 100%;
            padding: 15px;
            border: 2px solid var(--swiss-black);
            border-radius: 8px;
            font-size: 16px;
            margin-bottom: 15px;
            transition: all 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
            box-shadow: 3px 3px 0px rgba(0, 0, 0, 0.1);
        }

        #login input:focus {
            outline: none;
            border-color: var(--pop-electric-blue);
            box-shadow: 0 0 20px rgba(0, 240, 255, 0.4), 3px 3px 0px rgba(0, 0, 0, 0.1);
        }

        button {
            width: 100%;
            padding: 15px;
            background: var(--btn-primary);
            color: var(--swiss-white);
            border: var(--border-width) solid var(--border-color);
            border-radius: var(--border-radius);
            font-size: 16px;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            cursor: pointer;
            transition: all 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
            box-shadow: var(--shadow-button);
        }

        body[data-theme="terminal"] button {
            background: var(--btn-primary);
            color: #ffffff;
            font-size: 13px;
            padding: 10px 15px;
            letter-spacing: 0.02em;
            font-weight: 500;
        }

        button:hover {
            transform: translateY(-2px);
            box-shadow: 6px 6px 0px rgba(0, 0, 0, 0.2), 0 0 25px rgba(255, 107, 157, 0.4);
        }

        body[data-theme="terminal"] button:hover {
            transform: none;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.15);
            background: #333333;
        }

        button:active { transform: translateY(0px); }

        .error {
            background: #fee;
            color: #c33;
            padding: 12px;
            border: 2px solid #c33;
            border-radius: 8px;
            margin-top: 15px;
            display: none;
        }

        /* Tracker View */
        #tracker { display: none; }

        .actions {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 10px;
            margin-bottom: 20px;
        }

        .btn-share {
            background: linear-gradient(135deg, var(--memphis-green) 0%, var(--pop-lime-green) 100%);
            color: var(--swiss-black);
        }

        .btn-share:hover {
            box-shadow: 6px 6px 0px rgba(0, 0, 0, 0.2), 0 0 25px rgba(6, 255, 165, 0.4);
        }

        .btn-refresh {
            background: linear-gradient(135deg, var(--memphis-purple) 0%, var(--pop-purple-neon) 100%);
        }

        .btn-refresh:hover {
            box-shadow: 6px 6px 0px rgba(0, 0, 0, 0.2), 0 0 25px rgba(179, 0, 255, 0.4);
        }

        .location-card {
            background: var(--swiss-gray-100);
            border: 2px solid var(--swiss-black);
            border-radius: 12px;
            padding: 20px;
            margin-bottom: 20px;
            box-shadow: 4px 4px 0px rgba(0, 0, 0, 0.1);
            transition: all 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
        }

        .location-card:hover {
            transform: translateY(-4px);
            box-shadow: 6px 6px 0px rgba(0, 0, 0, 0.15);
        }

        .location-card h3 {
            color: var(--pop-hot-pink);
            margin-bottom: 15px;
            font-size: 18px;
            font-weight: 800;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        .location-detail {
            display: flex;
            justify-content: space-between;
            padding: 10px 0;
            border-bottom: 1.5px solid rgba(0, 0, 0, 0.1);
        }

        .location-detail:last-child { border-bottom: none; }

        .label {
            color: var(--swiss-gray-500);
            font-weight: 600;
            text-transform: uppercase;
            font-size: 11px;
            letter-spacing: 0.08em;
        }

        .value {
            color: var(--swiss-black);
            font-family: 'Courier New', monospace;
            font-weight: 600;
        }

        .map-link {
            display: inline-block;
            margin-top: 15px;
            padding: 12px 24px;
            background: linear-gradient(135deg, var(--pop-electric-blue) 0%, var(--memphis-blue) 100%);
            color: var(--swiss-black);
            text-decoration: none;
            border: 2px solid var(--swiss-black);
            border-radius: 8px;
            font-size: 13px;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            transition: all 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
            box-shadow: 3px 3px 0px rgba(0, 0, 0, 0.15);
        }

        .map-link:hover {
            transform: translateY(-2px);
            box-shadow: 5px 5px 0px rgba(0, 0, 0, 0.15), 0 0 20px rgba(0, 240, 255, 0.4);
        }

        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: var(--swiss-gray-500);
        }

        .empty-state svg {
            width: 80px;
            height: 80px;
            margin-bottom: 20px;
            opacity: 0.5;
        }

        .status {
            display: inline-block;
            padding: 6px 14px;
            border-radius: 20px;
            font-size: 11px;
            font-weight: 700;
            background: var(--memphis-yellow);
            color: var(--swiss-black);
            text-transform: uppercase;
            letter-spacing: 0.05em;
            border: 1.5px solid var(--swiss-black);
        }

        /* Collapsible Sections */
        .collapsible-header {
            cursor: pointer;
            user-select: none;
            transition: all 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
        }

        .collapsible-header:hover {
            opacity: 0.8;
        }

        .collapsible-toggle {
            display: inline-block;
            margin-left: 8px;
            transition: transform 0.3s ease;
            font-size: 14px;
        }

        .collapsible-toggle.expanded {
            transform: rotate(90deg);
        }

        .collapsible-content {
            max-height: 0;
            overflow: hidden;
            transition: max-height 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275);
        }

        .collapsible-content.expanded {
            max-height: 2000px;
        }

        /* Terminal Theme - Error Log Styling */
        body[data-theme="terminal"] .location-card {
            background: var(--bg-card);
            border: var(--border-width) solid var(--border-color);
            border-radius: var(--border-radius);
            box-shadow: none;
            margin-bottom: 10px;
            padding: 12px 15px;
            font-family: 'Courier New', monospace;
            font-size: 12px;
        }

        body[data-theme="terminal"] .location-card:hover {
            transform: none;
            box-shadow: var(--shadow-hover);
        }

        body[data-theme="terminal"] .location-card h3 {
            color: var(--text-primary);
            font-size: 14px;
            font-weight: 600;
            margin-bottom: 8px;
            letter-spacing: 0;
        }

        body[data-theme="terminal"] h3 {
            color: var(--text-primary);
            font-size: 13px;
            text-transform: none;
            letter-spacing: 0;
            border-bottom: 1px solid var(--border-color);
            padding-bottom: 6px;
            margin-bottom: 12px;
            font-weight: 600;
        }

        /* Terminal nested structure for error details */
        body[data-theme="terminal"] .collapsible-header {
            background: transparent !important;
            border: none !important;
            padding: 6px 0 6px 15px !important;
            margin-bottom: 4px !important;
            box-shadow: none !important;
            position: relative;
        }

        body[data-theme="terminal"] .collapsible-header::before {
            content: '├─';
            position: absolute;
            left: 0;
            color: var(--text-secondary);
            font-family: 'Courier New', monospace;
        }

        body[data-theme="terminal"] .collapsible-header:last-of-type::before {
            content: '└─';
        }

        body[data-theme="terminal"] .collapsible-header strong {
            background: none !important;
            -webkit-background-clip: unset !important;
            -webkit-text-fill-color: unset !important;
            background-clip: unset !important;
            color: var(--text-primary) !important;
            font-size: 11px !important;
            font-weight: 600 !important;
        }

        body[data-theme="terminal"] .collapsible-toggle {
            color: var(--text-secondary);
            font-size: 10px;
        }

        body[data-theme="terminal"] .collapsible-content {
            margin-left: 15px;
            padding-left: 15px;
            border-left: 1px solid var(--border-color);
        }

        /* Hide emojis in terminal theme for cleaner look */
        body[data-theme="terminal"] .collapsible-header span:first-child {
            display: none;
        }

        /* Terminal input styling */
        body[data-theme="terminal"] #login input,
        body[data-theme="terminal"] textarea {
            border: var(--border-width) solid var(--border-color);
            border-radius: var(--border-radius);
            box-shadow: none;
            font-family: 'Courier New', monospace;
            font-size: 12px;
            padding: 10px;
        }

        body[data-theme="terminal"] #login input:focus,
        body[data-theme="terminal"] textarea:focus {
            border-color: #000000;
            box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.1);
        }

        /* Terminal cards - flatten gradients */
        body[data-theme="terminal"] div[style*="linear-gradient"] {
            background: var(--bg-card) !important;
            border: var(--border-width) solid var(--border-color) !important;
        }

        /* Terminal error messages */
        body[data-theme="terminal"] .error {
            background: #ffffff;
            color: #000000;
            border: 1px solid #cccccc;
            font-family: 'Courier New', monospace;
            font-size: 11px;
        }

        /* Terminal GIF/image containers */
        body[data-theme="terminal"] div[style*="rgba(239, 68, 68"] {
            background: var(--bg-card) !important;
            border: var(--border-width) solid var(--border-color) !important;
            box-shadow: none !important;
            padding: 12px !important;
        }

        /* Terminal action buttons */
        body[data-theme="terminal"] .btn-share,
        body[data-theme="terminal"] .btn-refresh {
            background: var(--btn-primary) !important;
            color: #ffffff !important;
        }
    </style>
    {{if .GoogleMapsAPIKey}}<script src="https://maps.googleapis.com/maps/api/js?key={{.GoogleMapsAPIKey}}&libraries=places"></script>{{end}}
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="theme-toggle" onclick="toggleTheme()">
                <span class="theme-toggle-label" id="theme-label">Terminal</span>
                <div class="theme-toggle-switch"></div>
            </div>
            <h1>📍 Location Tracker</h1>
            <p>Educational Personal Security Project</p>
        </div>

        <div class="content">
            <!-- Login View -->
            <div id="login">
                <!-- Error Stories Description -->
                <div style="background: linear-gradient(135deg, rgba(102, 126, 234, 0.1), rgba(118, 75, 162, 0.1)); padding: 20px; border-radius: 12px; margin-bottom: 25px; border-left: 4px solid var(--memphis-purple);">
                    <h2 style="color: var(--pop-hot-pink); margin-bottom: 10px; font-size: 20px; font-weight: 800;">📚 Welcome to The Digital Detective Agency</h2>
                    <p style="color: var(--swiss-gray-500); line-height: 1.6; margin-bottom: 10px;">
                        Behind every system error lies a story—absurd, satirical, and darkly humorous tales of what went wrong.
                        These aren't just logs; they're narratives of digital chaos, complete with GIFs, songs, and AI-generated children's stories.
                    </p>
                    <p style="color: var(--swiss-gray-500); line-height: 1.6; font-weight: 600;">
                        Solve the cryptogram below to unlock access to the error logs (without location data).
                        Or enter the full password for complete access.
                    </p>
                </div>

                <!-- Cryptogram Puzzle -->
                <div style="background: var(--swiss-gray-100); padding: 20px; border-radius: 12px; margin-bottom: 20px; border: 2px solid var(--pop-electric-blue);">
                    <h3 style="color: var(--pop-purple-neon); margin-bottom: 15px; font-size: 16px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em;">🔐 Daily Cryptogram Challenge</h3>

                    <!-- Book Reference Card -->
                    <div id="book-reference" style="background: white; padding: 15px; border-radius: 8px; margin-bottom: 15px; border: 2px solid var(--swiss-black); box-shadow: 3px 3px 0px rgba(0,0,0,0.1);">
                        <div style="display: flex; gap: 15px; align-items: start;">
                            <img id="book-cover" src="" alt="Book cover" style="width: 80px; height: 120px; object-fit: cover; border-radius: 4px; border: 2px solid var(--swiss-black); display: none;">
                            <div style="flex: 1;">
                                <h4 id="book-title" style="color: var(--memphis-purple); margin-bottom: 5px; font-size: 14px; font-weight: 700;"></h4>
                                <p id="book-author" style="color: var(--swiss-gray-500); margin-bottom: 10px; font-size: 12px; font-style: italic;"></p>
                                <p id="book-description" style="color: var(--swiss-black); font-size: 12px; line-height: 1.5;"></p>
                            </div>
                        </div>
                        <div id="book-hints" style="margin-top: 12px; padding-top: 12px; border-top: 2px solid var(--swiss-gray-200);">
                            <p style="color: var(--pop-hot-pink); font-weight: 700; font-size: 12px; margin-bottom: 8px;">📖 HINTS FROM THE BOOK:</p>
                            <div id="hint-keywords" style="font-size: 12px; color: var(--swiss-gray-600); margin-bottom: 5px;"></div>
                            <div id="hint-numbers" style="font-size: 12px; color: var(--swiss-gray-600);"></div>
                        </div>
                    </div>

                    <!-- Cipher Text -->
                    <p id="cipher-text" style="font-family: 'Courier New', monospace; color: var(--swiss-black); font-size: 14px; line-height: 1.8; margin-bottom: 15px; text-align: center; font-weight: 600; background: var(--swiss-white); padding: 15px; border-radius: 8px; border: 2px solid var(--swiss-black);">
                        Loading today's cryptogram...
                    </p>

                    <input type="text" id="cryptogram" placeholder="Enter decoded message" style="width: 100%; padding: 12px; border: 2px solid var(--swiss-black); border-radius: 8px; font-size: 14px; margin-bottom: 10px;">
                    <button onclick="solveCryptogram()" style="width: 100%; background: linear-gradient(135deg, var(--memphis-purple), var(--pop-purple-neon));">🧩 Submit Answer</button>
                    <div class="error" id="cryptogram-error" style="margin-top: 10px;">Incorrect answer. Try again!</div>
                </div>

                <!-- Solid Login (New!) -->
                <div id="solid-login-section" style="background: linear-gradient(135deg, rgba(76, 175, 80, 0.1), rgba(56, 142, 60, 0.1)); padding: 20px; border-radius: 12px; margin-bottom: 20px; border-left: 4px solid #4caf50;">
                    <h3 style="color: #4caf50; margin-bottom: 10px; font-size: 16px; font-weight: 700;">🌐 Login with Solid Pod (Beta)</h3>
                    <p style="color: var(--swiss-gray-500); line-height: 1.5; margin-bottom: 15px; font-size: 13px;">
                        <strong>Own your data!</strong> Authenticate with your Solid Pod and store your location data in YOUR personal Pod, not our servers.
                        <a href="https://solidproject.org" target="_blank" style="color: #4caf50; text-decoration: underline;">Learn about Solid →</a>
                    </p>
                    <div id="solid-provider-select" style="margin-bottom: 15px;">
                        <label style="display: block; margin-bottom: 8px; font-weight: 600; color: var(--swiss-black); font-size: 13px;">Select Your Pod Provider:</label>
                        <select id="solid-provider" style="width: 100%; padding: 12px; border: 2px solid #4caf50; border-radius: 8px; font-size: 14px; background: white; cursor: pointer;">
                            <option value="">Choose a provider...</option>
                            <option value="https://login.inrupt.com">Inrupt PodSpaces (Recommended)</option>
                            <option value="https://solidcommunity.net">SolidCommunity.net</option>
                            <option value="https://solidweb.me">SolidWeb</option>
                            <option value="custom">Custom Provider...</option>
                        </select>
                    </div>
                    <input type="url" id="solid-custom-issuer" placeholder="Enter custom issuer URL (e.g., https://mypod.example.com)" style="width: 100%; padding: 12px; border: 2px solid #4caf50; border-radius: 8px; font-size: 14px; margin-bottom: 15px; display: none;">
                    <button onclick="solidLogin()" style="background: #4caf50; width: 100%;">🔓 Login with Solid Pod</button>
                    <div style="margin-top: 10px; font-size: 11px; color: var(--swiss-gray-500); text-align: center;">
                        Don't have a Pod? <a href="https://signup.pod.inrupt.com" target="_blank" style="color: #4caf50; text-decoration: underline;">Get one free</a>
                    </div>
                </div>

                <!-- Or Traditional Login -->
                <div style="text-align: center; margin: 20px 0; color: var(--swiss-gray-500); font-weight: 600; text-transform: uppercase; letter-spacing: 0.1em; font-size: 12px;">— OR —</div>

                <input type="password" id="password" placeholder="Enter full access password">
                <button onclick="login()">🔓 Full Login</button>
                <div class="error" id="error">Invalid password. Please try again.</div>
            </div>

            <!-- Tracker View -->
            <div id="tracker">
                <!-- types.Donation Section -->
                <div id="donation-section" style="background: linear-gradient(135deg, #ffd700 0%, #ffa500 100%); padding: 20px; border-radius: 12px; margin-bottom: 20px; border: 3px solid var(--swiss-black); box-shadow: 4px 4px 0px rgba(0, 0, 0, 0.2);">
                    <h3 style="color: #000; margin-bottom: 12px; text-transform: uppercase; letter-spacing: 0.1em; font-size: 18px;">💰 Support This Project</h3>
                    <p style="color: rgba(0,0,0,0.8); font-size: 13px; margin-bottom: 15px; font-family: 'Courier New', monospace; line-height: 1.5;">
                        Help maintain this satirical educational project and create a memorable bank statement entry
                    </p>
                    <div style="display: flex; gap: 12px; flex-wrap: wrap;">
                        <button onclick="openDonationModal('meme_disclaimer')" style="flex: 1; min-width: 200px; padding: 15px; background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%); color: white; border: 2px solid var(--swiss-black); border-radius: 8px; font-size: 14px; font-weight: 700; cursor: pointer; box-shadow: 3px 3px 0px rgba(0, 0, 0, 0.2); transition: all 0.2s;">
                            💳 $0.50 - Meme Disclaimer Record
                        </button>
                        <button onclick="openDonationModal('church_committee')" style="flex: 1; min-width: 200px; padding: 15px; background: linear-gradient(135deg, #8b5cf6 0%, #a78bfa 100%); color: white; border: 2px solid var(--swiss-black); border-radius: 8px; font-size: 14px; font-weight: 700; cursor: pointer; box-shadow: 3px 3px 0px rgba(0, 0, 0, 0.2); transition: all 0.2s;">
                            💳 $0.75 - Church Committee Reference
                        </button>
                    </div>
                </div>

                <div class="actions">
                    <button class="btn-share" onclick="shareLocation()">📍 Share Location</button>
                    <button class="btn-refresh" onclick="refreshLocations()">🔄 Refresh</button>
                </div>
                {{if .GoogleMapsAPIKey}}
                <div id="location-simulator" style="margin-top: 15px; padding: 15px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); border-radius: 12px; border: 3px solid var(--swiss-black); box-shadow: 4px 4px 0px rgba(0, 0, 0, 0.2); display: none;">
                    <h4 style="color: var(--swiss-white); margin-bottom: 10px; font-size: 14px; text-transform: uppercase;">🗺️ Simulate Location (Authenticated Users)</h4>
                    <input type="text" id="location-autocomplete" placeholder="Search for a location..." style="width: 100%; padding: 12px; border: 2px solid var(--swiss-black); border-radius: 8px; font-size: 14px; margin-bottom: 10px; box-shadow: 2px 2px 0px rgba(0, 0, 0, 0.1);">
                    <button onclick="shareSimulatedLocation()" style="width: 100%; background: linear-gradient(135deg, var(--pop-hot-pink) 0%, var(--memphis-pink) 100%);">📍 Share Simulated Location</button>
                </div>
                {{end}}

                <!-- Anonymous Tip Submission Form -->
                <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 20px; border-radius: 12px; margin-top: 20px; border: 3px solid var(--swiss-black); box-shadow: 4px 4px 0px rgba(0, 0, 0, 0.2);">
                    <h3 style="color: var(--swiss-white); margin-bottom: 15px; text-transform: uppercase; letter-spacing: 0.1em;">🕵️ Report Not Spy Work</h3>
                    <p style="color: rgba(255,255,255,0.9); font-size: 13px; margin-bottom: 15px; font-family: 'Courier New', monospace;">Anonymously submit tips about suspicious non-espionage activities</p>
                    <textarea id="tip-content" placeholder="Describe what you observed..." maxlength="1000" style="width: 100%; min-height: 100px; padding: 12px; border: 2px solid var(--swiss-black); border-radius: 8px; font-size: 14px; font-family: inherit; resize: vertical; box-shadow: 2px 2px 0px rgba(0, 0, 0, 0.1);"></textarea>
                    <div style="display: flex; justify-content: space-between; align-items: center; margin-top: 10px; color: rgba(255,255,255,0.8); font-size: 12px;">
                        <span><span id="char-count">0</span>/1000</span>
                        <span id="rate-limit-info" style="font-family: 'Courier New', monospace;"></span>
                    </div>
                    <button onclick="submitTip()" style="margin-top: 12px; background: linear-gradient(135deg, var(--pop-hot-pink) 0%, var(--memphis-pink) 100%);">📝 Submit Anonymous Tip</button>
                    <div id="tip-submission-result" style="margin-top: 12px; padding: 12px; border-radius: 8px; display: none;"></div>
                    <div id="user-hash-display" style="margin-top: 10px; font-family: 'Courier New', monospace; font-size: 12px; color: rgba(255,255,255,0.9); display: none;"></div>
                </div>

                <h3 style="margin-top: 20px; color: #667eea;">📍 Device Locations</h3>
                <div id="locations"></div>
                <div style="display: flex; justify-content: space-between; align-items: center; margin-top: 30px;">
                    <h3 style="color: #667eea; margin: 0;">📝 Recent Error Logs</h3>
                    <div style="display: flex; align-items: center; gap: 10px;">
                        <span style="font-size: 11px; color: var(--text-secondary); text-transform: uppercase; font-weight: 600;">With Notes/Tips Only</span>
                        <div class="filter-toggle" onclick="toggleErrorFilter()" style="position: relative; width: 44px; height: 22px; background: rgba(102, 126, 234, 0.2); border-radius: 11px; cursor: pointer; transition: background 0.3s ease; border: 1px solid rgba(102, 126, 234, 0.3);">
                            <div class="filter-toggle-switch" style="position: absolute; top: 2px; left: 2px; width: 16px; height: 16px; background: #667eea; border-radius: 50%; transition: transform 0.3s ease; box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);"></div>
                        </div>
                    </div>
                </div>
                <div id="errorlogs"></div>
                <h3 style="margin-top: 30px; color: #667eea;">🏢 Commercial Real Estate Near You</h3>
                <div id="commercialrealestate"></div>
            </div>
        </div>
    </div>

    <script>
        let deviceID = localStorage.getItem('deviceID');
        if (!deviceID) {
            deviceID = 'device_' + Math.random().toString(36).substr(2, 9);
            localStorage.setItem('deviceID', deviceID);
        }

        // Load today's cryptogram on page load
        async function loadCryptogram() {
            try {
                const res = await fetch('/api/cryptogram/info');
                if (!res.ok) throw new Error('Failed to load cryptogram');

                const crypto = await res.json();

                // Format cipher text with line breaks every 50 characters for readability
                const formattedCipher = crypto.cipher_text.match(/.{1,50}/g).join('<br/>');
                document.getElementById('cipher-text').innerHTML = formattedCipher;

                // Populate book info
                document.getElementById('book-title').textContent = crypto.book_title;
                document.getElementById('book-author').textContent = 'by ' + crypto.book_author;
                document.getElementById('book-description').textContent = crypto.book_description;

                // Show book cover if available
                if (crypto.book_cover) {
                    const coverImg = document.getElementById('book-cover');
                    coverImg.src = crypto.book_cover;
                    coverImg.style.display = 'block';
                }

                // Format hints
                const keywordsHint = '🔑 Key words to look for: ' + crypto.hint_keywords.join(', ');
                document.getElementById('hint-keywords').textContent = keywordsHint;

                const numbersHint = '📄 Chapter ' + crypto.hint_numbers[0] + ', Page ' + crypto.hint_numbers[1];
                document.getElementById('hint-numbers').textContent = numbersHint;

            } catch (e) {
                console.error('Error loading cryptogram:', e);
                document.getElementById('cipher-text').textContent = 'Failed to load cryptogram. Please refresh the page.';
            }
        }

        // Load cryptogram when page loads
        loadCryptogram();

        // Cryptogram solver
        async function solveCryptogram() {
            const answer = document.getElementById('cryptogram').value;
            const errorEl = document.getElementById('cryptogram-error');

            try {
                const res = await fetch('/api/cryptogram', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({answer})
                });

                if (res.ok) {
                    document.getElementById('login').style.display = 'none';
                    document.getElementById('tracker').style.display = 'block';
                    refreshErrorLogs();
                    // Hide location-related sections for puzzle-only access
                    document.querySelector('.btn-share').style.display = 'none';
                    document.querySelector('.btn-refresh').textContent = '🔄 Refresh';
                    // Auto-refresh error logs every 10 seconds
                    setInterval(() => {
                        refreshErrorLogs();
                    }, 10000);
                } else {
                    errorEl.style.display = 'block';
                    setTimeout(() => errorEl.style.display = 'none', 3000);
                }
            } catch (e) {
                alert('Connection error: ' + e.message);
            }
        }

        // Theme Toggle
        function toggleTheme() {
            const body = document.body;
            const themeLabel = document.getElementById('theme-label');
            const currentTheme = body.getAttribute('data-theme');

            if (currentTheme === 'terminal') {
                body.removeAttribute('data-theme');
                themeLabel.textContent = 'Terminal';
                localStorage.setItem('theme', 'default');
            } else {
                body.setAttribute('data-theme', 'terminal');
                themeLabel.textContent = 'Colorful';
                localStorage.setItem('theme', 'terminal');
            }
        }

        // Load saved theme on page load
        window.addEventListener('DOMContentLoaded', () => {
            const savedTheme = localStorage.getItem('theme');
            const themeLabel = document.getElementById('theme-label');
            if (savedTheme === 'terminal') {
                document.body.setAttribute('data-theme', 'terminal');
                themeLabel.textContent = 'Colorful';
            }

            // Load error filter preference
            const savedFilter = localStorage.getItem('errorFilter');
            if (savedFilter === 'notesOnly') {
                filterErrorsWithNotesOnly = true;
                document.querySelector('.filter-toggle-switch').style.transform = 'translateX(22px)';
                document.querySelector('.filter-toggle').style.background = '#667eea';
            }
        });

        // Error filter state
        let filterErrorsWithNotesOnly = false;
        let allErrorLogs = []; // Store all error logs
        let isLoggedIn = false; // Track authentication state
        let pendingRorschachInterpretations = new Set(); // Track which interpretations are being generated
        let displayedErrorIds = new Set(); // Track which error logs are already displayed to avoid reloading

        // Easter egg pattern detection for filter toggle
        let toggleClickTimes = [];
        const DOJ_PATTERN_TIMEOUT = 2000; // Max time window for pattern

        // Toggle error filter
        function toggleErrorFilter() {
            const toggle = document.querySelector('.filter-toggle-switch');
            const container = document.querySelector('.filter-toggle');

            // Track click timing for Easter egg pattern detection
            const now = Date.now();
            toggleClickTimes.push(now);

            // Keep only last 3 clicks
            if (toggleClickTimes.length > 3) {
                toggleClickTimes.shift();
            }

            // Check for pattern: 1 click, pause (>500ms), 2 fast clicks (<500ms apart)
            if (toggleClickTimes.length === 3) {
                const firstToSecond = toggleClickTimes[1] - toggleClickTimes[0];
                const secondToThird = toggleClickTimes[2] - toggleClickTimes[1];
                const totalTime = toggleClickTimes[2] - toggleClickTimes[0];

                // Pattern: first pause > 500ms, then two fast clicks < 500ms apart, total < 2s
                if (firstToSecond > 500 && secondToThird < 500 && totalTime < DOJ_PATTERN_TIMEOUT) {
                    // Easter egg triggered!
                    showDOJBanner();
                    toggleClickTimes = []; // Reset pattern
                    return; // Don't toggle the filter
                }
            }

            // Clean up old clicks (older than 2 seconds)
            toggleClickTimes = toggleClickTimes.filter(time => (now - time) < DOJ_PATTERN_TIMEOUT);

            // Normal toggle behavior
            filterErrorsWithNotesOnly = !filterErrorsWithNotesOnly;

            if (filterErrorsWithNotesOnly) {
                toggle.style.transform = 'translateX(22px)';
                container.style.background = '#667eea';
                localStorage.setItem('errorFilter', 'notesOnly');
            } else {
                toggle.style.transform = 'translateX(0)';
                container.style.background = 'rgba(102, 126, 234, 0.2)';
                localStorage.setItem('errorFilter', 'all');
            }

            // Re-display with filter (clear tracking to force full re-render)
            displayedErrorIds.clear();
            displayErrorLogs(allErrorLogs);
        }

        // Login
        async function login() {
            const password = document.getElementById('password').value;
            const errorEl = document.getElementById('error');

            try {
                const res = await fetch('/api/login', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({password})
                });

                if (res.ok) {
                    isLoggedIn = true; // Set authentication state
                    document.getElementById('login').style.display = 'none';
                    document.getElementById('tracker').style.display = 'block';
                    refreshLocations();
                    refreshErrorLogs();
                    // Show location simulator for authenticated users
                    if (document.getElementById('location-simulator')) {
                        document.getElementById('location-simulator').style.display = 'block';
                        initLocationAutocomplete();
                    }
                    // Auto-refresh every 10 seconds
                    setInterval(() => {
                        refreshLocations();
                        refreshErrorLogs();
                        refreshCommercialRealEstate();
                    }, 10000);
                } else {
                    errorEl.style.display = 'block';
                    setTimeout(() => errorEl.style.display = 'none', 3000);
                }
            } catch (e) {
                alert('Connection error: ' + e.message);
            }
        }

        // Solid Login Functions
        async function solidLogin() {
            const providerSelect = document.getElementById('solid-provider');
            const customIssuer = document.getElementById('solid-custom-issuer');

            let issuerURL = providerSelect.value;

            if (issuerURL === 'custom') {
                issuerURL = customIssuer.value.trim();
                if (!issuerURL || !issuerURL.startsWith('https://')) {
                    alert('Please enter a valid HTTPS URL for your Pod provider');
                    return;
                }
            }

            if (!issuerURL) {
                alert('Please select a Pod provider');
                return;
            }

            try {
                console.log('🔐 Initiating Solid login with issuer:', issuerURL);
                const res = await fetch('/api/solid/login', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({ issuer_url: issuerURL })
                });

                if (!res.ok) {
                    const errorData = await res.json().catch(() => ({ message: 'Unknown error' }));
                    console.error('❌ Solid login failed:', errorData);

                    let errorMessage = errorData.message || 'Failed to initiate Solid login';
                    if (errorData.help) {
                        errorMessage += '\n\n' + errorData.help;
                    }
                    if (errorData.error === 'solid_not_enabled') {
                        errorMessage = '⚠️ Solid authentication is currently disabled on this server.\n\n' +
                                     'The feature is in beta. Contact the administrator to enable it.';
                    }

                    alert(errorMessage);
                    return;
                }

                const data = await res.json();
                console.log('✅ Received authorization URL:', data.authorization_url);

                // Redirect to provider's login page
                window.location.href = data.authorization_url;
            } catch (e) {
                console.error('❌ Solid login error:', e);
                alert('Failed to initiate Solid login: ' + e.message + '\n\nPlease check your internet connection and try again.');
            }
        }

        // Check for Solid callback on page load
        async function checkSolidCallback() {
            const params = new URLSearchParams(window.location.search);
            if (params.has('solid') && params.get('solid') === 'success') {
                console.log('✅ Solid authentication successful!');
                // Check session
                const res = await fetch('/api/solid/session');
                const session = await res.json();

                if (session.authenticated && session.auth_type === 'solid') {
                    console.log('🌐 Logged in with Solid Pod:', session.pod_url);
                    isLoggedIn = true;
                    document.getElementById('login').style.display = 'none';
                    document.getElementById('tracker').style.display = 'block';

                    // Show storage info banner
                    showSolidBanner(session);

                    refreshLocations();
                    refreshErrorLogs();

                    // Show location simulator
                    if (document.getElementById('location-simulator')) {
                        document.getElementById('location-simulator').style.display = 'block';
                        initLocationAutocomplete();
                    }

                    // Auto-refresh
                    setInterval(() => {
                        refreshLocations();
                        refreshErrorLogs();
                        refreshCommercialRealEstate();
                    }, 10000);

                    // Clean URL
                    window.history.replaceState({}, '', '/');
                }
            }
        }

        // Show Solid session banner
        function showSolidBanner(session) {
            const banner = document.createElement('div');
            banner.style.cssText = 'background: linear-gradient(135deg, #4caf50, #45a049); color: white; padding: 15px; border-radius: 8px; margin-bottom: 20px; border: 2px solid #fff; box-shadow: 0 4px 6px rgba(0,0,0,0.1);';
            banner.innerHTML = '<div style="display: flex; justify-content: space-between; align-items: center;">' +
                '<div>' +
                '<strong style="font-size: 16px;">🌐 Connected to Your Solid Pod</strong>' +
                '<div style="font-size: 12px; margin-top: 5px; opacity: 0.9;">' +
                'Your data is stored in: <strong>' + session.pod_url + '</strong>' +
                '</div>' +
                '<div style="font-size: 11px; margin-top: 3px; opacity: 0.8;">' +
                'WebID: ' + session.webid +
                '</div>' +
                '</div>' +
                '<button onclick="solidLogout()" style="background: rgba(255,255,255,0.2); color: white; border: 2px solid white; padding: 8px 16px; border-radius: 6px; cursor: pointer; font-weight: 600;">' +
                'Logout' +
                '</button>' +
                '</div>';
            document.getElementById('tracker').insertBefore(banner, document.getElementById('tracker').firstChild);
        }

        // Solid logout
        async function solidLogout() {
            if (!confirm('Are you sure you want to logout? Your data remains safe in your Pod.')) {
                return;
            }

            try {
                await fetch('/api/solid/logout', { method: 'POST' });
                location.reload();
            } catch (e) {
                console.error('Error logging out:', e);
                location.reload();
            }
        }

        // Handle custom provider selection
        document.addEventListener('DOMContentLoaded', () => {
            const providerSelect = document.getElementById('solid-provider');
            const customIssuer = document.getElementById('solid-custom-issuer');

            providerSelect.addEventListener('change', (e) => {
                if (e.target.value === 'custom') {
                    customIssuer.style.display = 'block';
                } else {
                    customIssuer.style.display = 'none';
                }
            });

            // Check for Solid callback
            checkSolidCallback();
        });

        // Handle Enter key on password and cryptogram fields
        document.addEventListener('DOMContentLoaded', () => {
            document.getElementById('password').addEventListener('keypress', (e) => {
                if (e.key === 'Enter') login();
            });
            document.getElementById('cryptogram').addEventListener('keypress', (e) => {
                if (e.key === 'Enter') solveCryptogram();
            });
        });

        // Share current location
        async function shareLocation() {
            if (!navigator.geolocation) {
                alert('❌ Geolocation not supported by this browser');
                return;
            }

            const btn = event.target;
            btn.textContent = '📡 Getting location...';
            btn.disabled = true;

            navigator.geolocation.getCurrentPosition(async (pos) => {
                const location = {
                    latitude: pos.coords.latitude,
                    longitude: pos.coords.longitude,
                    accuracy: pos.coords.accuracy,
                    device_id: deviceID
                };

                try {
                    await fetch('/api/location', {
                        method: 'POST',
                        headers: {'Content-Type': 'application/json'},
                        body: JSON.stringify(location)
                    });

                    btn.textContent = '✅ Location shared!';
                    setTimeout(() => {
                        btn.textContent = '📍 Share Location';
                        btn.disabled = false;
                    }, 2000);

                    refreshLocations();
                } catch (e) {
                    alert('Error sharing location: ' + e.message);
                    btn.textContent = '📍 Share Location';
                    btn.disabled = false;
                }
            }, (err) => {
                alert('❌ Location access denied: ' + err.message);
                btn.textContent = '📍 Share Location';
                btn.disabled = false;
            }, {
                enableHighAccuracy: true,
                timeout: 10000,
                maximumAge: 0
            });
        }

        // Google Maps location autocomplete
        let selectedPlace = null;

        function initLocationAutocomplete() {
            if (typeof google === 'undefined' || !google.maps || !google.maps.places) {
                console.log('Google Maps not loaded, skipping autocomplete');
                return;
            }

            const input = document.getElementById('location-autocomplete');
            const autocomplete = new google.maps.places.Autocomplete(input);

            autocomplete.addListener('place_changed', () => {
                selectedPlace = autocomplete.getPlace();
                if (!selectedPlace.geometry) {
                    alert('No location data available for this place');
                    selectedPlace = null;
                }
            });
        }

        async function shareSimulatedLocation() {
            if (!selectedPlace || !selectedPlace.geometry) {
                alert('Please select a location from the autocomplete suggestions');
                return;
            }

            const btn = event.target;
            btn.textContent = '📡 Sharing simulated location...';
            btn.disabled = true;

            const location = {
                latitude: selectedPlace.geometry.location.lat(),
                longitude: selectedPlace.geometry.location.lng(),
                accuracy: 0,
                device_id: deviceID,
                simulated: true,
                location_name: selectedPlace.formatted_address || selectedPlace.name
            };

            try {
                await fetch('/api/location', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(location)
                });

                btn.textContent = '✅ Simulated location shared!';
                setTimeout(() => {
                    btn.textContent = '📍 Share Simulated Location';
                    btn.disabled = false;
                }, 2000);

                refreshLocations();
                // Clear the input
                document.getElementById('location-autocomplete').value = '';
                selectedPlace = null;
            } catch (e) {
                alert('Error sharing simulated location: ' + e.message);
                btn.textContent = '📍 Share Simulated Location';
                btn.disabled = false;
            }
        }

        // Submit anonymous tip
        async function submitTip() {
            const content = document.getElementById('tip-content').value;
            const resultEl = document.getElementById('tip-submission-result');
            const userHashEl = document.getElementById('user-hash-display');

            if (!content.trim()) {
                resultEl.style.cssText = 'display: block; background: #fee; color: #c33; border: 2px solid #c33;';
                resultEl.textContent = '❌ Please enter a tip';
                setTimeout(() => resultEl.style.display = 'none', 3000);
                return;
            }

            try {
                const res = await fetch('/api/tips', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({tip_content: content})
                });

                const result = await res.json();

                if (result.status === 'success') {
                    resultEl.style.cssText = 'display: block; background: #d1fae5; color: #065f46; border: 2px solid #10b981;';
                    resultEl.innerHTML = '✅ Tip submitted successfully!' + (result.moderated ? '<br>⚠️ Some content was redacted' : '');

                    userHashEl.style.display = 'block';
                    userHashEl.innerHTML = 'Your anonymous ID: <code style="background: rgba(102, 126, 234, 0.15); padding: 4px 8px; border-radius: 4px; color: #4338ca; font-weight: 600;">' + result.user_hash + '</code>';

                    document.getElementById('tip-content').value = '';
                    document.getElementById('char-count').textContent = '0';

                    // Update rate limit info
                    const remaining = res.headers.get('X-RateLimit-Remaining');
                    if (remaining) {
                        document.getElementById('rate-limit-info').textContent = remaining + ' tips remaining this hour';
                    }

                    setTimeout(() => {
                        resultEl.style.display = 'none';
                        userHashEl.style.display = 'none';
                    }, 10000);
                } else if (result.status === 'rejected') {
                    resultEl.style.cssText = 'display: block; background: #fee; color: #c33; border: 2px solid #c33;';
                    resultEl.textContent = '❌ ' + (result.reason || 'Content was rejected');
                    setTimeout(() => resultEl.style.display = 'none', 5000);
                } else if (result.status === 'rate_limited') {
                    resultEl.style.cssText = 'display: block; background: #fef3c7; color: #92400e; border: 2px solid #f59e0b;';
                    resultEl.textContent = '⏱️ Rate limit exceeded. Try again later.';
                    setTimeout(() => resultEl.style.display = 'none', 5000);
                } else if (result.status === 'banned') {
                    resultEl.style.cssText = 'display: block; background: #fee; color: #c33; border: 2px solid #c33;';
                    resultEl.textContent = '🚫 User temporarily banned';
                    setTimeout(() => resultEl.style.display = 'none', 5000);
                } else {
                    resultEl.style.cssText = 'display: block; background: #fee; color: #c33; border: 2px solid #c33;';
                    resultEl.textContent = '❌ ' + (result.reason || 'Submission failed');
                    setTimeout(() => resultEl.style.display = 'none', 5000);
                }
            } catch (e) {
                resultEl.style.cssText = 'display: block; background: #fee; color: #c33; border: 2px solid #c33;';
                resultEl.textContent = '❌ Connection error: ' + e.message;
                setTimeout(() => resultEl.style.display = 'none', 3000);
            }
        }

        // Character counter for tip form
        document.addEventListener('DOMContentLoaded', () => {
            const tipContent = document.getElementById('tip-content');
            if (tipContent) {
                tipContent.addEventListener('input', (e) => {
                    document.getElementById('char-count').textContent = e.target.value.length;
                });
            }
        });

        // Refresh locations
        async function refreshLocations() {
            try {
                const res = await fetch('/api/location');
                if (!res.ok) {
                    // Session expired, reload to login
                    location.reload();
                    return;
                }

                const locations = await res.json();
                displayLocations(locations);
            } catch (e) {
                console.error('Error fetching locations:', e);
            }
        }

        // Display locations
        function displayLocations(locations) {
            const container = document.getElementById('locations');

            if (Object.keys(locations).length === 0) {
                container.innerHTML = ` + "`" + `
                    <div class="empty-state">
                        <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                  d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"/>
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                                  d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"/>
                        </svg>
                        <p>No locations shared yet</p>
                        <p style="font-size: 14px; margin-top: 10px;">Click "Share Location" to start</p>
                    </div>
                ` + "`" + `;
                return;
            }

            container.innerHTML = '';

            for (const [id, loc] of Object.entries(locations)) {
                const age = getLocationAge(loc.timestamp);
                const isCurrentDevice = id === deviceID;

                const div = document.createElement('div');
                div.className = 'location-card';
                div.innerHTML = ` + "`" + `
                    <h3>
                        ${isCurrentDevice ? '📱 Your Device' : '📍 ' + id}
                        <span class="status">${age}</span>
                    </h3>
                    <div class="location-detail">
                        <span class="label">Latitude:</span>
                        <span class="value">${loc.latitude.toFixed(6)}°</span>
                    </div>
                    <div class="location-detail">
                        <span class="label">Longitude:</span>
                        <span class="value">${loc.longitude.toFixed(6)}°</span>
                    </div>
                    <div class="location-detail">
                        <span class="label">Accuracy:</span>
                        <span class="value">±${Math.round(loc.accuracy)}m</span>
                    </div>
                    <div class="location-detail">
                        <span class="label">Updated:</span>
                        <span class="value">${new Date(loc.timestamp).toLocaleString()}</span>
                    </div>
                    <a href="https://www.google.com/maps?q=${loc.latitude},${loc.longitude}"
                       target="_blank" class="map-link">
                        🗺️ View on Google Maps
                    </a>
                ` + "`" + `;
                container.appendChild(div);
            }
        }

        function getLocationAge(timestamp) {
            const seconds = Math.floor((new Date() - new Date(timestamp)) / 1000);
            if (seconds < 60) return 'Just now';
            if (seconds < 3600) return Math.floor(seconds / 60) + 'm ago';
            if (seconds < 86400) return Math.floor(seconds / 3600) + 'h ago';
            return Math.floor(seconds / 86400) + 'd ago';
        }

        // Refresh error logs
        async function refreshErrorLogs() {
            try {
                const res = await fetch('/api/errorlogs');
                if (!res.ok) {
                    location.reload();
                    return;
                }

                const errorLogs = await res.json();

                // Check if there are any new error logs
                const newErrors = errorLogs.filter(log => !displayedErrorIds.has(log.id));

                if (newErrors.length > 0) {
                    // There are new errors, update the display
                    displayErrorLogs(errorLogs);
                } else {
                    // No new errors, just update the allErrorLogs array for filtering
                    allErrorLogs = errorLogs || [];
                }
            } catch (e) {
                console.error('Error fetching error logs:', e);
            }
        }

        // Store expanded state of collapsibles (keyed by error timestamp)
        const expandedCollapsibles = new Map();

        // Display error logs
        function displayErrorLogs(errorLogs) {
            const container = document.getElementById('errorlogs');

            // Store all logs for filtering
            allErrorLogs = errorLogs || [];

            if (!errorLogs || errorLogs.length === 0) {
                container.innerHTML = ` + "`" + `
                    <div class="empty-state" style="padding: 40px 20px;">
                        <p style="color: #6b7280;">No error logs yet</p>
                        <p style="font-size: 14px; margin-top: 10px; color: #9ca3af;">
                            Error logs from error-generator will appear here
                        </p>
                    </div>
                ` + "`" + `;
                return;
            }

            // Apply filter if enabled
            let filteredLogs = errorLogs;
            if (filterErrorsWithNotesOnly) {
                filteredLogs = errorLogs.filter(log =>
                    (log.user_experience_note && log.user_experience_note.trim() !== '') ||
                    (log.anonymous_tips && log.anonymous_tips.length > 0)
                );

                if (filteredLogs.length === 0) {
                    container.innerHTML = ` + "`" + `
                        <div class="empty-state" style="padding: 40px 20px;">
                            <p style="color: #6b7280;">No error logs with notes or tips</p>
                            <p style="font-size: 14px; margin-top: 10px; color: #9ca3af;">
                                Toggle the filter off to see all error logs
                            </p>
                        </div>
                    ` + "`" + `;
                    return;
                }
            }

            errorLogs = filteredLogs;

            // Save current expanded state before re-rendering
            const existingCards = container.querySelectorAll('.location-card');
            existingCards.forEach((card, index) => {
                const satiricalContent = card.querySelector('.collapsible-content.satirical-fix');
                const storyContent = card.querySelector('.collapsible-content.story');
                const gifExpandButton = card.querySelector('[id^="gif-expand-btn-"]');

                if (satiricalContent && satiricalContent.classList.contains('expanded')) {
                    const timestamp = card.dataset.timestamp;
                    if (!expandedCollapsibles.has(timestamp)) {
                        expandedCollapsibles.set(timestamp, {});
                    }
                    expandedCollapsibles.get(timestamp).satirical = true;
                }

                if (storyContent && storyContent.classList.contains('expanded')) {
                    const timestamp = card.dataset.timestamp;
                    if (!expandedCollapsibles.has(timestamp)) {
                        expandedCollapsibles.set(timestamp, {});
                    }
                    expandedCollapsibles.get(timestamp).story = true;
                }

                // Save GIF expansion state
                if (gifExpandButton && gifExpandButton.dataset.expanded === 'true') {
                    const timestamp = card.dataset.timestamp;
                    if (!expandedCollapsibles.has(timestamp)) {
                        expandedCollapsibles.set(timestamp, {});
                    }
                    expandedCollapsibles.get(timestamp).gifsExpanded = true;
                }
            });

            container.innerHTML = '';

            // Backend already returns most recent errors first, no need to slice or reverse
            for (const errorLog of errorLogs) {
                const age = getLocationAge(errorLog.timestamp);

                const div = document.createElement('div');
                div.className = 'location-card';
                div.style.borderLeft = '4px solid #ef4444';
                div.dataset.timestamp = errorLog.timestamp;

                // Build C-SPAN section if present
                let cspanHTML = '';
                if (errorLog.cspan_video || errorLog.cspan_livestream) {
                    cspanHTML += '<div style="margin-bottom: 20px; padding: 20px; background: linear-gradient(135deg, rgba(59, 130, 246, 0.08) 0%, rgba(37, 99, 235, 0.12) 100%); border: 3px solid rgba(59, 130, 246, 0.4); border-radius: 12px; box-shadow: 0 0 25px rgba(59, 130, 246, 0.25), 4px 4px 0px rgba(59, 130, 246, 0.15);">';
                    cspanHTML += '<div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 15px;">';
                    cspanHTML += '<div style="display: flex; align-items: center; gap: 10px;">';
                    cspanHTML += '<span style="font-size: 24px;">🏛️</span>';
                    cspanHTML += '<strong style="background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; font-size: 16px; text-transform: uppercase; letter-spacing: 0.08em; font-weight: 800;">C-SPAN Coverage</strong>';
                    cspanHTML += '</div>';
                    cspanHTML += '<span style="font-size: 9px; color: #999; font-weight: 600; letter-spacing: 0.05em;">RELATED GOVERNMENT PROCEEDINGS</span>';
                    cspanHTML += '</div>';

                    if (errorLog.cspan_video) {
                        cspanHTML += '<div style="background: rgba(255, 255, 255, 0.9); padding: 15px; border-radius: 8px; border: 2px solid rgba(59, 130, 246, 0.3);">';
                        cspanHTML += '<div style="margin-bottom: 12px;">';
                        cspanHTML += '<div style="font-weight: 700; color: #1e40af; font-size: 15px; margin-bottom: 6px;">' + (errorLog.cspan_video.title || 'C-SPAN Video') + '</div>';
                        if (errorLog.cspan_video.description) {
                            cspanHTML += '<div style="font-size: 13px; color: #64748b; line-height: 1.5; margin-bottom: 8px;">' + errorLog.cspan_video.description + '</div>';
                        }
                        cspanHTML += '<div style="display: flex; gap: 12px; font-size: 12px; color: #94a3b8;">';
                        if (errorLog.cspan_video.date) {
                            cspanHTML += '<span>📅 ' + errorLog.cspan_video.date + '</span>';
                        }
                        if (errorLog.cspan_video.duration) {
                            cspanHTML += '<span>⏱️ ' + errorLog.cspan_video.duration + '</span>';
                        }
                        cspanHTML += '</div>';
                        cspanHTML += '</div>';
                        if (errorLog.cspan_video.url) {
                            cspanHTML += '<a href="' + errorLog.cspan_video.url + '" target="_blank" rel="noopener noreferrer" style="display: inline-block; padding: 10px 20px; background: #3b82f6; color: white; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 13px; transition: all 0.2s; box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);">View on C-SPAN →</a>';
                        }
                        cspanHTML += '</div>';
                    }

                    if (errorLog.cspan_livestream && errorLog.cspan_livestream.is_live) {
                        cspanHTML += '<div style="background: rgba(239, 68, 68, 0.1); padding: 12px; border-radius: 8px; border: 2px solid rgba(239, 68, 68, 0.3); margin-top: 10px;">';
                        cspanHTML += '<div style="display: flex; align-items: center; gap: 8px; color: #dc2626; font-weight: 700; font-size: 14px;">';
                        cspanHTML += '<span style="display: inline-block; width: 8px; height: 8px; background: #dc2626; border-radius: 50%; animation: pulse 2s infinite;"></span>';
                        cspanHTML += 'LIVE NOW: ' + (errorLog.cspan_livestream.title || 'C-SPAN Live Coverage');
                        cspanHTML += '</div>';
                        cspanHTML += '</div>';
                    }

                    cspanHTML += '</div>';
                }

                // Build TikTok section if present (collapsed by default, no iframe)
                let tiktokHTML = '';
                if (errorLog.tiktok_video) {
                    tiktokHTML += '<details style="margin-bottom: 15px; padding: 12px; background: linear-gradient(135deg, rgba(236, 72, 153, 0.06) 0%, rgba(219, 39, 119, 0.09) 100%); border: 2px solid rgba(236, 72, 153, 0.25); border-radius: 8px; cursor: pointer;">';
                    tiktokHTML += '<summary style="list-style: none; display: flex; align-items: center; justify-content: space-between; outline: none; user-select: none;">';
                    tiktokHTML += '<div style="display: flex; align-items: center; gap: 8px;">';
                    tiktokHTML += '<span style="font-size: 18px;">🎵</span>';
                    tiktokHTML += '<strong style="background: linear-gradient(135deg, #ec4899 0%, #db2777 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; font-size: 13px; text-transform: uppercase; letter-spacing: 0.08em; font-weight: 800;">TikTok</strong>';
                    if (errorLog.tiktok_video.title) {
                        tiktokHTML += '<span style="font-size: 12px; color: #64748b; font-weight: 500; margin-left: 6px;">• ' + errorLog.tiktok_video.title + '</span>';
                    }
                    tiktokHTML += '</div>';
                    tiktokHTML += '<span style="font-size: 9px; color: #94a3b8; font-weight: 600;">▶</span>';
                    tiktokHTML += '</summary>';

                    tiktokHTML += '<div style="background: rgba(255, 255, 255, 0.95); padding: 12px; border-radius: 6px; border: 1px solid rgba(236, 72, 153, 0.2); margin-top: 10px;">';

                    if (errorLog.tiktok_video.author) {
                        tiktokHTML += '<div style="font-size: 12px; color: #64748b; margin-bottom: 8px;">👤 @' + errorLog.tiktok_video.author + '</div>';
                    }

                    if (errorLog.tiktok_video.description) {
                        tiktokHTML += '<div style="font-size: 11px; color: #94a3b8; font-style: italic; margin-bottom: 10px; padding: 8px; background: rgba(236, 72, 153, 0.04); border-radius: 4px;">' + errorLog.tiktok_video.description + '</div>';
                    }

                    // Tags
                    if (errorLog.tiktok_video.tags && errorLog.tiktok_video.tags.length > 0) {
                        tiktokHTML += '<div style="display: flex; flex-wrap: wrap; gap: 5px; margin-bottom: 10px;">';
                        errorLog.tiktok_video.tags.forEach(tag => {
                            tiktokHTML += '<span style="background: rgba(236, 72, 153, 0.12); color: #be185d; padding: 3px 8px; border-radius: 10px; font-size: 10px; font-weight: 600;">#' + tag + '</span>';
                        });
                        tiktokHTML += '</div>';
                    }

                    if (errorLog.tiktok_video.url) {
                        tiktokHTML += '<a href="' + errorLog.tiktok_video.url + '" target="_blank" rel="noopener noreferrer" style="display: inline-block; padding: 8px 16px; background: linear-gradient(135deg, #ec4899 0%, #db2777 100%); color: white; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 12px; transition: all 0.2s; box-shadow: 0 2px 8px rgba(236, 72, 153, 0.3);">View on TikTok →</a>';
                    }

                    tiktokHTML += '</div>';
                    tiktokHTML += '</details>';
                }

                div.innerHTML = cspanHTML + tiktokHTML + ` + "`" + `
                    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px;">
                        <h3 style="margin: 0; color: #ef4444; font-size: 14px;">🚬 Error Log</h3>
                        <span class="status" style="background: #fee2e2; color: #991b1b;">${age}</span>
                    </div>
                    <div class="location-detail">
                        <span class="label">Message:</span>
                        <span class="value" style="font-family: inherit;">${errorLog.message}</span>
                    </div>
                    <div class="location-detail">
                        <span class="label">Slogan:</span>
                        <span class="value" style="font-family: inherit; color: #667eea;">${errorLog.slogan}</span>
                    </div>
                    ${errorLog.url ? ` + "`" + `
                        <div class="location-detail" style="margin-top: 8px; padding: 10px; background: linear-gradient(135deg, rgba(59, 130, 246, 0.06) 0%, rgba(99, 102, 241, 0.08) 100%); border-left: 3px solid #3b82f6; border-radius: 6px;">
                            <span class="label" style="color: #1e40af;">🔗 Share URL:</span>
                            <div style="margin-top: 6px; display: flex; gap: 8px; align-items: center;">
                                <input type="text" value="${errorLog.url}" readonly onclick="this.select()" style="flex: 1; padding: 6px 10px; font-family: 'Courier New', monospace; font-size: 11px; color: #1e40af; background: rgba(255, 255, 255, 0.8); border: 1px solid #60a5fa; border-radius: 4px; cursor: pointer;">
                                <button onclick="navigator.clipboard.writeText('${errorLog.url}').then(() => { const btn = event.target; const originalText = btn.textContent; btn.textContent = '✓ Copied!'; btn.style.background = '#10b981'; setTimeout(() => { btn.textContent = originalText; btn.style.background = ''; }, 2000); })" style="padding: 6px 12px; background: #3b82f6; color: white; border: none; border-radius: 4px; font-size: 11px; font-weight: 600; cursor: pointer; white-space: nowrap; transition: all 0.2s;">Copy</button>
                            </div>
                            <div style="margin-top: 10px; padding-top: 10px; border-top: 1px solid rgba(59, 130, 246, 0.2);">
                                <button onclick="shareFacebookCompilation('${errorLog.id}')" style="width: 100%; padding: 10px 16px; background: linear-gradient(135deg, #1877F2 0%, #4267B2 100%); color: white; border: none; border-radius: 6px; font-size: 13px; font-weight: 700; cursor: pointer; display: flex; align-items: center; justify-content: center; gap: 8px; box-shadow: 0 4px 12px rgba(24, 119, 242, 0.3); transition: all 0.3s; text-transform: uppercase; letter-spacing: 0.5px;" onmouseover="this.style.transform='translateY(-2px)'; this.style.boxShadow='0 6px 16px rgba(24, 119, 242, 0.4)';" onmouseout="this.style.transform='translateY(0)'; this.style.boxShadow='0 4px 12px rgba(24, 119, 242, 0.3)';">
                                    <svg style="width: 18px; height: 18px;" fill="currentColor" viewBox="0 0 24 24"><path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z"/></svg>
                                    Share to Facebook
                                </button>
                            </div>
                        </div>
                    ` + "`" + ` : ''}
                    ${errorLog.verbose_desc ? ` + "`" + `
                        <div style="margin-top: 12px; padding: 12px; background: linear-gradient(135deg, rgba(102, 126, 234, 0.06) 0%, rgba(118, 75, 162, 0.08) 100%); border-left: 4px solid #667eea; border-radius: 6px; font-family: 'Courier New', monospace; font-size: 13px; color: #374151; line-height: 1.6;">
                            ${errorLog.verbose_desc}
                        </div>
                    ` + "`" + ` : ''}
                    ${errorLog.song_title ? ` + "`" + `
                        <div class="location-detail">
                            <span class="label">Song:</span>
                            <span class="value" style="font-family: inherit; color: #1db954;">🎵 ${errorLog.song_title} by ${errorLog.song_artist}</span>
                        </div>
                    ` + "`" + ` : ''}
                    ${errorLog.user_experience_note ? ` + "`" + `
                        <div class="location-detail" style="border-left: 3px solid #10b981; padding-left: 12px; background: #f0fdf4;">
                            <span class="label" style="color: #065f46;">💬 User Note:</span>
                            <span class="value" style="font-family: inherit; color: #065f46; font-weight: 500;">${errorLog.user_experience_note}</span>
                        </div>
                    ` + "`" + ` : ''}
                    ${errorLog.satirical_fix ? ` + "`" + `
                        <div style="margin-top: 20px; padding: 20px; background: linear-gradient(135deg, rgba(139, 92, 246, 0.08) 0%, rgba(124, 58, 237, 0.12) 100%); border: 3px solid rgba(139, 92, 246, 0.4); border-radius: 12px; box-shadow: 0 0 25px rgba(139, 92, 246, 0.25), 4px 4px 0px rgba(139, 92, 246, 0.15);">
                            <div class="collapsible-header" onclick="toggleCollapsible(this)" style="display: flex; justify-content: space-between; align-items: center;">
                                <div style="display: flex; align-items: center; gap: 10px;">
                                    <span style="font-size: 24px;">🤖</span>
                                    <strong style="color: #8b5cf6; font-size: 16px; text-transform: uppercase; letter-spacing: 0.08em; font-weight: 800;">Satirical Fix Generated</strong>
                                    <span style="font-size: 9px; color: #999; font-weight: 600; letter-spacing: 0.05em;">POWERED BY DEEPSEEK</span>
                                </div>
                                <span class="collapsible-toggle">▶</span>
                            </div>
                            <div class="collapsible-content satirical-fix" style="margin-top: 15px;">
                                <pre style="white-space: pre-wrap; font-family: 'Courier New', monospace; font-size: 12px; color: #374151; background: rgba(255, 255, 255, 0.8); padding: 16px; border-radius: 8px; overflow-x: auto; margin: 0; border: 2px solid rgba(139, 92, 246, 0.3);">${errorLog.satirical_fix}</pre>
                            </div>
                        </div>
                    ` + "`" + ` : ''}
                    ${errorLog.childrens_story ? ` + "`" + `
                        <div style="margin-top: 20px; padding: 20px; background: linear-gradient(135deg, rgba(255, 107, 157, 0.08) 0%, rgba(72, 202, 228, 0.12) 100%); border: 3px solid rgba(255, 107, 157, 0.4); border-radius: 12px; box-shadow: 0 0 25px rgba(255, 107, 157, 0.25), 4px 4px 0px rgba(255, 107, 157, 0.15);">
                            <div class="collapsible-header" onclick="toggleCollapsible(this)" style="display: flex; justify-content: space-between; align-items: center;">
                                <div style="display: flex; align-items: center; gap: 10px;">
                                    <span style="font-size: 24px;">📚</span>
                                    <strong style="background: linear-gradient(135deg, #ff6b9d 0%, #48cae4 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; font-size: 16px; text-transform: uppercase; letter-spacing: 0.08em; font-weight: 800;">Investigative Comedy Story for Children</strong>
                                    <span style="font-size: 9px; color: #999; font-weight: 600; letter-spacing: 0.05em;">POWERED BY CLAUDE</span>
                                </div>
                                <span class="collapsible-toggle">▶</span>
                            </div>
                            <div class="collapsible-content story" style="margin-top: 15px;">
                                <div style="background: rgba(255, 255, 255, 0.8); padding: 20px; border-radius: 8px; border: 2px solid rgba(255, 107, 157, 0.3); font-family: 'Georgia', serif; line-height: 1.8; color: #1f2937; font-size: 14px;">
                                    ${errorLog.childrens_story.replace(/\n/g, '<br>')}
                                </div>
                            </div>
                        </div>
                    ` + "`" + ` : ''}
                    ${errorLog.meme_url ? ` + "`" + `
                        <div style="margin-top: 20px; padding: 20px; background: linear-gradient(135deg, rgba(139, 92, 246, 0.08) 0%, rgba(236, 72, 153, 0.12) 100%); border: 3px solid rgba(139, 92, 246, 0.4); border-radius: 12px; box-shadow: 0 0 25px rgba(139, 92, 246, 0.25), 4px 4px 0px rgba(139, 92, 246, 0.15);">
                            <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 15px;">
                                <div style="display: flex; align-items: center; gap: 10px;">
                                    <span style="font-size: 24px;">🎨</span>
                                    <strong style="background: linear-gradient(135deg, #8b5cf6 0%, #ec4899 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; font-size: 16px; text-transform: uppercase; letter-spacing: 0.08em; font-weight: 800;">Absurdist Meme</strong>
                                </div>
                                <span style="font-size: 9px; color: #999; font-weight: 600; letter-spacing: 0.05em;">BUILT WITH GEMINI + VERTEX AI</span>
                            </div>
                            <div style="background: rgba(255, 255, 255, 0.9); padding: 15px; border-radius: 8px; border: 2px solid rgba(139, 92, 246, 0.3); text-align: center;">
                                <img src="${errorLog.meme_url}" alt="AI-Generated Absurdist Meme" style="max-width: 100%; height: auto; border-radius: 8px; border: 3px solid rgba(139, 92, 246, 0.3); box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);">
                            </div>
                        </div>
                    ` + "`" + ` : ''}
                ` + "`" + `;

                // Add Rorschach test if available
                if (errorLog.rorschach_image_number) {
                    const rorschachHTML = ` + "`" + `
                        <div style="margin-top: 20px; padding: 20px; background: linear-gradient(135deg, rgba(99, 102, 241, 0.08) 0%, rgba(139, 92, 246, 0.12) 100%); border: 3px solid rgba(99, 102, 241, 0.4); border-radius: 12px; box-shadow: 0 0 25px rgba(99, 102, 241, 0.25), 4px 4px 0px rgba(99, 102, 241, 0.15);">
                            <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 15px;">
                                <div style="display: flex; align-items: center; gap: 10px;">
                                    <span style="font-size: 24px;">🎭</span>
                                    <strong style="color: #6366f1; font-size: 16px; text-transform: uppercase; letter-spacing: 0.08em; font-weight: 800;">Rorschach Test Card #${errorLog.rorschach_image_number}</strong>
                                </div>
                                <span style="font-size: 9px; color: #999; font-weight: 600; letter-spacing: 0.05em;">POWERED BY OPENAI</span>
                            </div>
                            <div style="background: white; padding: 20px; border-radius: 8px; border: 2px solid rgba(99, 102, 241, 0.3); text-align: center;">
                                <img src="${errorLog.rorschach_image_url}" alt="Rorschach Test Card ${errorLog.rorschach_image_number}" style="max-width: 400px; width: 100%; height: auto; display: block; margin: 0 auto;">
                            </div>
                            <div id="rorschach-interpretation-${errorLog.id}" style="margin-top: 15px;">
                                ${errorLog.rorschach_ai_response ? ` + "`" + `
                                    <div style="background: rgba(255, 255, 255, 0.8); padding: 15px; border-radius: 8px; border: 2px solid rgba(99, 102, 241, 0.3); margin-bottom: 10px;">
                                        <strong style="color: #6366f1; font-size: 14px;">💭 Patient Response:</strong>
                                        <p style="margin-top: 8px; color: #374151; line-height: 1.6; font-size: 14px; font-style: italic;">${errorLog.rorschach_ai_response}</p>
                                    </div>
                                ` + "`" + ` : (pendingRorschachInterpretations.has(errorLog.id) ? ` + "`" + `
                                    <button disabled id="interpret-btn-${errorLog.id}" style="width: 100%; padding: 12px 20px; background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%); color: white; border: none; border-radius: 8px; font-weight: 700; font-size: 14px; cursor: not-allowed; box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3); text-transform: uppercase; letter-spacing: 0.5px; opacity: 0.7;">
                                        <span style="display: inline-block; animation: spin 1s linear infinite;">🧠</span> Analyzing...
                                    </button>
                                ` + "`" + ` : ` + "`" + `
                                    <button onclick="requestRorschachInterpretation('${errorLog.id}')" id="interpret-btn-${errorLog.id}" style="width: 100%; padding: 12px 20px; background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%); color: white; border: none; border-radius: 8px; font-weight: 700; font-size: 14px; cursor: pointer; box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3); transition: all 0.2s; text-transform: uppercase; letter-spacing: 0.5px;">
                                        Generate Patient Response
                                    </button>
                                ` + "`" + `)}
                                ${errorLog.rorschach_user_response ? ` + "`" + `
                                    <div style="background: rgba(255, 255, 255, 0.8); padding: 15px; border-radius: 8px; border: 2px solid rgba(16, 185, 129, 0.3); margin-top: 10px;">
                                        <strong style="color: #10b981; font-size: 14px;">💭 Your Response:</strong>
                                        <p style="margin-top: 8px; color: #374151; line-height: 1.6; font-size: 14px;">${errorLog.rorschach_user_response}</p>
                                    </div>
                                ` + "`" + ` : (isLoggedIn ? ` + "`" + `
                                    <button onclick="showRorschachResponseForm('${errorLog.id}')" id="respond-btn-${errorLog.id}" style="width: 100%; padding: 12px 20px; background: linear-gradient(135deg, #10b981 0%, #059669 100%); color: white; border: none; border-radius: 8px; font-weight: 700; font-size: 14px; cursor: pointer; box-shadow: 0 4px 12px rgba(16, 185, 129, 0.3); transition: all 0.2s; text-transform: uppercase; letter-spacing: 0.5px; margin-top: 10px;">
                                        Respond to Image
                                    </button>
                                    <div id="response-form-${errorLog.id}" style="display: none; margin-top: 10px;">
                                        <textarea id="response-text-${errorLog.id}" placeholder="What do you see in this image?" style="width: 100%; padding: 12px; border: 2px solid rgba(16, 185, 129, 0.3); border-radius: 8px; font-size: 14px; font-family: inherit; resize: vertical; min-height: 100px;"></textarea>
                                        <div style="display: flex; gap: 10px; margin-top: 10px;">
                                            <button onclick="submitRorschachResponse('${errorLog.id}')" style="flex: 1; padding: 10px; background: #10b981; color: white; border: none; border-radius: 6px; font-weight: 600; cursor: pointer;">Submit</button>
                                            <button onclick="hideRorschachResponseForm('${errorLog.id}')" style="flex: 1; padding: 10px; background: #6b7280; color: white; border: none; border-radius: 6px; font-weight: 600; cursor: pointer;">Cancel</button>
                                        </div>
                                    </div>
                                ` + "`" + ` : '')}
                            </div>
                        </div>
                    ` + "`" + `;

                    div.innerHTML += rorschachHTML;
                }

                // Add GIFs if available (support both single and multiple GIFs)
                const gifURLs = errorLog.gif_urls || (errorLog.gif_url ? [errorLog.gif_url] : []);
                if (gifURLs.length > 0) {
                    const gifContainerID = 'gif-container-' + errorLog.id;
                    const expandButtonID = 'gif-expand-btn-' + errorLog.id;

                    let gifHTML = '<div style="margin-top: 20px; padding: 20px; background: linear-gradient(135deg, rgba(239, 68, 68, 0.08) 0%, rgba(220, 38, 38, 0.12) 100%); border: 3px solid rgba(239, 68, 68, 0.4); border-radius: 12px; box-shadow: 0 0 25px rgba(239, 68, 68, 0.25), 4px 4px 0px rgba(239, 68, 68, 0.15);">';
                    gifHTML += '<div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 15px;"><div style="display: flex; align-items: center; gap: 10px;"><span style="font-size: 24px;">🎬</span><strong style="color: #ef4444; font-size: 16px; text-transform: uppercase; letter-spacing: 0.08em; font-weight: 800;">Animated Reaction GIF' + (gifURLs.length > 1 ? 'S (' + gifURLs.length + ')' : '') + '</strong></div><span style="font-size: 10px; color: #999; font-weight: 600; letter-spacing: 0.05em;">POWERED BY GIPHY</span></div>';
                    gifHTML += '<div id="' + gifContainerID + '" style="display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 15px;">';

                    // Display first 2 GIFs by default (no action endpoint trigger)
                    const initialGifCount = Math.min(2, gifURLs.length);
                    for (let i = 0; i < initialGifCount; i++) {
                        gifHTML += '<div style="border: 4px solid rgba(239, 68, 68, 0.5); border-radius: 12px; padding: 6px; background: rgba(255, 255, 255, 0.6); box-shadow: inset 0 0 20px rgba(239, 68, 68, 0.15);"><img src="' + gifURLs[i] + '" alt="Reaction GIF ' + (i + 1) + '" style="width: 100%; border-radius: 8px; display: block; object-fit: contain;"></div>';
                    }

                    // Add hidden GIFs (will be shown on expansion)
                    for (let i = 2; i < gifURLs.length; i++) {
                        gifHTML += '<div id="gif-' + errorLog.id + '-' + i + '" style="display: none; border: 4px solid rgba(239, 68, 68, 0.5); border-radius: 12px; padding: 6px; background: rgba(255, 255, 255, 0.6); box-shadow: inset 0 0 20px rgba(239, 68, 68, 0.15);"><img data-src="' + gifURLs[i] + '" alt="Reaction GIF ' + (i + 1) + '" style="width: 100%; border-radius: 8px; display: block; object-fit: contain;"></div>';
                    }

                    gifHTML += '</div>';

                    // Add expansion button if more than 2 GIFs
                    if (gifURLs.length > 2) {
                        const remainingCount = gifURLs.length - 2;
                        gifHTML += '<button id="' + expandButtonID + '" data-total-gifs="' + gifURLs.length + '" onclick="toggleGifExpansion(\'' + errorLog.id + '\', ' + gifURLs.length + ')" style="margin-top: 15px; padding: 10px 20px; background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%); color: white; border: none; border-radius: 24px; font-weight: 700; font-size: 13px; letter-spacing: 0.05em; text-transform: uppercase; cursor: pointer; box-shadow: 0 4px 12px rgba(239, 68, 68, 0.3); transition: all 0.2s;">Show More GIFs (' + remainingCount + ') ▼</button>';
                    }

                    gifHTML += '</div>';
                    div.innerHTML += gifHTML;
                }

                // Add Spotify if available
                if (errorLog.song_url) {
                    const songTitleID = 'song-title-' + errorLog.id;
                    let spotifyHTML = '<div style="margin-top: 20px; padding: 15px; background: linear-gradient(135deg, rgba(29, 185, 84, 0.08) 0%, rgba(30, 215, 96, 0.12) 100%); border: 3px solid rgba(29, 185, 84, 0.4); border-radius: 12px; box-shadow: 0 0 25px rgba(29, 185, 84, 0.25), 4px 4px 0px rgba(29, 185, 84, 0.15);"><div style="display: flex; align-items: center; justify-content: space-between;"><div><div style="display: flex; align-items: center; gap: 10px; margin-bottom: 5px;"><span style="font-size: 24px;">🎵</span><strong style="color: #1db954; font-size: 14px; text-transform: uppercase; letter-spacing: 0.08em; font-weight: 800;">Recommended Track</strong><span style="font-size: 9px; color: #999; font-weight: 600; letter-spacing: 0.05em; margin-left: 10px;">POWERED BY SPOTIFY</span></div>';
                    if (errorLog.song_title && errorLog.song_artist) {
                        spotifyHTML += '<div style="margin-left: 34px;"><div id="' + songTitleID + '" style="font-size: 15px; font-weight: 600; color: #1a1a1a; cursor: pointer; user-select: none;">' + errorLog.song_title + '</div><div style="font-size: 13px; color: #666; margin-top: 2px;">' + errorLog.song_artist + '</div></div>';
                    }
                    spotifyHTML += '</div><a href="' + errorLog.song_url + '" target="_blank" rel="noopener noreferrer" style="padding: 10px 20px; background: #1db954; color: white; text-decoration: none; border-radius: 24px; font-weight: 700; font-size: 13px; letter-spacing: 0.05em; text-transform: uppercase; white-space: nowrap; transition: all 0.2s; box-shadow: 0 4px 12px rgba(29, 185, 84, 0.3);">Play ▶</a></div></div>';
                    div.innerHTML += spotifyHTML;

                    // Add Easter egg: 4 clicks triggers CIA Spy Kids image then redirects to Spotify
                    setTimeout(() => {
                        const songTitleElement = document.getElementById(songTitleID);
                        if (songTitleElement) {
                            let clickCount = 0;
                            const maxClicks = 4;
                            let clickTimer = null;

                            songTitleElement.addEventListener('click', function(e) {
                                e.preventDefault();
                                clickCount++;

                                // Reset click count after 2 seconds of inactivity
                                clearTimeout(clickTimer);
                                clickTimer = setTimeout(() => {
                                    clickCount = 0;
                                }, 2000);

                                if (clickCount === maxClicks) {
                                    // Trigger Easter egg
                                    triggerSpyKidsEasterEgg(errorLog.song_url);
                                    clickCount = 0;
                                }
                            });
                        }
                    }, 0);
                }

                // Add food image if available
                if (errorLog.food_image_url) {
                    const foodAttr = errorLog.food_image_attr || 'Stock food photography';
                    div.innerHTML += '<div style="margin-top: 20px; padding: 20px; background: linear-gradient(135deg, rgba(165, 94, 234, 0.08) 0%, rgba(179, 0, 255, 0.12) 100%); border: 3px solid rgba(165, 94, 234, 0.4); border-radius: 12px; box-shadow: 0 0 25px rgba(165, 94, 234, 0.25), 4px 4px 0px rgba(165, 94, 234, 0.15);"><div style="display: flex; align-items: center; gap: 10px; margin-bottom: 15px;"><span style="font-size: 24px;">🍽️</span><strong style="color: #a55eea; font-size: 16px; text-transform: uppercase; letter-spacing: 0.08em; font-weight: 800;">Food Blog Imagery</strong></div><span style="font-size: 9px; color: #999; font-weight: 600; letter-spacing: 0.05em;">POWERED BY PEXELS</span></div><div style="border: 4px solid rgba(165, 94, 234, 0.5); border-radius: 12px; padding: 6px; background: rgba(255, 255, 255, 0.6); box-shadow: inset 0 0 20px rgba(165, 94, 234, 0.15);"><img src="' + errorLog.food_image_url + '" alt="Food blog image" style="width: 100%; max-width: 600px; border-radius: 8px; display: block; object-fit: cover;"></div><p style="font-size: 11px; color: #6b46c1; font-style: italic; margin-top: 12px; font-family: \'Courier New\', monospace; text-transform: uppercase; letter-spacing: 0.05em;">' + foodAttr + '</p></div>';
                }

                // Add anonymous tips if available
                if (errorLog.anonymous_tips && errorLog.anonymous_tips.length > 0) {
                    div.innerHTML += '<div style="margin-top: 20px; padding: 20px; background: linear-gradient(135deg, rgba(102, 126, 234, 0.08) 0%, rgba(118, 75, 162, 0.12) 100%); border: 3px solid rgba(102, 126, 234, 0.4); border-radius: 12px; box-shadow: 0 0 25px rgba(102, 126, 234, 0.25), 4px 4px 0px rgba(102, 126, 234, 0.15);"><div style="display: flex; align-items: center; gap: 10px; margin-bottom: 15px;"><span style="font-size: 24px;">🕵️</span><strong style="color: #667eea; font-size: 16px; text-transform: uppercase; letter-spacing: 0.08em; font-weight: 800;">Anonymous Not-Spy-Work Tips</strong></div><div id="tips-' + errorLog.id + '"></div></div>';
                }

                container.appendChild(div);

                // Track this error as displayed
                displayedErrorIds.add(errorLog.id);

                // Fetch and display tip details if tips are present
                if (errorLog.anonymous_tips && errorLog.anonymous_tips.length > 0) {
                    (async () => {
                        const tipsContainer = document.getElementById('tips-' + errorLog.id);
                        for (const tipID of errorLog.anonymous_tips) {
                            try {
                                const tipRes = await fetch('/api/tips/' + tipID);
                                if (tipRes.ok) {
                                    const tip = await tipRes.json();
                                    const tipDiv = document.createElement('div');
                                    tipDiv.style.cssText = 'background: rgba(255, 255, 255, 0.8); padding: 15px; border-radius: 8px; margin-bottom: 12px; border: 2px solid rgba(102, 126, 234, 0.3);';

                                    const tipTime = new Date(tip.timestamp).toLocaleString();
                                    const moderatedBadge = tip.moderation_status === 'redacted' ? '<span style="background: #fbbf24; color: #78350f; padding: 3px 8px; border-radius: 4px; font-size: 10px; font-weight: 700; text-transform: uppercase; margin-left: 10px;">Redacted</span>' : '';

                                    tipDiv.innerHTML = '<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px;"><code style="background: rgba(102, 126, 234, 0.15); padding: 4px 8px; border-radius: 4px; font-size: 11px; color: #4338ca; font-weight: 600;">' + tip.user_hash + '</code><span style="font-size: 11px; color: #6b7280; font-family: \'Courier New\', monospace;">' + tipTime + moderatedBadge + '</span></div><p style="font-family: inherit; line-height: 1.6; color: #1f2937; margin: 0;">' + tip.moderated_content + '</p>';
                                    tipsContainer.appendChild(tipDiv);
                                }
                            } catch (e) {
                                console.error('Error fetching tip:', e);
                            }
                        }
                    })();
                }

                // Restore expanded state if it was previously expanded
                const savedState = expandedCollapsibles.get(errorLog.timestamp);
                if (savedState) {
                    if (savedState.satirical) {
                        const satiricalContent = div.querySelector('.collapsible-content.satirical-fix');
                        const satiricalToggle = div.querySelector('.collapsible-content.satirical-fix').previousElementSibling.querySelector('.collapsible-toggle');
                        if (satiricalContent && satiricalToggle) {
                            satiricalContent.classList.add('expanded');
                            satiricalToggle.classList.add('expanded');
                        }
                    }
                    if (savedState.story) {
                        const storyContent = div.querySelector('.collapsible-content.story');
                        const storyToggle = div.querySelector('.collapsible-content.story').previousElementSibling.querySelector('.collapsible-toggle');
                        if (storyContent && storyToggle) {
                            storyContent.classList.add('expanded');
                            storyToggle.classList.add('expanded');
                        }
                    }
                    // Restore GIF expansion state
                    if (savedState.gifsExpanded) {
                        const gifExpandButton = div.querySelector('[id^="gif-expand-btn-"]');
                        if (gifExpandButton) {
                            // Extract error log ID and total GIFs from button ID
                            const buttonId = gifExpandButton.id;
                            const errorLogID = buttonId.replace('gif-expand-btn-', '');
                            const totalGifs = parseInt(gifExpandButton.getAttribute('data-total-gifs') || '0');

                            // Expand the GIFs
                            if (totalGifs > 2) {
                                for (let i = 2; i < totalGifs; i++) {
                                    const gifDiv = document.getElementById('gif-' + errorLogID + '-' + i);
                                    if (gifDiv) {
                                        gifDiv.style.display = 'block';
                                        // Load the image if not already loaded
                                        const img = gifDiv.querySelector('img');
                                        if (img && img.dataset.src && !img.src) {
                                            img.src = img.dataset.src;
                                        }
                                    }
                                }
                                gifExpandButton.textContent = 'Show Less ▲';
                                gifExpandButton.dataset.expanded = 'true';
                            }
                        }
                    }
                }
            }
        }

        // Refresh commercial real estate
        async function refreshCommercialRealEstate() {
            try {
                const res = await fetch('/api/commercialrealestate');
                if (!res.ok) {
                    return;
                }

                const commercialRealEstate = await res.json();
                displayCommercialRealEstate(commercialRealEstate);
            } catch (e) {
                console.error('Error fetching commercial real estate:', e);
            }
        }

        // Display commercial real estate
        // Store all businesses for pagination
        let allBusinessesData = {};
        let visibleBusinessCount = 0;
        const BUSINESS_PAGE_SIZE = 11;

        function displayCommercialRealEstate(commercialRealEstateMap) {
            const container = document.getElementById('commercialrealestate');

            // Convert new format (map of types.CommercialRealEstate records) to old format for compatibility
            allBusinessesData = {};
            let allGoverningBodies = [];

            for (const [locationName, record] of Object.entries(commercialRealEstateMap || {})) {
                if (record.properties && record.properties.length > 0) {
                    allBusinessesData[locationName] = record.properties;
                }
                if (record.governing_bodies && record.governing_bodies.length > 0) {
                    allGoverningBodies = allGoverningBodies.concat(record.governing_bodies);
                }
            }

            if (Object.keys(allBusinessesData).length === 0 && allGoverningBodies.length === 0) {
                container.innerHTML = '<div class="empty-state" style="padding: 40px 20px;">' +
                    '<p style="color: #6b7280;">No commercial real estate data yet</p>' +
                    '<p style="font-size: 14px; margin-top: 10px; color: #9ca3af;">' +
                    'Share a location to discover commercial properties and businesses in your area (queried randomly within 10 miles)' +
                    '</p></div>';
                return;
            }

            // Initial display - show first 11 businesses
            visibleBusinessCount = 0;
            container.innerHTML = '';
            renderBusinesses(container, allGoverningBodies);
        }

        function renderBusinesses(container, governingBodies = []) {
            let currentCount = 0;

            for (const [locationName, properties] of Object.entries(allBusinessesData)) {
                for (const prop of properties) {
                    if (currentCount < visibleBusinessCount + BUSINESS_PAGE_SIZE) {
                        if (currentCount >= visibleBusinessCount) {
                            const propDiv = createBusinessCard(locationName, prop);
                            container.appendChild(propDiv);
                        }
                        currentCount++;
                    }
                }
            }

            // Remove existing load more button and governing bodies section
            const existingBtn = container.querySelector('.load-more-btn');
            if (existingBtn) existingBtn.remove();
            const existingGov = container.querySelector('.governing-bodies-section');
            if (existingGov) existingGov.remove();

            // Add load more button if there are more businesses
            const totalBusinesses = Object.values(allBusinessesData).reduce((sum, props) => sum + props.length, 0);
            if (currentCount > visibleBusinessCount + BUSINESS_PAGE_SIZE) {
                const loadMoreBtn = document.createElement('button');
                loadMoreBtn.className = 'load-more-btn';
                loadMoreBtn.textContent = '📄 Load More Businesses (' + (totalBusinesses - (visibleBusinessCount + BUSINESS_PAGE_SIZE)) + ' remaining)';
                loadMoreBtn.style.cssText = 'width: 100%; margin-top: 15px; padding: 15px; background: linear-gradient(135deg, var(--memphis-purple) 0%, var(--pop-purple-neon) 100%); color: var(--swiss-white); border: 2px solid var(--swiss-black); border-radius: 8px; font-size: 14px; font-weight: 700; text-transform: uppercase; cursor: pointer; transition: all 0.3s; box-shadow: 4px 4px 0px rgba(0, 0, 0, 0.2);';
                loadMoreBtn.onclick = () => loadMoreBusinesses(governingBodies);
                container.appendChild(loadMoreBtn);
            }

            // Display governing bodies at the bottom
            if (governingBodies && governingBodies.length > 0) {
                const govSection = document.createElement('div');
                govSection.className = 'governing-bodies-section';
                govSection.style.cssText = 'margin-top: 25px; padding-top: 20px; border-top: 2px solid #8b5cf6;';

                const govHeader = document.createElement('h3');
                govHeader.textContent = '🏛️  Local Governing Authorities';
                govHeader.style.cssText = 'color: #8b5cf6; font-size: 16px; margin-bottom: 15px; font-weight: 700;';
                govSection.appendChild(govHeader);

                governingBodies.forEach(body => {
                    const bodyCard = createGoverningBodyCard(body);
                    govSection.appendChild(bodyCard);
                });

                container.appendChild(govSection);
            }

            visibleBusinessCount = currentCount;
        }

        function loadMoreBusinesses(governingBodies) {
            const container = document.getElementById('commercialrealestate');
            renderBusinesses(container, governingBodies);
        }

        function createBusinessCard(locationName, prop) {
            const statusColor = prop.status === 'available' ? '#059669' : prop.status === 'leased' ? '#dc2626' : '#2563eb';
            const statusIcon = prop.status === 'available' ? '🟢' : prop.status === 'leased' ? '🔴' : '🔵';

            let contactHTML = '';
            if (prop.contact_info) {
                if (prop.contact_info.phone) {
                    contactHTML += '<div style="margin-top: 4px; font-size: 13px;">📞 ' + prop.contact_info.phone + '</div>';
                }
                if (prop.contact_info.email) {
                    contactHTML += '<div style="margin-top: 4px; font-size: 13px;">✉️ <a href="mailto:' + prop.contact_info.email + '" style="color: #667eea;">' + prop.contact_info.email + '</a></div>';
                }
                if (prop.contact_info.website) {
                    contactHTML += '<div style="margin-top: 4px; font-size: 13px;">🔗 <a href="' + prop.contact_info.website + '" target="_blank" style="color: #667eea; text-decoration: none;">' + prop.contact_info.website + '</a></div>';
                }
            }

            let detailsHTML = '';
            if (prop.square_footage) {
                detailsHTML += '<div style="margin-top: 4px; font-size: 13px;">📏 ' + prop.square_footage + '</div>';
            }
            if (prop.price_info) {
                detailsHTML += '<div style="margin-top: 4px; font-size: 13px;">💰 ' + prop.price_info + '</div>';
            }
            if (prop.current_business) {
                detailsHTML += '<div style="margin-top: 4px; font-size: 13px;"><strong>Current Business:</strong> ' + prop.current_business + (prop.business_type ? ' (' + prop.business_type + ')' : '') + '</div>';
            }

            const div = document.createElement('div');
            div.className = 'location-card';
            div.style.borderLeft = '4px solid #8b5cf6';
            div.style.marginBottom = '15px';

            div.innerHTML = '<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px;">' +
                '<h3 style="margin: 0; color: #8b5cf6; font-size: 14px;">📍 ' + locationName + '</h3>' +
                '</div>' +
                '<div style="padding: 12px; background: #f9fafb; border-radius: 6px;">' +
                '<div style="display: flex; align-items: center; gap: 8px; margin-bottom: 6px;">' +
                '<span>' + statusIcon + '</span>' +
                '<strong style="color: #1f2937; font-size: 14px;">' + prop.address + '</strong>' +
                '<span style="background: ' + statusColor + '20; color: ' + statusColor + '; padding: 2px 8px; border-radius: 4px; font-size: 11px; text-transform: uppercase;">' + prop.status + '</span>' +
                '</div>' +
                '<div style="color: #8b5cf6; font-size: 12px; text-transform: uppercase; font-weight: 600; margin-bottom: 6px;">' + prop.property_type + '</div>' +
                (prop.description ? '<div style="color: #6b7280; font-size: 13px; margin-bottom: 6px;">' + prop.description + '</div>' : '') +
                detailsHTML +
                contactHTML +
                '</div>';

            return div;
        }

        function createGoverningBodyCard(body) {
            const typeColors = {
                'city_council': '#3b82f6',
                'planning': '#10b981',
                'zoning': '#f59e0b',
                'civic': '#8b5cf6'
            };
            const typeIcons = {
                'city_council': '🏛️',
                'planning': '📋',
                'zoning': '🗺️',
                'civic': '🤝'
            };

            const typeColor = typeColors[body.type] || '#6b7280';
            const typeIcon = typeIcons[body.type] || '🏢';

            const div = document.createElement('div');
            div.className = 'location-card';
            div.style.borderLeft = '4px solid ' + typeColor;
            div.style.marginBottom = '15px';

            div.innerHTML = '<div style="padding: 12px; background: #f9fafb; border-radius: 6px;">' +
                '<div style="display: flex; align-items: center; gap: 8px; margin-bottom: 6px;">' +
                '<span>' + typeIcon + '</span>' +
                '<strong style="color: ' + typeColor + '; font-size: 14px;">' + body.name + '</strong>' +
                '<span style="background: ' + typeColor + '20; color: ' + typeColor + '; padding: 2px 8px; border-radius: 4px; font-size: 11px; text-transform: uppercase;">' + body.type.replace('_', ' ') + '</span>' +
                '</div>' +
                (body.jurisdiction ? '<div style="color: #6b7280; font-size: 13px; margin-bottom: 6px;">📍 ' + body.jurisdiction + '</div>' : '') +
                (body.contact ? '<div style="margin-top: 6px; font-size: 13px;">📞 ' + body.contact + '</div>' : '') +
                '</div>';

            return div;
        }

        // Toggle collapsible sections
        function toggleCollapsible(headerElement) {
            const toggle = headerElement.querySelector('.collapsible-toggle');
            const content = headerElement.nextElementSibling;

            if (content.classList.contains('expanded')) {
                content.classList.remove('expanded');
                toggle.classList.remove('expanded');
            } else {
                content.classList.add('expanded');
                toggle.classList.add('expanded');
            }
        }

        // Share error log to Facebook with compilation image
        async function shareFacebookCompilation(errorID) {
            const btn = event.target.closest('button');
            const originalHTML = btn.innerHTML;

            // Show loading state
            btn.disabled = true;
            btn.innerHTML = '<svg style="width: 18px; height: 18px; animation: spin 1s linear infinite;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" stroke-width="4" stroke-opacity="0.25"/><path d="M12 2a10 10 0 0 1 10 10" stroke-width="4" stroke-linecap="round"/></svg> Generating...';

            try {
                // Call the Facebook share API with just the error ID
                const response = await fetch('/api/facebook-share/' + errorID);

                if (!response.ok) {
                    throw new Error('Failed to generate share image');
                }

                const data = await response.json();

                // Success state
                btn.innerHTML = '✓ Opening Facebook...';
                btn.style.background = 'linear-gradient(135deg, #10b981 0%, #059669 100%)';

                // Open Facebook share dialog
                window.open(data.direct_share_url, '_blank', 'width=600,height=400');

                // Reset button after delay
                setTimeout(() => {
                    btn.disabled = false;
                    btn.innerHTML = originalHTML;
                    btn.style.background = '';
                }, 3000);

            } catch (error) {
                console.error('Facebook share error:', error);
                btn.innerHTML = '✗ Failed - Try Again';
                btn.style.background = 'linear-gradient(135deg, #ef4444 0%, #dc2626 100%)';

                setTimeout(() => {
                    btn.disabled = false;
                    btn.innerHTML = originalHTML;
                    btn.style.background = '';
                }, 3000);
            }
        }

        // Request AI interpretation of Rorschach image
        async function requestRorschachInterpretation(errorID) {
            // Mark as pending so spinner persists across refreshes
            pendingRorschachInterpretations.add(errorID);

            const btn = document.getElementById('interpret-btn-' + errorID);
            const originalHTML = btn.innerHTML;

            btn.disabled = true;
            btn.innerHTML = '<span style="display: inline-block; animation: spin 1s linear infinite;">🧠</span> Analyzing...';

            try {
                const response = await fetch('/api/rorschach/interpret/' + errorID, {
                    method: 'POST',
                });

                if (!response.ok) {
                    throw new Error('Failed to generate interpretation');
                }

                const data = await response.json();

                // Remove from pending set
                pendingRorschachInterpretations.delete(errorID);

                // Update UI with patient response
                const interpretationDiv = document.getElementById('rorschach-interpretation-' + errorID);
                interpretationDiv.innerHTML = ` + "`" + `
                    <div style="background: rgba(255, 255, 255, 0.8); padding: 15px; border-radius: 8px; border: 2px solid rgba(99, 102, 241, 0.3); margin-bottom: 10px; animation: fadeIn 0.5s;">
                        <strong style="color: #6366f1; font-size: 14px;">💭 Patient Response:</strong>
                        <p style="margin-top: 8px; color: #374151; line-height: 1.6; font-size: 14px; font-style: italic;">${data.interpretation}</p>
                    </div>
                    ${isLoggedIn ? ` + "`" + `
                        <button onclick="showRorschachResponseForm('${errorID}')" id="respond-btn-${errorID}" style="width: 100%; padding: 12px 20px; background: linear-gradient(135deg, #10b981 0%, #059669 100%); color: white; border: none; border-radius: 8px; font-weight: 700; font-size: 14px; cursor: pointer; box-shadow: 0 4px 12px rgba(16, 185, 129, 0.3); transition: all 0.2s; text-transform: uppercase; letter-spacing: 0.5px; margin-top: 10px;">
                            Respond to Image
                        </button>
                        <div id="response-form-${errorID}" style="display: none; margin-top: 10px;">
                            <textarea id="response-text-${errorID}" placeholder="What do you see in this image?" style="width: 100%; padding: 12px; border: 2px solid rgba(16, 185, 129, 0.3); border-radius: 8px; font-size: 14px; font-family: inherit; resize: vertical; min-height: 100px;"></textarea>
                            <div style="display: flex; gap: 10px; margin-top: 10px;">
                                <button onclick="submitRorschachResponse('${errorID}')" style="flex: 1; padding: 10px; background: #10b981; color: white; border: none; border-radius: 6px; font-weight: 600; cursor: pointer;">Submit</button>
                                <button onclick="hideRorschachResponseForm('${errorID}')" style="flex: 1; padding: 10px; background: #6b7280; color: white; border: none; border-radius: 6px; font-weight: 600; cursor: pointer;">Cancel</button>
                            </div>
                        </div>
                    ` + "`" + ` : ''}
                ` + "`" + `;

            } catch (error) {
                console.error('Rorschach interpretation error:', error);
                pendingRorschachInterpretations.delete(errorID);
                btn.disabled = false;
                btn.innerHTML = '✗ Failed - Try Again';
                setTimeout(() => {
                    btn.innerHTML = originalHTML;
                }, 3000);
            }
        }

        // Show Rorschach response form
        function showRorschachResponseForm(errorID) {
            document.getElementById('respond-btn-' + errorID).style.display = 'none';
            document.getElementById('response-form-' + errorID).style.display = 'block';
        }

        // Hide Rorschach response form
        function hideRorschachResponseForm(errorID) {
            document.getElementById('respond-btn-' + errorID).style.display = 'block';
            document.getElementById('response-form-' + errorID).style.display = 'none';
        }

        // Submit user's Rorschach response
        async function submitRorschachResponse(errorID) {
            const textarea = document.getElementById('response-text-' + errorID);
            const response = textarea.value.trim();

            if (!response) {
                alert('Please enter a response');
                return;
            }

            try {
                const res = await fetch('/api/rorschach/respond/' + errorID, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ response: response }),
                });

                if (!response.ok) {
                    throw new Error('Failed to submit response');
                }

                // Update UI with user response
                const interpretationDiv = document.getElementById('rorschach-interpretation-' + errorID);
                const existingInterpretation = interpretationDiv.querySelector('div').outerHTML;
                interpretationDiv.innerHTML = existingInterpretation + ` + "`" + `
                    <div style="background: rgba(255, 255, 255, 0.8); padding: 15px; border-radius: 8px; border: 2px solid rgba(16, 185, 129, 0.3); margin-top: 10px; animation: fadeIn 0.5s;">
                        <strong style="color: #10b981; font-size: 14px;">💭 Your Response:</strong>
                        <p style="margin-top: 8px; color: #374151; line-height: 1.6; font-size: 14px;">${response}</p>
                    </div>
                ` + "`" + `;

                textarea.value = '';

            } catch (error) {
                console.error('Rorschach response error:', error);
                alert('Failed to submit response. Please try again.');
            }
        }

        // Toggle GIF expansion (show/hide additional GIFs)
        function toggleGifExpansion(errorLogID, totalGifs) {
            const button = document.getElementById('gif-expand-btn-' + errorLogID);
            const isExpanded = button.dataset.expanded === 'true';

            if (isExpanded) {
                // Collapse: hide GIFs 3-8
                for (let i = 2; i < totalGifs; i++) {
                    const gifDiv = document.getElementById('gif-' + errorLogID + '-' + i);
                    if (gifDiv) {
                        gifDiv.style.display = 'none';
                    }
                }
                button.textContent = 'Show More GIFs (' + (totalGifs - 2) + ') ▼';
                button.dataset.expanded = 'false';
            } else {
                // Expand: show GIFs 3-8 and lazy load them
                for (let i = 2; i < totalGifs; i++) {
                    const gifDiv = document.getElementById('gif-' + errorLogID + '-' + i);
                    if (gifDiv) {
                        gifDiv.style.display = 'block';
                        // Lazy load the image
                        const img = gifDiv.querySelector('img');
                        if (img && img.dataset.src && !img.src) {
                            img.src = img.dataset.src;
                        }
                    }
                }
                button.textContent = 'Show Less ▲';
                button.dataset.expanded = 'true';
            }
        }

        // Easter egg: DOJ banner at top of error logs
        function showDOJBanner() {
            const errorlogsContainer = document.getElementById('errorlogs');
            if (!errorlogsContainer) return;

            // Check if banner already exists
            if (document.getElementById('doj-banner')) {
                // If it exists, remove it (toggle behavior)
                document.getElementById('doj-banner').remove();
                return;
            }

            // Create DOJ banner
            const banner = document.createElement('div');
            banner.id = 'doj-banner';
            banner.style.cssText = 'margin-bottom: 20px; border-radius: 12px; overflow: hidden; box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3); border: 3px solid rgba(0, 82, 204, 0.5); animation: dojFadeIn 0.5s ease-out; position: relative;';

            // Add fade-in animation
            const style = document.createElement('style');
            style.textContent = '@keyframes dojFadeIn { from { opacity: 0; transform: translateY(-20px); } to { opacity: 1; transform: translateY(0); } }';
            if (!document.getElementById('doj-banner-style')) {
                style.id = 'doj-banner-style';
                document.head.appendChild(style);
            }

            // Add close button
            const closeBtn = document.createElement('button');
            closeBtn.innerHTML = '×';
            closeBtn.style.cssText = 'position: absolute; top: 10px; right: 10px; background: rgba(0, 0, 0, 0.7); color: white; border: none; border-radius: 50%; width: 32px; height: 32px; font-size: 24px; cursor: pointer; z-index: 10; line-height: 1; padding: 0; transition: all 0.2s;';
            closeBtn.onmouseover = function() {
                this.style.background = 'rgba(0, 0, 0, 0.9)';
                this.style.transform = 'scale(1.1)';
            };
            closeBtn.onmouseout = function() {
                this.style.background = 'rgba(0, 0, 0, 0.7)';
                this.style.transform = 'scale(1)';
            };
            closeBtn.onclick = function() {
                banner.remove();
            };

            // Add image
            const img = document.createElement('img');
            img.src = 'https://www.justice.gov/d9/2025-07/blue_simple_quotes_twitter_header_2_0.png';
            img.alt = 'DOJ Banner';
            img.style.cssText = 'width: 100%; display: block; cursor: pointer;';
            img.onclick = function() {
                window.open('https://www.justice.gov/', '_blank');
            };

            banner.appendChild(closeBtn);
            banner.appendChild(img);

            // Insert at the top of error logs
            errorlogsContainer.insertBefore(banner, errorlogsContainer.firstChild);
        }

        // Easter egg: CIA Spy Kids image flash before Spotify redirect
        function triggerSpyKidsEasterEgg(spotifyURL) {
            // Create fullscreen overlay with loading spinner
            const overlay = document.createElement('div');
            overlay.id = 'spy-kids-overlay';
            overlay.style.cssText = 'position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0, 0, 0, 0.95); z-index: 10000; display: flex; flex-direction: column; align-items: center; justify-content: center;';

            // Add loading spinner
            const spinner = document.createElement('div');
            spinner.style.cssText = 'border: 8px solid rgba(255, 255, 255, 0.2); border-top: 8px solid #fff; border-radius: 50%; width: 60px; height: 60px; animation: spin 1s linear infinite;';
            overlay.appendChild(spinner);

            // Add spinner animation
            const style = document.createElement('style');
            style.textContent = '@keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }';
            document.head.appendChild(style);

            document.body.appendChild(overlay);

            // Preload the CIA Spy Kids image
            const ciaImageURL = 'https://www.cia.gov/spy-kids/static/25d41deb3dbe3106185c4fdac03ef3d9/5fb19/cia_seal_100x100.webp';
            const img = new Image();

            img.onload = function() {
                // Remove spinner
                spinner.remove();

                // Display CIA image briefly
                const imgElement = document.createElement('img');
                imgElement.src = ciaImageURL;
                imgElement.style.cssText = 'max-width: 90%; max-height: 90%; border-radius: 12px; box-shadow: 0 0 50px rgba(255, 255, 255, 0.5);';
                overlay.appendChild(imgElement);

                // After 800ms, redirect to Spotify
                setTimeout(() => {
                    window.open(spotifyURL, '_blank');
                    overlay.remove();
                }, 800);
            };

            img.onerror = function() {
                // If image fails to load, just redirect to Spotify
                setTimeout(() => {
                    window.open(spotifyURL, '_blank');
                    overlay.remove();
                }, 500);
            };

            img.src = ciaImageURL;
        }

        // ═══════════════════════════════════════════════════════════════
        // 🕵️ CONSOLE EASTER EGGS - HIDDEN INTERACTIVE PUZZLES
        // ═══════════════════════════════════════════════════════════════

        console.log('%c🕵️ SECRET AGENT CONSOLE 🕵️', 'font-size: 24px; font-weight: bold; color: #667eea;');
        console.log('%c═══════════════════════════════════════════════════════════════', 'color: #764ba2;');
        console.log('%cWelcome to the surveillance matrix, agent.', 'font-size: 14px; color: #333;');
        console.log('%c', 'font-size: 12px;');
        console.log('%cYour access level: PUBLIC (MONITORED)', 'color: red; font-weight: bold;');
        console.log('%cActive surveillance nodes: 847', 'color: #666;');
        console.log('%c', 'font-size: 12px;');
        console.log('%c🎮 INTERACTIVE CHALLENGE:', 'font-size: 14px; font-weight: bold; color: #667eea;');
        console.log('%cType the following command to unlock a secret message:', 'color: #333;');
        console.log('%c    window.revealSecret()', 'background: #f0f0f0; padding: 8px; font-family: monospace; color: #d63384;');
        console.log('%c', 'font-size: 12px;');
        console.log('%c💡 Bonus: Check the HTML source (View Page Source) for an encrypted puzzle!', 'color: #666; font-style: italic;');
        console.log('%c═══════════════════════════════════════════════════════════════', 'color: #764ba2;');

        window.revealSecret = function() {
            console.clear();
            console.log('%c', 'font-size: 20px;');
            console.log('%c    ███████╗███████╗ ██████╗██████╗ ███████╗████████╗', 'color: #667eea; font-family: monospace;');
            console.log('%c    ██╔════╝██╔════╝██╔════╝██╔══██╗██╔════╝╚══██╔══╝', 'color: #667eea; font-family: monospace;');
            console.log('%c    ███████╗█████╗  ██║     ██████╔╝█████╗     ██║   ', 'color: #667eea; font-family: monospace;');
            console.log('%c    ╚════██║██╔══╝  ██║     ██╔══██╗██╔══╝     ██║   ', 'color: #667eea; font-family: monospace;');
            console.log('%c    ███████║███████╗╚██████╗██║  ██║███████╗   ██║   ', 'color: #667eea; font-family: monospace;');
            console.log('%c    ╚══════╝╚══════╝ ╚═════╝╚═╝  ╚═╝╚══════╝   ╚═╝   ', 'color: #667eea; font-family: monospace;');
            console.log('%c', 'font-size: 20px;');
            console.log('%c🎉 CONGRATULATIONS! You found the hidden console puzzle!', 'font-size: 16px; color: green; font-weight: bold;');
            console.log('%c', 'font-size: 12px;');
            console.log('%c📜 SECRET MESSAGE:', 'font-size: 14px; font-weight: bold; color: #764ba2;');
            console.log('%c"The watchers are themselves watched. Every system of surveillance', 'color: #333; font-style: italic;');
            console.log('%ccontains within it the seeds of its own transparency. You have', 'color: #333; font-style: italic;');
            console.log('%cdemonstrated this by finding what was hidden in plain sight."', 'color: #333; font-style: italic;');
            console.log('%c', 'font-size: 12px;');
            console.log('%c🔑 ACHIEVEMENT UNLOCKED: "Console Spelunker"', 'background: gold; color: black; padding: 8px; font-weight: bold;');
            console.log('%c', 'font-size: 12px;');
            console.log('%c💡 Bonus Hint: Check the HTML source comments for another puzzle...', 'color: #666; font-style: italic;');
            console.log('%cTry the ROT13 cipher! Also check out window.surveillance', 'color: #666; font-style: italic;');
            console.log('%c', 'font-size: 12px;');
            console.log('%c👁️ Remember: You are not paranoid if they really are watching.', 'color: red; font-weight: bold;');
            return '🕵️ Secret revealed! Welcome to the inner circle, agent.';
        };

        // Hidden function for extra curious developers
        window.surveillance = {
            status: 'ACTIVE',
            targets: '[REDACTED]',
            errorLogsMonitored: Math.floor(Math.random() * 10000) + 1000,
            activeNodes: 847,
            dataCollected: '∞ petabytes',
            yourSession: {
                tracked: true,
                anonymized: false,
                dataShared: 'With partners',
                optOut: 'Not available'
            },
            lastUpdate: new Date().toISOString(),
            message: '👁️ The panopticon sees all. Type window.surveillance.bypass() to attempt escape.'
        };

        window.surveillance.bypass = function() {
            console.log('%c⚠️  ACCESS DENIED ⚠️', 'font-size: 20px; color: red; font-weight: bold;');
            console.log('%cNice try, agent. But you cannot simply walk out of the matrix.', 'color: red;');
            console.log('%cYour attempt has been logged and reported to the authorities.', 'color: red;');
            console.log('%c...', 'color: gray;');
            setTimeout(() => {
                console.log('%cJust kidding! 😄 There is no escape, but we appreciate the effort.', 'color: green; font-weight: bold;');
                console.log('%c', 'font-size: 12px;');
                console.log('%c"You can check out any time you like, but you can never leave."', 'color: #666; font-style: italic;');
                console.log('%c   - Hotel California (Eagles, 1976)', 'color: #999; font-style: italic; font-size: 11px;');
            }, 2000);
            return '🚫 Escape impossible. Welcome to Hotel California.';
        };

        // Hidden konami code easter egg
        let konamiCode = ['ArrowUp', 'ArrowUp', 'ArrowDown', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'ArrowLeft', 'ArrowRight', 'b', 'a'];
        let konamiPosition = 0;
        document.addEventListener('keydown', function(e) {
            if (e.key === konamiCode[konamiPosition]) {
                konamiPosition++;
                if (konamiPosition === konamiCode.length) {
                    console.log('%c🎮 KONAMI CODE ACTIVATED! 🎮', 'font-size: 20px; color: gold; font-weight: bold;');
                    console.log('%cYou have unlocked: ULTIMATE SURVEILLANCE MODE', 'color: red; font-weight: bold;');
                    console.log('%cJust kidding. But nice work, retro gamer! 👾', 'color: green;');
                    konamiPosition = 0;
                }
            } else {
                konamiPosition = 0;
            }
        });

        console.log('%c', 'font-size: 12px;');
        console.log('%c🎯 Pro tip: Try the Konami Code (↑↑↓↓←→←→BA)', 'color: #666; font-style: italic; font-size: 11px;');
    </script>

    <!-- types.Donation Modal -->
    <div id="donation-modal" style="display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0, 0, 0, 0.7); z-index: 10000; align-items: center; justify-content: center; padding: 20px;">
        <div style="background: white; border-radius: 16px; max-width: 500px; width: 100%; max-height: 90vh; overflow-y: auto; box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5); position: relative;">
            <div style="padding: 25px; border-bottom: 2px solid #e5e7eb;">
                <button onclick="closeDonationModal()" style="position: absolute; top: 15px; right: 15px; background: none; border: none; font-size: 24px; cursor: pointer; color: #6b7280; width: 32px; height: 32px; display: flex; align-items: center; justify-content: center; border-radius: 50%; transition: all 0.2s;">×</button>
                <h2 id="donation-modal-title" style="color: #000; font-size: 20px; font-weight: 800; margin-bottom: 12px;">Donation</h2>
                <div id="donation-modal-description" style="color: #4b5563; font-size: 14px; line-height: 1.6;"></div>
            </div>

            <form id="payment-form" onsubmit="handleDonationSubmit(event)" style="padding: 25px;">
                <div id="payment-element" style="margin-bottom: 20px;"></div>
                <button id="submit-donation" type="submit" style="width: 100%; padding: 15px; background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%); color: white; border: none; border-radius: 8px; font-size: 16px; font-weight: 700; cursor: pointer; box-shadow: 0 4px 6px rgba(59, 130, 246, 0.3); transition: all 0.2s;">
                    Complete Payment
                </button>
                <div id="donation-result"></div>
                <p style="color: #6b7280; font-size: 12px; margin-top: 15px; text-align: center; font-family: 'Courier New', monospace;">
                    🔒 Secure payment processed by Stripe
                </p>
            </form>
        </div>
    </div>
</body>
</html>`
