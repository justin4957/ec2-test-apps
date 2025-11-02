package main

import (
	"bytes"
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

type SpotifyPlaylistResponse struct {
	Items []struct {
		Track struct {
			Name        string `json:"name"`
			ExternalURL struct {
				Spotify string `json:"spotify"`
			} `json:"external_urls"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"track"`
	} `json:"items"`
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
	playlistID       string
	refreshNeeded    bool
}

func newSpotifyCache(clientID, clientSecret, playlistID string) *SpotifyCache {
	return &SpotifyCache{
		songs:         make([]Song, 0),
		clientID:      clientID,
		clientSecret:  clientSecret,
		playlistID:    playlistID,
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

func (spotifyCache *SpotifyCache) loadSongsFromPlaylist() error {
	if spotifyCache.playlistID == "" {
		log.Println("SPOTIFY_PLAYLIST_ID not set, using placeholder songs")
		spotifyCache.songs = []Song{
			{Title: "Bohemian Rhapsody", Artist: "Queen", URL: "https://open.spotify.com/track/placeholder1"},
			{Title: "Stairway to Heaven", Artist: "Led Zeppelin", URL: "https://open.spotify.com/track/placeholder2"},
			{Title: "Hotel California", Artist: "Eagles", URL: "https://open.spotify.com/track/placeholder3"},
			{Title: "Imagine", Artist: "John Lennon", URL: "https://open.spotify.com/track/placeholder4"},
			{Title: "Smells Like Teen Spirit", Artist: "Nirvana", URL: "https://open.spotify.com/track/placeholder5"},
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

	playlistURL := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks?limit=50", spotifyCache.playlistID)

	httpRequest, err := http.NewRequest("GET", playlistURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create playlist request: %w", err)
	}

	httpRequest.Header.Set("Authorization", "Bearer "+spotifyCache.accessToken)

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("failed to fetch playlist: %w", err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("spotify API returned status: %d", httpResponse.StatusCode)
	}

	var playlistResponse SpotifyPlaylistResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&playlistResponse); err != nil {
		return fmt.Errorf("failed to decode playlist response: %w", err)
	}

	spotifyCache.songs = make([]Song, 0, len(playlistResponse.Items))
	for _, item := range playlistResponse.Items {
		if item.Track.Name != "" {
			artistName := "Unknown Artist"
			if len(item.Track.Artists) > 0 {
				artistName = item.Track.Artists[0].Name
			}
			spotifyCache.songs = append(spotifyCache.songs, Song{
				Title:  item.Track.Name,
				Artist: artistName,
				URL:    item.Track.ExternalURL.Spotify,
			})
		}
	}

	spotifyCache.lastRefresh = time.Now()
	spotifyCache.currentIndex = 0
	spotifyCache.refreshNeeded = false

	log.Printf("Loaded %d songs from Spotify playlist", len(spotifyCache.songs))
	return nil
}

func (spotifyCache *SpotifyCache) getNextSong() Song {
	if spotifyCache.refreshNeeded || len(spotifyCache.songs) == 0 {
		if err := spotifyCache.loadSongsFromPlaylist(); err != nil {
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

	httpResponse, err := http.Post(trackerURL+"/api/errorlogs", "application/json", bytes.NewBuffer(requestBody))
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
	spotifyPlaylistID := os.Getenv("SPOTIFY_PLAYLIST_ID")

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
	spotifyCache := newSpotifyCache(spotifyClientID, spotifyClientSecret, spotifyPlaylistID)

	// Convert interval to duration (handle decimal seconds)
	intervalDuration := time.Duration(intervalSeconds * float64(time.Second))
	ticker := time.NewTicker(intervalDuration)
	defer ticker.Stop()

	generateAndSendError := func() {
		randomErrorMessage := errorMessages[rand.Intn(len(errorMessages))]
		gifURL := gifCache.getNextGif()
		song := spotifyCache.getNextSong()

		errorLogRequest := ErrorLogRequest{
			Message:    randomErrorMessage,
			GifURL:     gifURL,
			SongTitle:  song.Title,
			SongArtist: song.Artist,
			SongURL:    song.URL,
		}

		log.Printf("Sending error: %s", randomErrorMessage)
		log.Printf("With GIF: %s", gifURL)
		log.Printf("With Song: %s by %s", song.Title, song.Artist)

		sloganResponse, err := sendErrorLogToSloganServer(sloganServerURL, errorLogRequest)
		if err != nil {
			log.Printf("Error sending to slogan server: %v", err)
			return
		}

		log.Printf("Received response: %s %s", sloganResponse.Emoji, sloganResponse.Slogan)
		fmt.Printf("\n=== ERROR LOG ===\n")
		fmt.Printf("Error: %s\n", randomErrorMessage)
		fmt.Printf("GIF: %s\n", gifURL)
		fmt.Printf("Song: %s by %s\n", song.Title, song.Artist)
		fmt.Printf("Song URL: %s\n", song.URL)
		fmt.Printf("Response: %s %s\n", sloganResponse.Emoji, sloganResponse.Slogan)
		fmt.Printf("================\n\n")

		// Send to location tracker if configured
		if locationTrackerURL != "" {
			if err := sendErrorLogToTracker(locationTrackerURL, randomErrorMessage, gifURL, sloganResponse.Slogan, song.Title, song.Artist, song.URL); err != nil {
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
