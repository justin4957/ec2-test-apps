package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Solid authentication support (dual-mode: works alongside existing password auth)

// SolidSession represents an authenticated Solid session
type SolidSession struct {
	WebID        string    `json:"webid"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	PodURL       string    `json:"pod_url"`
	Provider     string    `json:"provider"`
}

// OIDCConfiguration represents OpenID Connect discovery document
type OIDCConfiguration struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKsURI               string `json:"jwks_uri"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
}

// SolidProvider represents a Solid Pod provider
type SolidProvider struct {
	Name        string
	IssuerURL   string
	Description string
}

// Common Solid providers
var solidProviders = []SolidProvider{
	{
		Name:        "Inrupt PodSpaces",
		IssuerURL:   "https://login.inrupt.com",
		Description: "Commercial Solid provider by Inrupt",
	},
	{
		Name:        "SolidCommunity.net",
		IssuerURL:   "https://solidcommunity.net",
		Description: "Community-run Solid server",
	},
	{
		Name:        "SolidWeb",
		IssuerURL:   "https://solidweb.me",
		Description: "Public Solid provider",
	},
}

var (
	// In-memory session storage (in production, use Redis or database)
	solidSessions      = make(map[string]*SolidSession)
	solidSessionsMutex sync.RWMutex

	// OIDC state for CSRF protection
	oidcStates      = make(map[string]time.Time)
	oidcStatesMutex sync.RWMutex

	// Solid feature flag
	solidEnabled = os.Getenv("SOLID_ENABLED") == "true"

	// OAuth client configuration
	solidClientID     = os.Getenv("SOLID_CLIENT_ID")
	solidClientSecret = os.Getenv("SOLID_CLIENT_SECRET")
	solidRedirectURI  = os.Getenv("SOLID_REDIRECT_URI")
)

// handleSolidProviders returns list of available Solid providers
func handleSolidProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !solidEnabled {
		http.Error(w, "Solid authentication not enabled", http.StatusNotImplemented)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": solidProviders,
		"enabled":   true,
	})
}

// handleSolidLogin initiates the Solid OIDC authentication flow
func handleSolidLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !solidEnabled {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "solid_not_enabled",
			"message": "Solid authentication is not enabled on this server. Set SOLID_ENABLED=true to activate.",
		})
		return
	}

	// Parse request
	var req struct {
		IssuerURL string `json:"issuer_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
		return
	}

	if req.IssuerURL == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "missing_issuer",
			"message": "issuer_url is required",
		})
		return
	}

	log.Printf("🔐 Initiating Solid login with issuer: %s", req.IssuerURL)

	// Discover OIDC configuration
	oidcConfig, err := discoverOIDCConfiguration(req.IssuerURL)
	if err != nil {
		log.Printf("❌ Failed to discover OIDC configuration: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "discovery_failed",
			"message": fmt.Sprintf("Failed to discover OIDC provider at %s: %v", req.IssuerURL, err),
			"help":    "Please check that the issuer URL is correct and supports OpenID Connect",
		})
		return
	}

	// Generate state for CSRF protection
	state := generateRandomString(32)
	oidcStatesMutex.Lock()
	oidcStates[state] = time.Now().Add(10 * time.Minute)
	oidcStatesMutex.Unlock()

	// Build authorization URL
	authURL := buildAuthorizationURL(oidcConfig.AuthorizationEndpoint, state)

	log.Printf("✅ Redirecting to authorization endpoint: %s", oidcConfig.AuthorizationEndpoint)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"authorization_url": authURL,
		"state":             state,
	})
}

// handleSolidCallback handles the OAuth callback from Solid provider
func handleSolidCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !solidEnabled {
		http.Error(w, "Solid authentication not enabled", http.StatusNotImplemented)
		return
	}

	// Parse callback parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		log.Printf("❌ OAuth error: %s - %s", errorParam, r.URL.Query().Get("error_description"))
		http.Redirect(w, r, "/?error=auth_failed", http.StatusSeeOther)
		return
	}

	if code == "" || state == "" {
		http.Error(w, "Missing code or state parameter", http.StatusBadRequest)
		return
	}

	// Verify state (CSRF protection)
	oidcStatesMutex.Lock()
	stateExpiry, exists := oidcStates[state]
	if exists {
		delete(oidcStates, state)
	}
	oidcStatesMutex.Unlock()

	if !exists || time.Now().After(stateExpiry) {
		log.Printf("❌ Invalid or expired state: %s", state)
		http.Redirect(w, r, "/?error=invalid_state", http.StatusSeeOther)
		return
	}

	log.Printf("🔐 Received OAuth callback with valid state")

	// For PoC, we'll simulate successful authentication
	// In production, exchange code for tokens and validate
	webID := fmt.Sprintf("https://example.solidcommunity.net/profile/card#me-%s", generateRandomString(8))
	sessionID := generateRandomString(32)

	session := &SolidSession{
		WebID:       webID,
		AccessToken: "poc_token_" + generateRandomString(32),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		PodURL:      "https://example.solidcommunity.net/",
		Provider:    "SolidCommunity.net",
	}

	solidSessionsMutex.Lock()
	solidSessions[sessionID] = session
	solidSessionsMutex.Unlock()

	log.Printf("✅ Solid authentication successful - WebID: %s", webID)

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "solid_session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   useHTTPS,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 hours
	})

	// Redirect to main page
	http.Redirect(w, r, "/?solid=success", http.StatusSeeOther)
}

// handleSolidSession returns current Solid session info
func handleSolidSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session := getSolidSession(r)
	if session == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
			"auth_type":     nil,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": true,
		"auth_type":     "solid",
		"webid":         session.WebID,
		"pod_url":       session.PodURL,
		"provider":      session.Provider,
		"expires_at":    session.ExpiresAt,
	})
}

// handleSolidLogout logs out Solid session
func handleSolidLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("solid_session")
	if err == nil {
		solidSessionsMutex.Lock()
		delete(solidSessions, cookie.Value)
		solidSessionsMutex.Unlock()
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "solid_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// getSolidSession retrieves Solid session from request
func getSolidSession(r *http.Request) *SolidSession {
	cookie, err := r.Cookie("solid_session")
	if err != nil {
		return nil
	}

	solidSessionsMutex.RLock()
	session, exists := solidSessions[cookie.Value]
	solidSessionsMutex.RUnlock()

	if !exists || time.Now().After(session.ExpiresAt) {
		return nil
	}

	return session
}

// isSolidAuthenticated checks if request has valid Solid authentication
func isSolidAuthenticated(r *http.Request) bool {
	return getSolidSession(r) != nil
}

// discoverOIDCConfiguration fetches OpenID Connect configuration
func discoverOIDCConfiguration(issuerURL string) (*OIDCConfiguration, error) {
	// Ensure issuerURL doesn't have trailing slash
	issuerURL = strings.TrimSuffix(issuerURL, "/")

	discoveryURL := issuerURL + "/.well-known/openid-configuration"

	log.Printf("🔍 Discovering OIDC configuration at: %s", discoveryURL)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OIDC configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OIDC discovery failed with status: %d", resp.StatusCode)
	}

	var config OIDCConfiguration
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse OIDC configuration: %w", err)
	}

	log.Printf("✅ OIDC configuration discovered - Issuer: %s", config.Issuer)

	return &config, nil
}

// buildAuthorizationURL constructs OAuth authorization URL
func buildAuthorizationURL(authEndpoint, state string) string {
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", getClientID())
	params.Set("redirect_uri", getRedirectURI())
	params.Set("scope", "openid profile offline_access")
	params.Set("state", state)

	return authEndpoint + "?" + params.Encode()
}

// getClientID returns OAuth client ID (with fallback for PoC)
func getClientID() string {
	if solidClientID != "" {
		return solidClientID
	}
	// PoC: Use a placeholder client ID
	return "location-tracker-poc"
}

// getRedirectURI returns OAuth redirect URI
func getRedirectURI() string {
	if solidRedirectURI != "" {
		return solidRedirectURI
	}
	// PoC: Build from current server
	baseURL := "http://localhost:8080"
	if useHTTPS {
		baseURL = "https://notspies.org"
	}
	return baseURL + "/api/solid/callback"
}

// generateRandomString generates a cryptographically random string
func generateRandomString(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(b)[:length]
}

// cleanupExpiredStates removes expired OIDC states
func cleanupExpiredStates() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		oidcStatesMutex.Lock()
		now := time.Now()
		for state, expiry := range oidcStates {
			if now.After(expiry) {
				delete(oidcStates, state)
			}
		}
		oidcStatesMutex.Unlock()
	}
}

// Solid Pod HTTP operations (simplified for PoC)

// SolidResource represents a resource in a Solid Pod
type SolidResource struct {
	URL         string
	ContentType string
	Data        []byte
}

// writeToPod writes data to user's Solid Pod (PoC implementation)
func writeToPod(session *SolidSession, resourcePath string, data []byte, contentType string) error {
	if session == nil {
		return fmt.Errorf("no Solid session")
	}

	// In PoC, we log instead of actually writing
	log.Printf("📝 [POC] Would write to Pod: %s%s (type: %s, size: %d bytes)",
		session.PodURL, resourcePath, contentType, len(data))

	// In production, this would do:
	// 1. Construct full URL: session.PodURL + resourcePath
	// 2. Create HTTP PUT request with data
	// 3. Add Authorization: DPoP <session.AccessToken>
	// 4. Send request and handle response

	return nil
}

// readFromPod reads data from user's Solid Pod (PoC implementation)
func readFromPod(session *SolidSession, resourcePath string) (*SolidResource, error) {
	if session == nil {
		return nil, fmt.Errorf("no Solid session")
	}

	log.Printf("📥 [POC] Would read from Pod: %s%s", session.PodURL, resourcePath)

	// In PoC, return empty data
	return &SolidResource{
		URL:         session.PodURL + resourcePath,
		ContentType: "text/turtle",
		Data:        []byte{},
	}, nil

	// In production, this would do:
	// 1. Construct full URL
	// 2. Create HTTP GET request
	// 3. Add Authorization header
	// 4. Parse response and return data
}

// listPodContainers lists containers in user's Pod (PoC implementation)
func listPodContainers(session *SolidSession, containerPath string) ([]string, error) {
	if session == nil {
		return nil, fmt.Errorf("no Solid session")
	}

	log.Printf("📋 [POC] Would list Pod containers at: %s%s", session.PodURL, containerPath)

	// In PoC, return empty list
	return []string{}, nil
}
