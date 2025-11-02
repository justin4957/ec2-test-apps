package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type GiphyResponse struct {
	Data []struct {
		URL string `json:"url"`
	} `json:"data"`
}

type SpotifyAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type SpotifyTracksResponse struct {
	Tracks []struct {
		Name        string `json:"name"`
		ExternalURL struct {
			Spotify string `json:"spotify"`
		} `json:"external_urls"`
		Artists []struct {
			Name string `json:"name"`
		} `json:"artists"`
	} `json:"tracks"`
}

// Rock classics track IDs - hardcoded list since recommendations API requires user auth
var rockClassicsTrackIDs = []string{
	"4u7EnebtmKWzUH433cf5Qv", // Bohemian Rhapsody - Queen
	"5CQ30WqJwcep0pYcV4AMNc", // Stairway to Heaven - Led Zeppelin
	"40riOy7x9W7GXjyGp4pjAv", // Hotel California - Eagles
	"7o2CTH4ctstm8TNelqjb51", // Sweet Child O' Mine - Guns N' Roses
	"08mG3Y1vljYA6bvDt4Wqkj", // Back In Black - AC/DC
	"2Fxmhks0bxGSBdJ92vM42m", // Smells Like Teen Spirit - Nirvana
	"3qiyyUfYe7CRYLucrPmulD", // Thunderstruck - AC/DC
	"5v4GgrXPMghOnBBLmveLac", // Dream On - Aerosmith
	"2nLtzopw4rPReszdYBJU6h", // Enter Sandman - Metallica
	"5ghIJDpPoe3CfHMGu71E6T", // More Than A Feeling - Boston
}

type Song struct {
	Title  string
	Artist string
	URL    string
}

type ErrorLogRequest struct {
	Message    string `json:"message"`
	GifURL     string `json:"gif_url"`
	SongTitle  string `json:"song_title"`
	SongArtist string `json:"song_artist"`
	SongURL    string `json:"song_url"`
}

type SloganResponse struct {
	Emoji  string `json:"emoji"`
	Slogan string `json:"slogan"`
}

type Business struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Address  string `json:"address"`
	PlaceID  string `json:"place_id"`
	Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"location"`
}

type BusinessesResponse struct {
	Businesses []Business `json:"businesses"`
	Count      int        `json:"count"`
}

var errorMessages = []string{
	"NullPointerException in UserService.java:42",
	"IndexOutOfBoundsException: Index: 5, Size: 3",
	"ConnectionTimeoutException: Unable to reach database",
	"MemoryError: heap exhausted after 3.2GB allocation",
	"UnhandledPromiseRejection: undefined is not a function",
	"StackOverflowError in recursive fibonacci(50000)",
	"FileNotFoundException: config.yaml not found",
	"PermissionDeniedException: Access denied to /etc/secrets",
	"ConcurrentModificationException in ArrayList iteration",
	"DeadlockDetected: Thread pool exhausted",
	"SSLHandshakeException: Certificate expired",
	"OutOfMemoryError: GC overhead limit exceeded",
	"IllegalArgumentException: negative timeout value",
	"ClassCastException: String cannot be cast to Integer",
	"ArithmeticException: division by zero",
}

// Business-related error templates (placeholders will be filled with actual business names)
var businessErrorTemplates = []string{
	"APIRateLimitExceeded: %s payment gateway throttling requests",
	"OAuthTokenRevoked: %s authentication service denying access",
	"MerchantIDConflict: %s POS system reporting duplicate transaction",
	"InventoryMismatchException: %s database sync failed",
	"GeofenceViolation: %s location service boundary exceeded",
	"PaymentGatewayTimeout: %s checkout process unresponsive",
	"CustomerDataLeakage: %s CRM exposing sensitive records",
	"ReservationCollision: %s booking system double-allocated slot",
	"LoyaltyPointsCorruption: %s rewards API returning invalid balances",
	"DeliveryRouteOptimizationFailure: %s logistics algorithm deadlocked",
	"MenuItemPricingDisagreement: %s order system vs. %s catalog mismatch",
	"RefundProcessorHalted: %s transaction reversal stuck in limbo",
	"StockLevelDesync: %s warehouse claiming negative inventory",
	"BusinessHoursParsingError: %s schedule API returned malformed data",
	"TaxCalculationBreach: %s payment system vs. local regulations conflict",
}

// HTTP client for location tracker with TLS skip verify (for self-signed certs)
var locationTrackerHTTPClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
	Timeout: 10 * time.Second,
}

type GifCache struct {
	gifURLs       []string
	currentIndex  int
	lastRefresh   time.Time
	giphyAPIKey   string
	refreshNeeded bool
}

func newGifCache(apiKey string) *GifCache {
	return &GifCache{
		gifURLs:       make([]string, 0),
		giphyAPIKey:   apiKey,
		refreshNeeded: true,
	}
}

func (gifCache *GifCache) loadGifsFromGiphy() error {
	if gifCache.giphyAPIKey == "" {
		log.Println("GIPHY_API_KEY not set, using placeholder GIFs")
		gifCache.gifURLs = []string{
			"https://giphy.com/gifs/error-placeholder-1",
			"https://giphy.com/gifs/error-placeholder-2",
			"https://giphy.com/gifs/error-placeholder-3",
			"https://giphy.com/gifs/error-placeholder-4",
			"https://giphy.com/gifs/error-placeholder-5",
		}
		gifCache.lastRefresh = time.Now()
		gifCache.currentIndex = 0
		gifCache.refreshNeeded = false
		return nil
	}

	searchTerms := []string{"error", "fail", "glitch", "broken", "oops"}
	randomSearchTerm := searchTerms[rand.Intn(len(searchTerms))]

	url := fmt.Sprintf("https://api.giphy.com/v1/gifs/search?api_key=%s&q=%s&limit=25&rating=g",
		gifCache.giphyAPIKey, randomSearchTerm)

	httpResponse, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch from Giphy: %w", err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("giphy API returned status: %d", httpResponse.StatusCode)
	}

	var giphyResponse GiphyResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&giphyResponse); err != nil {
		return fmt.Errorf("failed to decode Giphy response: %w", err)
	}

	gifCache.gifURLs = make([]string, 0, len(giphyResponse.Data))
	for _, gif := range giphyResponse.Data {
		gifCache.gifURLs = append(gifCache.gifURLs, gif.URL)
	}

	gifCache.lastRefresh = time.Now()
	gifCache.currentIndex = 0
	gifCache.refreshNeeded = false

	log.Printf("Loaded %d GIF URLs from Giphy (search term: %s)", len(gifCache.gifURLs), randomSearchTerm)
	return nil
}

func (gifCache *GifCache) getNextGif() string {
	if gifCache.refreshNeeded || len(gifCache.gifURLs) == 0 {
		if err := gifCache.loadGifsFromGiphy(); err != nil {
			log.Printf("Error loading GIFs: %v", err)
			return "https://giphy.com/gifs/error-fallback"
		}
	}

	if gifCache.currentIndex >= len(gifCache.gifURLs) {
		gifCache.refreshNeeded = true
		gifCache.currentIndex = 0
		if err := gifCache.loadGifsFromGiphy(); err != nil {
			log.Printf("Error refreshing GIFs: %v", err)
			return "https://giphy.com/gifs/error-fallback"
		}
	}

	gif := gifCache.gifURLs[gifCache.currentIndex]
	gifCache.currentIndex++

	return gif
}

type SpotifyCache struct {
	songs            []Song
	currentIndex     int
	lastRefresh      time.Time
	accessToken      string
	tokenExpiry      time.Time
	clientID         string
	clientSecret     string
	seedGenres       string
	refreshNeeded    bool
}

func newSpotifyCache(clientID, clientSecret, seedGenres string) *SpotifyCache {
	return &SpotifyCache{
		songs:         make([]Song, 0),
		clientID:      clientID,
		clientSecret:  clientSecret,
		seedGenres:    seedGenres,
		refreshNeeded: true,
	}
}

func (spotifyCache *SpotifyCache) authenticate() error {
	if spotifyCache.clientID == "" || spotifyCache.clientSecret == "" {
		return fmt.Errorf("spotify credentials not set")
	}

	authURL := "https://accounts.spotify.com/api/token"
	requestBody := bytes.NewBufferString("grant_type=client_credentials")

	httpRequest, err := http.NewRequest("POST", authURL, requestBody)
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}

	httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpRequest.SetBasicAuth(spotifyCache.clientID, spotifyCache.clientSecret)

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("failed to authenticate with Spotify: %w", err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("spotify auth returned status: %d", httpResponse.StatusCode)
	}

	var authResponse SpotifyAuthResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&authResponse); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	spotifyCache.accessToken = authResponse.AccessToken
	spotifyCache.tokenExpiry = time.Now().Add(time.Duration(authResponse.ExpiresIn) * time.Second)

	log.Printf("Authenticated with Spotify (token expires in %d seconds)", authResponse.ExpiresIn)
	return nil
}

func (spotifyCache *SpotifyCache) loadSongsFromSpotify() error {
	if spotifyCache.clientID == "" || spotifyCache.clientSecret == "" {
		log.Println("Spotify credentials not set, using placeholder songs")
		spotifyCache.songs = []Song{
			{Title: "Bohemian Rhapsody", Artist: "Queen", URL: "https://open.spotify.com/track/4u7EnebtmKWzUH433cf5Qv"},
			{Title: "Stairway to Heaven", Artist: "Led Zeppelin", URL: "https://open.spotify.com/track/5CQ30WqJwcep0pYcV4AMNc"},
			{Title: "Hotel California", Artist: "Eagles", URL: "https://open.spotify.com/track/40riOy7x9W7GXjyGp4pjAv"},
			{Title: "Sweet Child O' Mine", Artist: "Guns N' Roses", URL: "https://open.spotify.com/track/7o2CTH4ctstm8TNelqjb51"},
			{Title: "Back In Black", Artist: "AC/DC", URL: "https://open.spotify.com/track/08mG3Y1vljYA6bvDt4Wqkj"},
		}
		spotifyCache.lastRefresh = time.Now()
		spotifyCache.currentIndex = 0
		spotifyCache.refreshNeeded = false
		return nil
	}

	// Check if we need to refresh the access token
	if time.Now().After(spotifyCache.tokenExpiry) || spotifyCache.accessToken == "" {
		if err := spotifyCache.authenticate(); err != nil {
			return err
		}
	}

	// Use Spotify Tracks API with hardcoded rock classics
	// Recommendations API requires user authorization, so we use a curated list
	trackIDs := rockClassicsTrackIDs
	if len(trackIDs) > 50 {
		trackIDs = trackIDs[:50] // Spotify API limit
	}
	idsParam := ""
	for i, id := range trackIDs {
		if i > 0 {
			idsParam += ","
		}
		idsParam += id
	}

	tracksURL := fmt.Sprintf("https://api.spotify.com/v1/tracks?ids=%s", idsParam)

	httpRequest, err := http.NewRequest("GET", tracksURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create tracks request: %w", err)
	}

	httpRequest.Header.Set("Authorization", "Bearer "+spotifyCache.accessToken)

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("failed to fetch tracks: %w", err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("spotify API returned status: %d", httpResponse.StatusCode)
	}

	var tracksResponse SpotifyTracksResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&tracksResponse); err != nil {
		return fmt.Errorf("failed to decode tracks response: %w", err)
	}

	spotifyCache.songs = make([]Song, 0, len(tracksResponse.Tracks))
	for _, track := range tracksResponse.Tracks {
		if track.Name != "" {
			artistName := "Unknown Artist"
			if len(track.Artists) > 0 {
				artistName = track.Artists[0].Name
			}
			spotifyCache.songs = append(spotifyCache.songs, Song{
				Title:  track.Name,
				Artist: artistName,
				URL:    track.ExternalURL.Spotify,
			})
		}
	}

	spotifyCache.lastRefresh = time.Now()
	spotifyCache.currentIndex = 0
	spotifyCache.refreshNeeded = false

	log.Printf("Loaded %d rock classics from Spotify", len(spotifyCache.songs))
	return nil
}

func (spotifyCache *SpotifyCache) getNextSong() Song {
	if spotifyCache.refreshNeeded || len(spotifyCache.songs) == 0 {
		if err := spotifyCache.loadSongsFromSpotify(); err != nil {
			log.Printf("Error loading songs: %v", err)
			return Song{
				Title:  "Error Song",
				Artist: "Unknown",
				URL:    "https://open.spotify.com/track/fallback",
			}
		}
	}

	if spotifyCache.currentIndex >= len(spotifyCache.songs) {
		spotifyCache.currentIndex = 0
	}

	song := spotifyCache.songs[spotifyCache.currentIndex]
	spotifyCache.currentIndex++

	return song
}

func fetchBusinesses(trackerURL string) ([]Business, error) {
	if trackerURL == "" {
		return []Business{}, nil
	}

	resp, err := locationTrackerHTTPClient.Get(trackerURL + "/api/businesses")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch businesses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("businesses endpoint returned status: %d", resp.StatusCode)
	}

	var businessesResp BusinessesResponse
	if err := json.NewDecoder(resp.Body).Decode(&businessesResp); err != nil {
		return nil, fmt.Errorf("failed to decode businesses response: %w", err)
	}

	return businessesResp.Businesses, nil
}

func generateBusinessError(businesses []Business) string {
	if len(businesses) == 0 {
		// Fallback to regular error if no businesses available
		return errorMessages[rand.Intn(len(errorMessages))]
	}

	template := businessErrorTemplates[rand.Intn(len(businessErrorTemplates))]

	// Count placeholders in template
	placeholderCount := 0
	for i := 0; i < len(template); i++ {
		if template[i] == '%' && i+1 < len(template) && template[i+1] == 's' {
			placeholderCount++
		}
	}

	// Fill placeholders with random business names
	args := make([]interface{}, placeholderCount)
	for i := 0; i < placeholderCount; i++ {
		args[i] = businesses[rand.Intn(len(businesses))].Name
	}

	return fmt.Sprintf(template, args...)
}

func sendErrorLogToSloganServer(sloganServerURL string, errorLogRequest ErrorLogRequest) (*SloganResponse, error) {
	requestBody, err := json.Marshal(errorLogRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpResponse, err := http.Post(sloganServerURL+"/error-log", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("slogan server returned status: %d", httpResponse.StatusCode)
	}

	var sloganResponse SloganResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&sloganResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &sloganResponse, nil
}

func sendErrorLogToTracker(trackerURL string, message string, gifURL string, slogan string, songTitle string, songArtist string, songURL string) error {
	errorLog := map[string]string{
		"message":     message,
		"gif_url":     gifURL,
		"slogan":      slogan,
		"song_title":  songTitle,
		"song_artist": songArtist,
		"song_url":    songURL,
	}

	requestBody, err := json.Marshal(errorLog)
	if err != nil {
		return fmt.Errorf("failed to marshal error log: %w", err)
	}

	httpResponse, err := locationTrackerHTTPClient.Post(trackerURL+"/api/errorlogs", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to send to tracker: %w", err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("tracker returned status: %d", httpResponse.StatusCode)
	}

	return nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	giphyAPIKey := os.Getenv("GIPHY_API_KEY")
	spotifyClientID := os.Getenv("SPOTIFY_CLIENT_ID")
	spotifyClientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	spotifySeedGenres := os.Getenv("SPOTIFY_SEED_GENRES")

	sloganServerURL := os.Getenv("SLOGAN_SERVER_URL")
	if sloganServerURL == "" {
		sloganServerURL = "http://localhost:8080"
	}

	// Location tracker URL (optional)
	locationTrackerURL := os.Getenv("LOCATION_TRACKER_URL")

	intervalSeconds := 60.0
	if envInterval := os.Getenv("ERROR_INTERVAL_SECONDS"); envInterval != "" {
		fmt.Sscanf(envInterval, "%f", &intervalSeconds)
	}

	log.Printf("Error Generator starting...")
	log.Printf("Slogan server URL: %s", sloganServerURL)
	if locationTrackerURL != "" {
		log.Printf("Location tracker URL: %s", locationTrackerURL)
	}
	log.Printf("Sending errors every %.2f seconds", intervalSeconds)

	gifCache := newGifCache(giphyAPIKey)
	spotifyCache := newSpotifyCache(spotifyClientID, spotifyClientSecret, spotifySeedGenres)

	// Convert interval to duration (handle decimal seconds)
	intervalDuration := time.Duration(intervalSeconds * float64(time.Second))
	ticker := time.NewTicker(intervalDuration)
	defer ticker.Stop()

	generateAndSendError := func() {
		// Fetch current businesses from location tracker
		var errorMessage string
		businesses, err := fetchBusinesses(locationTrackerURL)
		if err != nil {
			log.Printf("⚠️  Error fetching businesses: %v", err)
			errorMessage = errorMessages[rand.Intn(len(errorMessages))]
		} else if len(businesses) > 0 {
			errorMessage = generateBusinessError(businesses)
			log.Printf("🏢 Using %d nearby businesses for error", len(businesses))
		} else {
			errorMessage = errorMessages[rand.Intn(len(errorMessages))]
		}

		gifURL := gifCache.getNextGif()
		song := spotifyCache.getNextSong()

		errorLogRequest := ErrorLogRequest{
			Message:    errorMessage,
			GifURL:     gifURL,
			SongTitle:  song.Title,
			SongArtist: song.Artist,
			SongURL:    song.URL,
		}

		log.Printf("Sending error: %s", errorMessage)
		log.Printf("With GIF: %s", gifURL)
		log.Printf("With Song: %s by %s", song.Title, song.Artist)

		sloganResponse, err := sendErrorLogToSloganServer(sloganServerURL, errorLogRequest)
		if err != nil {
			log.Printf("Error sending to slogan server: %v", err)
			return
		}

		log.Printf("Received response: %s %s", sloganResponse.Emoji, sloganResponse.Slogan)
		fmt.Printf("\n=== ERROR LOG ===\n")
		fmt.Printf("Error: %s\n", errorMessage)
		fmt.Printf("GIF: %s\n", gifURL)
		fmt.Printf("Song: %s by %s\n", song.Title, song.Artist)
		fmt.Printf("Song URL: %s\n", song.URL)
		fmt.Printf("Response: %s %s\n", sloganResponse.Emoji, sloganResponse.Slogan)
		fmt.Printf("================\n\n")

		// Send to location tracker if configured
		if locationTrackerURL != "" {
			if err := sendErrorLogToTracker(locationTrackerURL, errorMessage, gifURL, sloganResponse.Slogan, song.Title, song.Artist, song.URL); err != nil {
				log.Printf("Warning: Failed to send to location tracker: %v", err)
			} else {
				log.Printf("📍 Sent error log to location tracker")
			}
		}
	}

	generateAndSendError()

	for range ticker.C {
		generateAndSendError()
	}
}
