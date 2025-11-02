package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"html/template"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"sort"
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
	Timestamp           time.Time `json:"timestamp" dynamodbav:"timestamp"`
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

	// Pending user experience note from Twilio SMS
	pendingUserExperienceNote string
	userExperienceNoteMutex   sync.RWMutex

	// Global password from environment
	globalPassword = os.Getenv("TRACKER_PASSWORD")

	// Google Maps API key
	googleMapsAPIKey = os.Getenv("GOOGLE_MAPS_API_KEY")

	// HTTPS mode flag
	useHTTPS = false

	// DynamoDB client
	dynamoClient *dynamodb.Client
	useDynamoDB  = false

	// DynamoDB table names
	errorLogsTableName = "location-tracker-error-logs"
	locationsTableName = "location-tracker-locations"
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
	http.HandleFunc("/api/health", handleHealth)
	http.HandleFunc("/api/twilio/sms", handleTwilioWebhook)

	// Start cleanup goroutine (remove locations older than 24h)
	go cleanupOldLocations()

	// Load existing data from DynamoDB on startup (preserves all existing records)
	if useDynamoDB {
		go loadExistingData()
	}

	port := os.Getenv("PORT")
	if port == "" {
		if useHTTPS {
			port = "8443"
		} else {
			port = "8080"
		}
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

		log.Printf("🌍 Server running on https://:%s", port)
		log.Fatal(http.ListenAndServeTLS(":"+port, certFile, keyFile, nil))
	} else {
		log.Printf("⚠️  Running in HTTP mode - geolocation may not work in browsers!")
		log.Printf("💡 Set USE_HTTPS=true to enable HTTPS")
		log.Printf("🌍 Server running on http://:%s", port)
		log.Fatal(http.ListenAndServe(":"+port, nil))
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

		// Attach pending user experience note from Twilio SMS if available
		userExperienceNoteMutex.Lock()
		if pendingUserExperienceNote != "" {
			errorLog.UserExperienceNote = pendingUserExperienceNote
			log.Printf("💬 Attached user experience note: %s", pendingUserExperienceNote)
			pendingUserExperienceNote = "" // Clear after attaching
		}
		userExperienceNoteMutex.Unlock()

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

	// Store the message as pending user experience note
	userExperienceNoteMutex.Lock()
	pendingUserExperienceNote = messageBody
	userExperienceNoteMutex.Unlock()

	log.Printf("📱 Received SMS from %s (SID: %s): %s", messageFrom, messageSid, messageBody)
	log.Printf("💬 Stored user experience note, will attach to next error log")

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
	tmpl.Execute(w, nil)
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
    </script>
</body>
</html>`
