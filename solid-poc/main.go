package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/justin4957/ec2-test-apps/solid-poc/internal/solid"
)

// loggingMiddleware wraps an http.Handler and logs all requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request
		log.Printf("→ %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Log headers (excluding sensitive tokens)
		if log.Writer() != nil {
			for key, values := range r.Header {
				if key == "Authorization" || key == "Cookie" {
					log.Printf("  %s: [REDACTED]", key)
				} else {
					log.Printf("  %s: %v", key, values)
				}
			}
		}

		// Create a response wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Log response
		duration := time.Since(start)
		log.Printf("← %s %s - %d (%v)", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// SolidProvider represents a Solid identity provider
type SolidProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IssuerURL   string `json:"issuer_url"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url,omitempty"`
}

// SolidSession represents a user's Solid session
type SolidSession struct {
	SessionID string    `json:"session_id"`
	WebID     string    `json:"webid"`
	Name      string    `json:"name,omitempty"`
	Photo     string    `json:"photo,omitempty"`
	Provider  string    `json:"provider"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// In-memory session storage (replace with database in production)
var (
	solidSessions = make(map[string]*SolidSession)
	sessionsMutex sync.RWMutex
)

// Common Solid identity providers
var solidProviders = []SolidProvider{
	{
		ID:          "solidcommunity",
		Name:        "SolidCommunity.net",
		IssuerURL:   "https://solidcommunity.net",
		Description: "Free public Solid Pod provider",
	},
	{
		ID:          "inrupt",
		Name:        "Inrupt PodSpaces",
		IssuerURL:   "https://login.inrupt.com",
		Description: "Commercial Solid Pod service by Inrupt",
	},
	{
		ID:          "solidweb",
		Name:        "solidweb.org",
		IssuerURL:   "https://solidweb.org",
		Description: "Community Solid Pod provider",
	},
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	// Serve static files from frontend directory with logging
	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", loggingMiddleware(fs))

	// API endpoints for RDF operations
	http.HandleFunc("/api/health", loggingHandler(healthHandler))
	http.HandleFunc("/api/rdf/serialize", loggingHandler(serializeHandler))
	http.HandleFunc("/api/rdf/deserialize", loggingHandler(deserializeHandler))

	// Solid WebID authentication endpoints
	http.HandleFunc("/api/solid/providers", loggingHandler(handleSolidProviders))
	http.HandleFunc("/api/solid/session", loggingHandler(handleSolidSession))
	http.HandleFunc("/api/solid/logout", loggingHandler(handleSolidLogout))
	http.HandleFunc("/api/solid/webid/profile", loggingHandler(handleWebIDProfile))

	// Start background session cleanup
	go cleanupExpiredSessions()

	log.Printf("🔐 Solid PoC Server starting on port %s", port)
	log.Printf("📂 Serving frontend from ./frontend")
	log.Printf("🌐 Access at: http://localhost:%s", port)
	log.Printf("📝 Logging enabled for all requests")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// loggingHandler wraps an http.HandlerFunc and logs requests/responses
func loggingHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request
		log.Printf("→ %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Log headers (excluding sensitive tokens)
		for key, values := range r.Header {
			if key == "Authorization" || key == "Cookie" {
				log.Printf("  %s: [REDACTED]", key)
			} else {
				log.Printf("  %s: %v", key, values)
			}
		}

		// Create a response wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler
		next(wrapped, r)

		// Log response
		duration := time.Since(start)
		log.Printf("← %s %s - %d (%v)", r.Method, r.URL.Path, wrapped.statusCode, duration)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "solid-poc",
		"version": "1.0.0",
	})
}

// serializeHandler converts Location data to RDF (Turtle or JSON-LD)
func serializeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("❌ RDF serialize failed: method not allowed (%s)", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Format string                 `json:"format"` // "turtle" or "jsonld"
		Data   map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ RDF serialize failed: invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("🔄 Serializing to %s format", req.Format)

	var result string
	var err error

	switch req.Format {
	case "turtle":
		result, err = solid.SerializeToTurtle(req.Data)
	case "jsonld":
		result, err = solid.SerializeToJSONLD(req.Data)
	default:
		log.Printf("❌ RDF serialize failed: invalid format '%s'", req.Format)
		http.Error(w, "Invalid format. Use 'turtle' or 'jsonld'", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("❌ RDF serialize failed: %v", err)
		http.Error(w, fmt.Sprintf("Serialization failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ RDF serialization successful: %s (%d bytes)", req.Format, len(result))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"format": req.Format,
		"result": result,
	})
}

// deserializeHandler converts RDF to Go data structures
func deserializeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("❌ RDF deserialize failed: method not allowed (%s)", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Format string `json:"format"` // "turtle" or "jsonld"
		Data   string `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ RDF deserialize failed: invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("🔄 Deserializing from %s format (%d bytes)", req.Format, len(req.Data))

	var result map[string]interface{}
	var err error

	switch req.Format {
	case "turtle":
		result, err = solid.DeserializeFromTurtle(req.Data)
	case "jsonld":
		result, err = solid.DeserializeFromJSONLD(req.Data)
	default:
		log.Printf("❌ RDF deserialize failed: invalid format '%s'", req.Format)
		http.Error(w, "Invalid format. Use 'turtle' or 'jsonld'", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("❌ RDF deserialize failed: %v", err)
		http.Error(w, fmt.Sprintf("Deserialization failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ RDF deserialization successful: extracted %d fields", len(result))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleSolidProviders returns list of available Solid providers
func handleSolidProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": solidProviders,
	})
}

// handleSolidSession handles POST (create) and GET (check status) for sessions
func handleSolidSession(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		createSolidSession(w, r)
	case http.MethodGet:
		getSolidSessionStatus(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// createSolidSession creates a new Solid session from ID token
func createSolidSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDToken  string `json:"id_token"`
		Provider string `json:"provider"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify ID token and extract WebID
	webid, name, photo, err := verifyIDTokenAndExtractWebID(req.IDToken, req.Provider)
	if err != nil {
		log.Printf("Failed to verify ID token: %v", err)
		http.Error(w, fmt.Sprintf("Invalid ID token: %v", err), http.StatusUnauthorized)
		return
	}

	// Generate session ID
	sessionID, err := generateSessionID()
	if err != nil {
		http.Error(w, "Failed to generate session", http.StatusInternalServerError)
		return
	}

	// Create session
	session := &SolidSession{
		SessionID: sessionID,
		WebID:     webid,
		Name:      name,
		Photo:     photo,
		Provider:  req.Provider,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hour session
	}

	// Store session
	sessionsMutex.Lock()
	solidSessions[sessionID] = session
	sessionsMutex.Unlock()

	log.Printf("Created Solid session for WebID: %s", webid)

	// Return session info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// getSolidSessionStatus checks if a session is valid
func getSolidSessionStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id parameter required", http.StatusBadRequest)
		return
	}

	sessionsMutex.RLock()
	session, exists := solidSessions[sessionID]
	sessionsMutex.RUnlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Check if session expired
	if time.Now().After(session.ExpiresAt) {
		sessionsMutex.Lock()
		delete(solidSessions, sessionID)
		sessionsMutex.Unlock()

		http.Error(w, "Session expired", http.StatusUnauthorized)
		return
	}

	// Return session info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":      true,
		"session":    session,
		"expires_in": int(time.Until(session.ExpiresAt).Seconds()),
	})
}

// handleSolidLogout logs out a Solid session
func handleSolidLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Remove session
	sessionsMutex.Lock()
	session, exists := solidSessions[req.SessionID]
	if exists {
		delete(solidSessions, req.SessionID)
		log.Printf("Logged out Solid session for WebID: %s", session.WebID)
	}
	sessionsMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}

// handleWebIDProfile fetches public WebID profile information
func handleWebIDProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	webid := r.URL.Query().Get("webid")
	if webid == "" {
		http.Error(w, "webid parameter required", http.StatusBadRequest)
		return
	}

	// Fetch public WebID profile
	profile, err := fetchPublicWebIDProfile(webid)
	if err != nil {
		log.Printf("Failed to fetch WebID profile: %v", err)
		http.Error(w, fmt.Sprintf("Failed to fetch profile: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// verifyIDTokenAndExtractWebID verifies the ID token and extracts WebID
// NOTE: This is a simplified implementation. Production should:
// 1. Fetch provider's OIDC discovery document
// 2. Verify JWT signature using provider's public keys
// 3. Validate token claims (iss, aud, exp, etc.)
func verifyIDTokenAndExtractWebID(idToken, provider string) (webid, name, photo string, err error) {
	// TODO: Implement proper JWT verification
	// For now, this is a placeholder that trusts the frontend
	// In production, you MUST verify the JWT signature!

	log.Printf("⚠️  WARNING: ID token verification not yet implemented")
	log.Printf("⚠️  Accepting token from provider: %s without verification", provider)

	// Parse JWT payload (without verification - UNSAFE for production)
	parts := splitJWT(idToken)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid JWT format")
	}

	// Decode payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", "", fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	// Parse claims
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", "", "", fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	// Extract WebID
	webidClaim, ok := claims["webid"]
	if !ok {
		// Try alternative claim names
		if sub, ok := claims["sub"].(string); ok {
			webid = sub
		} else {
			return "", "", "", fmt.Errorf("no webid claim found in token")
		}
	} else {
		webid = webidClaim.(string)
	}

	// Extract optional profile info
	if nameClaim, ok := claims["name"].(string); ok {
		name = nameClaim
	}
	if photoClaim, ok := claims["picture"].(string); ok {
		photo = photoClaim
	}

	return webid, name, photo, nil
}

// fetchPublicWebIDProfile fetches public profile information from a WebID
func fetchPublicWebIDProfile(webid string) (map[string]interface{}, error) {
	// Fetch WebID document
	resp, err := http.Get(webid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch WebID document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("WebID returned status %d", resp.StatusCode)
	}

	// TODO: Parse RDF (Turtle/JSON-LD) to extract profile info
	// For now, return basic info
	return map[string]interface{}{
		"webid":       webid,
		"name":        "", // Would extract from RDF
		"photo":       "", // Would extract from RDF
		"description": "", // Would extract from RDF
	}, nil
}

// Helper functions

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func splitJWT(token string) []string {
	parts := make([]string, 0, 3)
	start := 0
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			parts = append(parts, token[start:i])
			start = i + 1
		}
	}
	parts = append(parts, token[start:])
	return parts
}

// cleanupExpiredSessions runs periodically to remove expired sessions
func cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		sessionsMutex.Lock()
		now := time.Now()
		for sessionID, session := range solidSessions {
			if now.After(session.ExpiresAt) {
				delete(solidSessions, sessionID)
				log.Printf("Cleaned up expired session for WebID: %s", session.WebID)
			}
		}
		sessionsMutex.Unlock()
	}
}
