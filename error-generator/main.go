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

type ErrorLogRequest struct {
	Message string `json:"message"`
	GifURL  string `json:"gif_url"`
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

func main() {
	rand.Seed(time.Now().UnixNano())

	giphyAPIKey := os.Getenv("GIPHY_API_KEY")
	sloganServerURL := os.Getenv("SLOGAN_SERVER_URL")
	if sloganServerURL == "" {
		sloganServerURL = "http://localhost:8080"
	}

	intervalSeconds := 60
	if envInterval := os.Getenv("ERROR_INTERVAL_SECONDS"); envInterval != "" {
		fmt.Sscanf(envInterval, "%d", &intervalSeconds)
	}

	log.Printf("Error Generator starting...")
	log.Printf("Slogan server URL: %s", sloganServerURL)
	log.Printf("Sending errors every %d seconds", intervalSeconds)

	gifCache := newGifCache(giphyAPIKey)

	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	generateAndSendError := func() {
		randomErrorMessage := errorMessages[rand.Intn(len(errorMessages))]
		gifURL := gifCache.getNextGif()

		errorLogRequest := ErrorLogRequest{
			Message: randomErrorMessage,
			GifURL:  gifURL,
		}

		log.Printf("Sending error: %s", randomErrorMessage)
		log.Printf("With GIF: %s", gifURL)

		sloganResponse, err := sendErrorLogToSloganServer(sloganServerURL, errorLogRequest)
		if err != nil {
			log.Printf("Error sending to slogan server: %v", err)
			return
		}

		log.Printf("Received response: %s %s", sloganResponse.Emoji, sloganResponse.Slogan)
		fmt.Printf("\n=== ERROR LOG ===\n")
		fmt.Printf("Error: %s\n", randomErrorMessage)
		fmt.Printf("GIF: %s\n", gifURL)
		fmt.Printf("Response: %s %s\n", sloganResponse.Emoji, sloganResponse.Slogan)
		fmt.Printf("================\n\n")
	}

	generateAndSendError()

	for range ticker.C {
		generateAndSendError()
	}
}
