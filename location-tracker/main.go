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
	"math/big"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// Location represents a device location with timestamp
type Location struct {
	Latitude  float64   `json:"latitude" dynamodbav:"latitude"`
	Longitude float64   `json:"longitude" dynamodbav:"longitude"`
	Accuracy  float64   `json:"accuracy" dynamodbav:"accuracy"`
	Timestamp time.Time `json:"timestamp" dynamodbav:"timestamp"`
	DeviceID  string    `json:"device_id" dynamodbav:"device_id"`
	UserAgent string    `json:"user_agent" dynamodbav:"user_agent"`
}

// ErrorLog represents an error message with timestamp
type ErrorLog struct {
	ID                  string    `json:"id,omitempty" dynamodbav:"id"`
	Message             string    `json:"message" dynamodbav:"message"`
	GifURL              string    `json:"gif_url" dynamodbav:"gif_url"`
	Slogan              string    `json:"slogan" dynamodbav:"slogan"`
	SongTitle           string    `json:"song_title,omitempty" dynamodbav:"song_title"`
	SongArtist          string    `json:"song_artist,omitempty" dynamodbav:"song_artist"`
	SongURL             string    `json:"song_url,omitempty" dynamodbav:"song_url"`
	UserExperienceNote  string    `json:"user_experience_note,omitempty" dynamodbav:"user_experience_note"`
	UserNoteKeywords    []string  `json:"user_note_keywords,omitempty" dynamodbav:"user_note_keywords"`
	NearbyBusinesses    []string  `json:"nearby_businesses,omitempty" dynamodbav:"nearby_businesses"`
	Timestamp           time.Time `json:"timestamp" dynamodbav:"timestamp"`
}

// GoverningBody represents regulatory or private interest organizations with oversight responsibilities
type GoverningBody struct {
	ID               string                 `json:"id,omitempty" dynamodbav:"id"`
	OrganizationName string                 `json:"organization_name" dynamodbav:"organization_name"`
	GoverningBodies  []GoverningBodyDetails `json:"governing_bodies" dynamodbav:"governing_bodies"`
	Timestamp        time.Time              `json:"timestamp" dynamodbav:"timestamp"`
}

// GoverningBodyDetails stores flexible information about a governing authority
type GoverningBodyDetails struct {
	Name        string                 `json:"name" dynamodbav:"name"`
	Type        string                 `json:"type" dynamodbav:"type"` // "regulatory" or "private_interest"
	Description string                 `json:"description,omitempty" dynamodbav:"description"`
	Website     string                 `json:"website,omitempty" dynamodbav:"website"`
	ContactInfo map[string]interface{} `json:"contact_info,omitempty" dynamodbav:"contact_info"` // Flexible contact storage
	SourceData  map[string]interface{} `json:"source_data,omitempty" dynamodbav:"source_data"`   // Flexible additional data
}

// PerplexityRequest represents a request to Perplexity API
type PerplexityRequest struct {
	Model    string              `json:"model"`
	Messages []PerplexityMessage `json:"messages"`
}

// PerplexityMessage represents a message in Perplexity API format
type PerplexityMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// PerplexityResponse represents response from Perplexity API
type PerplexityResponse struct {
	Choices []struct {
		Message PerplexityMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Business represents a nearby business from Google Maps
type Business struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Address  string   `json:"address"`
	PlaceID  string   `json:"place_id"`
	Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"location"`
}

// Google Maps API response types
type GooglePlacesResponse struct {
	Results []struct {
		Name         string   `json:"name"`
		Types        []string `json:"types"`
		PlaceID      string   `json:"place_id"`
		FormattedAddress string `json:"formatted_address"`
		Geometry struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
	} `json:"results"`
	Status string `json:"status"`
}

// TwilioWebhook represents incoming SMS data from Twilio
type TwilioWebhook struct {
	MessageSid string `json:"MessageSid"`
	Body       string `json:"Body"`
	From       string `json:"From"`
	To         string `json:"To"`
}

var (
	// In-memory cache (locations expire after 24 hours)
	locations     = make(map[string]Location)
	locationMutex sync.RWMutex

	// Error log cache (keep last 50 errors)
	errorLogs     = make([]ErrorLog, 0, 50)
	errorLogMutex sync.RWMutex

	// Nearby businesses from last shared location
	currentBusinesses     = make([]Business, 0, 5)
	currentBusinessesMutex sync.RWMutex

	// Governing bodies cache (keyed by business name)
	governingBodiesCache     = make(map[string][]GoverningBodyDetails)
	governingBodiesCacheMutex sync.RWMutex

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
	errorLogsTableName      = "location-tracker-error-logs"
	locationsTableName      = "location-tracker-locations"
	governingBodiesTableName = "location-tracker-governing-bodies"
)

func main() {
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

	log.Printf("✅ Location tracker starting...")
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
	http.HandleFunc("/api/location", handleLocation)
	http.HandleFunc("/api/errorlogs", handleErrorLogs)
	http.HandleFunc("/api/businesses", handleBusinesses)
	http.HandleFunc("/api/keywords", handlePendingKeywords)
	http.HandleFunc("/api/governingbodies", handleGoverningBodies)
	http.HandleFunc("/api/health", handleHealth)
	http.HandleFunc("/api/twilio/sms", handleTwilioWebhook)

	// Start cleanup goroutine (remove locations older than 24h)
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

func fetchNearbyBusinesses(lat, lng float64) ([]Business, error) {
	if googleMapsAPIKey == "" {
		log.Println("⚠️  Google Maps API key not set, skipping business fetch")
		return []Business{}, nil
	}

	url := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/place/nearbysearch/json?location=%f,%f&radius=500&key=%s",
		lat, lng, googleMapsAPIKey,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch places: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google Maps API returned status: %d", resp.StatusCode)
	}

	var placesResp GooglePlacesResponse
	if err := json.NewDecoder(resp.Body).Decode(&placesResp); err != nil {
		return nil, fmt.Errorf("failed to decode places response: %w", err)
	}

	if placesResp.Status != "OK" && placesResp.Status != "ZERO_RESULTS" {
		return nil, fmt.Errorf("Google Maps API error: %s", placesResp.Status)
	}

	// Randomly select up to 5 businesses
	businesses := make([]Business, 0, 5)
	if len(placesResp.Results) > 0 {
		// Shuffle and take up to 5
		indices := make([]int, len(placesResp.Results))
		for i := range indices {
			indices[i] = i
		}

		// Simple shuffle
		for i := range indices {
			j := i + int(time.Now().UnixNano())%(len(indices)-i)
			indices[i], indices[j] = indices[j], indices[i]
		}

		count := 5
		if len(placesResp.Results) < count {
			count = len(placesResp.Results)
		}

		for i := 0; i < count; i++ {
			place := placesResp.Results[indices[i]]
			business := Business{
				Name:    place.Name,
				Type:    getBusinessType(place.Types),
				Address: place.FormattedAddress,
				PlaceID: place.PlaceID,
			}
			business.Location.Lat = place.Geometry.Location.Lat
			business.Location.Lng = place.Geometry.Location.Lng
			businesses = append(businesses, business)
		}

		log.Printf("🏢 Found %d nearby businesses", len(businesses))
	}

	return businesses, nil
}

func getBusinessType(types []string) string {
	// Return the first meaningful type
	for _, t := range types {
		if t != "point_of_interest" && t != "establishment" {
			return t
		}
	}
	if len(types) > 0 {
		return types[0]
	}
	return "business"
}

// extractKeywords extracts meaningful keywords from user notes for satirical purposes
func extractKeywords(userNote string) []string {
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

// searchGoverningBodies uses Perplexity API to find governing bodies for businesses in an area
func searchGoverningBodies(businessName string, businessType string, businessAddress string, userKeywords []string) ([]GoverningBodyDetails, error) {
	if perplexityAPIKey == "" {
		log.Println("⚠️  Perplexity API key not set, skipping governing body search")
		return []GoverningBodyDetails{}, nil
	}

	// Build satirical prompt that references user keywords if available
	keywordContext := ""
	if len(userKeywords) > 0 {
		keywordContext = fmt.Sprintf("\n\nFor satirical purposes, the user mentioned these keywords: %s. Feel free to acknowledge the absurd irony in the regulatory landscape.", strings.Join(userKeywords, ", "))
	}

	// Build business context
	businessContext := fmt.Sprintf("%s (type: %s)", businessName, businessType)
	if businessAddress != "" {
		businessContext += fmt.Sprintf(" located at %s", businessAddress)
	}

	prompt := fmt.Sprintf(`Find the governing authorities (both legal regulatory bodies and private interest organizations) responsible for oversight of this business: %s

Focus on returning:
1. Official website URLs for regulatory bodies
2. Contact information (phone, email, addresses) for these authorities
3. Names of local, state/provincial, and federal regulatory agencies
4. Industry associations and private oversight groups

Return the information in this JSON format:
{
  "governing_bodies": [
    {
      "name": "Authority Name",
      "type": "regulatory" or "private_interest",
      "description": "Brief description of their oversight role",
      "website": "https://...",
      "contact_info": {
        "phone": "...",
        "email": "...",
        "address": "..."
      }
    }
  ]
}%s

Return ONLY valid JSON, no additional text.`, businessContext, keywordContext)

	reqBody := PerplexityRequest{
		Model: "sonar",
		Messages: []PerplexityMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal perplexity request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.perplexity.ai/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create perplexity request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+perplexityAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call perplexity API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read perplexity response: %w", err)
	}

	var perplexityResp PerplexityResponse
	if err := json.Unmarshal(body, &perplexityResp); err != nil {
		return nil, fmt.Errorf("failed to parse perplexity response: %w", err)
	}

	if perplexityResp.Error != nil {
		return nil, fmt.Errorf("perplexity API error: %s", perplexityResp.Error.Message)
	}

	if len(perplexityResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in perplexity response")
	}

	// Parse the JSON response from Perplexity
	content := perplexityResp.Choices[0].Message.Content

	// Try to extract JSON from response (sometimes wrapped in markdown)
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		content = content[jsonStart : jsonEnd+1]
	}

	var result struct {
		GoverningBodies []GoverningBodyDetails `json:"governing_bodies"`
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		log.Printf("⚠️  Failed to parse governing bodies JSON, raw content: %s", content)
		return []GoverningBodyDetails{}, nil
	}

	log.Printf("🏛️  Found %d governing bodies for %s (%s)", len(result.GoverningBodies), businessName, businessType)
	return result.GoverningBodies, nil
}

// saveGoverningBodyToDynamoDB stores governing body information in DynamoDB
func saveGoverningBodyToDynamoDB(governingBody GoverningBody) {
	if !useDynamoDB {
		return
	}

	ctx := context.Background()

	item, err := attributevalue.MarshalMap(governingBody)
	if err != nil {
		log.Printf("❌ Failed to marshal governing body: %v", err)
		return
	}

	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(governingBodiesTableName),
		Item:      item,
	})
	if err != nil {
		log.Printf("❌ Failed to save governing body to DynamoDB: %v", err)
		return
	}

	log.Printf("💾 Governing body info saved to DynamoDB: %s", governingBody.OrganizationName)
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
		var loc Location
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
		var errorLog ErrorLog
		if err := json.NewDecoder(r.Body).Decode(&errorLog); err != nil {
			http.Error(w, "Invalid error log data", http.StatusBadRequest)
			return
		}

		errorLog.Timestamp = time.Now()
		errorLog.ID = fmt.Sprintf("%d", errorLog.Timestamp.UnixNano())

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
		businesses := make([]Business, len(currentBusinesses))
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

			// Search for governing bodies for each business (asynchronously)
			go func() {
				for _, business := range businesses {
					governingBodies, err := searchGoverningBodies(business.Name, business.Type, business.Address, userKeywords)
					if err != nil {
						log.Printf("⚠️  Error searching governing bodies for %s: %v", business.Name, err)
						continue
					}

					if len(governingBodies) > 0 {
						// Cache the governing bodies
						governingBodiesCacheMutex.Lock()
						governingBodiesCache[business.Name] = governingBodies
						governingBodiesCacheMutex.Unlock()

						governingBodyRecord := GoverningBody{
							ID:               fmt.Sprintf("%s-%d", business.Name, time.Now().UnixNano()),
							OrganizationName: business.Name,
							GoverningBodies:  governingBodies,
							Timestamp:        time.Now(),
						}

						saveGoverningBodyToDynamoDB(governingBodyRecord)

						// Log the results with website and contact info
						for _, gb := range governingBodies {
							contactInfo := ""
							if phone, ok := gb.ContactInfo["phone"].(string); ok && phone != "" {
								contactInfo += fmt.Sprintf(" | Phone: %s", phone)
							}
							if email, ok := gb.ContactInfo["email"].(string); ok && email != "" {
								contactInfo += fmt.Sprintf(" | Email: %s", email)
							}
							log.Printf("🏛️  %s (%s) - %s%s", gb.Name, gb.Type, gb.Website, contactInfo)
						}
					}
				}
			}()
		}

		// Store in memory cache
		errorLogMutex.Lock()
		errorLogs = append(errorLogs, errorLog)
		// Keep only last 50 errors in memory
		if len(errorLogs) > 50 {
			errorLogs = errorLogs[len(errorLogs)-50:]
		}
		errorLogMutex.Unlock()

		// Persist to DynamoDB (appends to existing data, never deletes)
		if useDynamoDB {
			go saveErrorLogToDynamoDB(errorLog)
		}

		log.Printf("📝 Error logged: %s", errorLog.Message)

		json.NewEncoder(w).Encode(map[string]bool{"success": true})

	case "GET":
		// GET requires auth (viewing logs in UI)
		if !isAuthenticated(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Return recent error logs
		errorLogMutex.RLock()
		defer errorLogMutex.RUnlock()

		json.NewEncoder(w).Encode(errorLogs)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
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

func handleGoverningBodies(w http.ResponseWriter, r *http.Request) {
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

	governingBodiesCacheMutex.RLock()
	defer governingBodiesCacheMutex.RUnlock()

	json.NewEncoder(w).Encode(governingBodiesCache)
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
	keywords := extractKeywords(messageBody)

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

	// Respond with TwiML (Twilio expects XML response)
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><Response></Response>`)
}

func isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("auth")
	if err != nil {
		return false
	}
	return cookie.Value == "authenticated"
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
	if err := tmpl.Execute(w, nil); err != nil {
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

// saveErrorLogToDynamoDB appends error log to DynamoDB (never deletes existing data)
func saveErrorLogToDynamoDB(errorLog ErrorLog) {
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
func saveLocationToDynamoDB(location Location) {
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
		var loadedErrorLogs []ErrorLog
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
		var loadedLocations []Location
		err = attributevalue.UnmarshalListOfMaps(locationsResult.Items, &loadedLocations)
		if err != nil {
			log.Printf("⚠️  Failed to unmarshal locations: %v", err)
		} else {
			// Filter to last 24 hours and get most recent per device
			now := time.Now()
			recentLocations := make(map[string]Location)

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
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            max-width: 600px;
            width: 100%;
            overflow: hidden;
        }
        .header {
            background: #667eea;
            color: white;
            padding: 30px;
            text-align: center;
        }
        .header h1 { font-size: 24px; margin-bottom: 5px; }
        .header p { opacity: 0.9; font-size: 14px; }
        .content { padding: 30px; }

        /* Login Form */
        #login input {
            width: 100%;
            padding: 15px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 16px;
            margin-bottom: 15px;
            transition: border-color 0.3s;
        }
        #login input:focus {
            outline: none;
            border-color: #667eea;
        }
        button {
            width: 100%;
            padding: 15px;
            background: #667eea;
            color: white;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: background 0.3s;
        }
        button:hover { background: #5568d3; }
        button:active { transform: scale(0.98); }
        .error {
            background: #fee;
            color: #c33;
            padding: 12px;
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
        .btn-share { background: #10b981; }
        .btn-share:hover { background: #059669; }
        .btn-refresh { background: #6366f1; }
        .btn-refresh:hover { background: #4f46e5; }

        .location-card {
            background: #f9fafb;
            border: 2px solid #e5e7eb;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 15px;
        }
        .location-card h3 {
            color: #667eea;
            margin-bottom: 10px;
            font-size: 16px;
        }
        .location-detail {
            display: flex;
            justify-content: space-between;
            padding: 8px 0;
            border-bottom: 1px solid #e5e7eb;
        }
        .location-detail:last-child { border-bottom: none; }
        .label { color: #6b7280; font-weight: 500; }
        .value { color: #1f2937; font-family: monospace; }
        .map-link {
            display: inline-block;
            margin-top: 15px;
            padding: 10px 20px;
            background: #667eea;
            color: white;
            text-decoration: none;
            border-radius: 6px;
            font-size: 14px;
            transition: background 0.3s;
        }
        .map-link:hover { background: #5568d3; }
        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: #6b7280;
        }
        .empty-state svg {
            width: 80px;
            height: 80px;
            margin-bottom: 20px;
            opacity: 0.5;
        }
        .status {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: 600;
            background: #d1fae5;
            color: #065f46;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📍 Location Tracker</h1>
            <p>Educational Personal Security Project</p>
        </div>

        <div class="content">
            <!-- Login View -->
            <div id="login">
                <input type="password" id="password" placeholder="Enter password" autofocus>
                <button onclick="login()">🔓 Login</button>
                <div class="error" id="error">Invalid password. Please try again.</div>
            </div>

            <!-- Tracker View -->
            <div id="tracker">
                <div class="actions">
                    <button class="btn-share" onclick="shareLocation()">📍 Share Location</button>
                    <button class="btn-refresh" onclick="refreshLocations()">🔄 Refresh</button>
                </div>
                <h3 style="margin-top: 20px; color: #667eea;">📍 Device Locations</h3>
                <div id="locations"></div>
                <h3 style="margin-top: 30px; color: #667eea;">📝 Recent Error Logs</h3>
                <div id="errorlogs"></div>
                <h3 style="margin-top: 30px; color: #667eea;">🏛️ Governing Bodies for Nearby Businesses</h3>
                <div id="governingbodies"></div>
            </div>
        </div>
    </div>

    <script>
        let deviceID = localStorage.getItem('deviceID');
        if (!deviceID) {
            deviceID = 'device_' + Math.random().toString(36).substr(2, 9);
            localStorage.setItem('deviceID', deviceID);
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
                    document.getElementById('login').style.display = 'none';
                    document.getElementById('tracker').style.display = 'block';
                    refreshLocations();
                    refreshErrorLogs();
                    // Auto-refresh every 10 seconds
                    setInterval(() => {
                        refreshLocations();
                        refreshErrorLogs();
                        refreshGoverningBodies();
                    }, 10000);
                } else {
                    errorEl.style.display = 'block';
                    setTimeout(() => errorEl.style.display = 'none', 3000);
                }
            } catch (e) {
                alert('Connection error: ' + e.message);
            }
        }

        // Handle Enter key on password field
        document.addEventListener('DOMContentLoaded', () => {
            document.getElementById('password').addEventListener('keypress', (e) => {
                if (e.key === 'Enter') login();
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
                displayErrorLogs(errorLogs);
            } catch (e) {
                console.error('Error fetching error logs:', e);
            }
        }

        // Display error logs
        function displayErrorLogs(errorLogs) {
            const container = document.getElementById('errorlogs');

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

            container.innerHTML = '';

            // Show most recent errors first
            const recentErrors = errorLogs.slice(-10).reverse();

            for (const errorLog of recentErrors) {
                const age = getLocationAge(errorLog.timestamp);

                const div = document.createElement('div');
                div.className = 'location-card';
                div.style.borderLeft = '4px solid #ef4444';
                div.innerHTML = ` + "`" + `
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
                    ${errorLog.gif_url ? ` + "`" + `
                        <a href="${errorLog.gif_url}" target="_blank" class="map-link" style="background: #ef4444;">
                            🎬 View GIF
                        </a>
                    ` + "`" + ` : ''}
                    ${errorLog.song_url ? ` + "`" + `
                        <a href="${errorLog.song_url}" target="_blank" class="map-link" style="background: #1db954;">
                            🎵 Play on Spotify
                        </a>
                    ` + "`" + ` : ''}
                ` + "`" + `;
                container.appendChild(div);
            }
        }

        // Refresh governing bodies
        async function refreshGoverningBodies() {
            try {
                const res = await fetch('/api/governingbodies');
                if (!res.ok) {
                    return;
                }

                const governingBodies = await res.json();
                displayGoverningBodies(governingBodies);
            } catch (e) {
                console.error('Error fetching governing bodies:', e);
            }
        }

        // Display governing bodies
        function displayGoverningBodies(governingBodiesMap) {
            const container = document.getElementById('governingbodies');

            if (!governingBodiesMap || Object.keys(governingBodiesMap).length === 0) {
                container.innerHTML = '<div class="empty-state" style="padding: 40px 20px;">' +
                    '<p style="color: #6b7280;">No governing body information yet</p>' +
                    '<p style="font-size: 14px; margin-top: 10px; color: #9ca3af;">' +
                    'Share a location to fetch nearby businesses and their regulatory authorities' +
                    '</p></div>';
                return;
            }

            container.innerHTML = '';

            for (const [businessName, governingBodies] of Object.entries(governingBodiesMap)) {
                const div = document.createElement('div');
                div.className = 'location-card';
                div.style.borderLeft = '4px solid #8b5cf6';

                let bodiesHTML = '';
                for (const gb of governingBodies) {
                    const typeColor = gb.type === 'regulatory' ? '#dc2626' : '#2563eb';
                    const typeIcon = gb.type === 'regulatory' ? '⚖️' : '🏢';

                    let contactHTML = '';
                    if (gb.contact_info) {
                        if (gb.contact_info.phone) {
                            contactHTML += '<div style="margin-top: 4px; font-size: 13px;">📞 ' + gb.contact_info.phone + '</div>';
                        }
                        if (gb.contact_info.email) {
                            contactHTML += '<div style="margin-top: 4px; font-size: 13px;">✉️ <a href="mailto:' + gb.contact_info.email + '" style="color: #667eea;">' + gb.contact_info.email + '</a></div>';
                        }
                        if (gb.contact_info.address) {
                            contactHTML += '<div style="margin-top: 4px; font-size: 13px;">📍 ' + gb.contact_info.address + '</div>';
                        }
                    }

                    bodiesHTML += '<div style="padding: 12px; background: #f9fafb; border-radius: 6px; margin-bottom: 10px;">' +
                        '<div style="display: flex; align-items: center; gap: 8px; margin-bottom: 6px;">' +
                        '<span>' + typeIcon + '</span>' +
                        '<strong style="color: ' + typeColor + '; font-size: 14px;">' + gb.name + '</strong>' +
                        '<span style="background: ' + typeColor + '20; color: ' + typeColor + '; padding: 2px 8px; border-radius: 4px; font-size: 11px; text-transform: uppercase;">' + gb.type + '</span>' +
                        '</div>' +
                        (gb.description ? '<div style="color: #6b7280; font-size: 13px; margin-bottom: 6px;">' + gb.description + '</div>' : '') +
                        (gb.website ? '<div style="margin-top: 6px;"><a href="' + gb.website + '" target="_blank" style="color: #667eea; text-decoration: none; font-size: 13px;">🔗 ' + gb.website + '</a></div>' : '') +
                        contactHTML +
                        '</div>';
                }

                div.innerHTML = '<h3 style="margin-bottom: 15px; color: #8b5cf6; font-size: 16px;">🏢 ' + businessName + '</h3>' + bodiesHTML;
                container.appendChild(div);
            }
        }
    </script>
</body>
</html>`
