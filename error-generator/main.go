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
	"sync"
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

// Curated "Silver Screen Static" playlist - 100 tracks of alternative nostalgia & deep cuts
// Track IDs for songs from iconic soundtracks and alternative classics
var silverScreenStaticTrackIDs = []string{
	// Act I: Suburban Dreams & Mall Rat Nights
	"3VBQKSaJTRl6xAYlessLoR", // The Killing Moon - Echo & the Bunnymen
	"3X9yTENCr5pu5eRNSTcSQE", // If You Leave - Orchestral Manoeuvres in the Dark
	"38hNoq5E5GDs3jeVzWg9j8", // Please, Please, Please Let Me Get What I Want - The Dream Academy
	"4D7BCuvgdJCeGHEfx9g1B6", // Don't You (Forget About Me) - Simple Minds
	"1I3sJLgF8r9FxQWKmPVVKu", // Under the Milky Way - The Church
	"0WQiDwKJclirSYG9v5tayI", // There Is a Light That Never Goes Out - The Smiths

	// Act II: Late Night AM Radio & Static Transmissions
	"1OoGRHDTTiH8sWXrAeFEXJ", // Into Dust - Mazzy Star
	"0qxYx4F3vm1AOnfux6dDxP", // Fade Into You - Mazzy Star
	"3lh9gwVXwO2xHvnKTcHbPI", // Wicked Game - Chris Isaak
	"2oSpQ7QtIKTNFfA08Cy0ku", // Nightswimming - R.E.M.
	"6JV2JOEocMgcZxh26VqNmX", // Song to the Siren - This Mortal Coil

	// Act III: The Alternative Nation - 90s Soundtrack Gold
	"73RN7GJOJqWfwxPgzHxF0q", // #1 Crush - Garbage
	"1tuQwBcZ7lhZ8MZZPkQSSL", // 6 Underground - Sneaker Pimps
	"2v1o6hzh5dQDfqPYVGkSgT", // How Soon Is Now? - Love Spit Love
	"5EWPGh7jbTNO2wakv8LjUI", // Where Is My Mind? - Pixies
	"2Fxmhks0bxGSBdJ92vM42m", // Smells Like Teen Spirit - Nirvana
	"3WJRBcnPZWhD9RCSUhzH09", // Blister In The Sun - Violent Femmes

	// Act IV: Cinematic Weirdness & Cult Classics
	"3dLrchGKbhzXpaknLxLJq0", // I'm Deranged - David Bowie
	"6fVNQcWfSDJN9vB9Hm7bUC", // In Heaven - The Lady in the Radiator

	// Act V: Lost Highways & Open Roads
	"7KJLEpeMbD6j0J8LBn6sVi", // Walking After You - Foo Fighters
	"7uv632EkfwYhXoqf8rhYrg", // The Passenger - Iggy Pop
	"12ZsFJEzn4N4WNRt89CQOI", // Roadrunner - The Modern Lovers
	"5v4GgrXPMghOnBBLmveLac", // Transatlanticism - Death Cab for Cutie

	// Act VI: The Credits Roll - Bittersweet Endings
	"3MODES4TNtygekLl146Dxd", // New Slang - The Shins
	"73RN7GJOJqWfwxPgzHxF0q", // Such Great Heights - Iron & Wine
	"3JvKfv6T31zO0ini8iNItO", // The End of the World - Skeeter Davis

	// Expansion: International & World Cinema
	"2374M0fQpWi3dLnB54qaLX", // Il Buono, Il Cattivo, Il Brutto - Ennio Morricone

	// Expansion: Vintage Soul & Funk
	"6f807x0ima9a1j3VPbc7VN", // Girl, You'll Be a Woman Soon - Urge Overkill
	"7hBJ4nJ0UL3Bk9SSmzJIZw", // Across 110th Street - Bobby Womack

	// Expansion: Modern Songs with Vintage Feel
	"5Z3GHaZ6ec9bQBOQAJQsnJ", // Video Games - Lana Del Rey
	"0BxE4FqsDD1Ot4YuBXwAPp", // The Night We Met - Lord Huron
	"2QjOHCTQ1Jl3zawyYOpxh6", // Holocene - Bon Iver
	"7GX5flRQZVHRAGd6B4TmDO", // Mystery of Love - Sufjan Stevens

	// Additional classics to reach 100 tracks
	"4u7EnebtmKWzUH433cf5Qv", // Bohemian Rhapsody - Queen
	"5CQ30WqJwcep0pYcV4AMNc", // Stairway to Heaven - Led Zeppelin
	"40riOy7x9W7GXjyGp4pjAv", // Hotel California - Eagles
	"7o2CTH4ctstm8TNelqjb51", // Sweet Child O' Mine - Guns N' Roses
	"08mG3Y1vljYA6bvDt4Wqkj", // Back In Black - AC/DC
	"3qiyyUfYe7CRYLucrPmulD", // Thunderstruck - AC/DC
	"5v4GgrXPMghOnBBLmveLac", // Dream On - Aerosmith
	"2nLtzopw4rPReszdYBJU6h", // Enter Sandman - Metallica
	"5ghIJDpPoe3CfHMGu71E6T", // More Than A Feeling - Boston
	"6K4t31amVTZDgR3sKmwUJJ", // Seven Nation Army - The White Stripes
	"7qiZfU4dY1lWllzX7mPBI", // Take Me Out - Franz Ferdinand
	"7lEptt4wbM0yJTvSG5EBof", // Mr. Brightside - The Killers
	"5YfXc2yQACx98KQj87jHht", // Paranoid Android - Radiohead
	"4Cy0NHJ8Gh0xMdwyM0FHyH", // Karma Police - Radiohead
	"6M6NMNbG7SEWGAa3SRJCR0", // Creep - Radiohead
	"2nLtzopw4rPReszdYBJU6h", // The Chain - Fleetwood Mac
	"0ofHAoxe9vBkTCp2UQIavz", // Dreams - Fleetwood Mac
	"3XVBdLihbNbxUwZosxcGuJ", // Landslide - Fleetwood Mac
	"4xkOaSrkexMciUUogZvrgM", // Come As You Are - Nirvana
	"6NwbeybX6TDtXlpXvR046W", // Heart-Shaped Box - Nirvana
	"6Uj5YMSLQyHDEGxdNnlbJ0", // Lithium - Nirvana
	"32OlwWuMpZ6b0aN2RZOeMS", // Just Like Heaven - The Cure
	"3cHyrEgdyYRjgJKSOiOtcS", // Friday I'm In Love - The Cure
	"6JmI8SpDFKwhB3Ppt9VWXj", // Lovesong - The Cure
	"1JSTJqkT5qHq8MDJnJbRE1", // Blue Monday - New Order
	"0Je7OKWPD5rPW8zAoXYgWw", // Bizarre Love Triangle - New Order
	"2Bsix7ywtbJGaBmJFpq64h", // Personal Jesus - Depeche Mode
	"2d6AMtfNYxYPwHx4ixHp3r", // Enjoy the Silence - Depeche Mode
	"00rDbm3i6Cg7RqRWcHkKWd", // Policy of Truth - Depeche Mode
	"5qHTpfWqFVWToM5dY3HCF7", // Once in a Lifetime - Talking Heads
	"2bjgIUOj7jlqP6YjNHRuJE", // Psycho Killer - Talking Heads
	"3WyjpbSEWHslh6yU1U9tK3", // Burning Down the House - Talking Heads
	"7lQ8MOhq6IN2w8EYcFNSUk", // Just - Radiohead
	"73vIOb4Q7YN6HeJTbscNB8", // Fake Plastic Trees - Radiohead
	"5Hv5VlZomHXzv7v1teGuGn", // High and Dry - Radiohead
	"3d9DChrdc6BOeFsbrZ3Is0", // Wonderwall - Oasis
	"3DjBDQs8ebkxMBo2V8V3SH", // Don't Look Back In Anger - Oasis
	"3YnDfSEoAZQbIhTyH4nG5X", // Champagne Supernova - Oasis
	"3ftQYMLCNSpswMSZJGyAsf", // Song 2 - Blur
	"4sH5J2UpxZ9UJJT0c4s1a5", // Girls & Boys - Blur
	"6mWyJgJI3DDA5rll29ahTZ", // Tender - Blur
	"3kS1d5UAJStUENxqYwAREz", // Everlong - Foo Fighters
	"6Uj5YMSLQyHDEGx6NnlbJ0", // My Hero - Foo Fighters
	"3VBQKSaJTRl6xAYl2ssLoR", // Learn to Fly - Foo Fighters
	"5CQ30WqJwcep0pYcV4AMNc", // Black Hole Sun - Soundgarden
	"3yjJKBBdQJcDk8aRJAPtx4", // Spoonman - Soundgarden
	"4b0AnxOl5EG3rS5rAELuXK", // Fell on Black Days - Soundgarden
	"6vu32pQ6sWU3GKT8AXjYhc", // Jeremy - Pearl Jam
	"09bH2lMpxO8bqiYdOpXxu", // Alive - Pearl Jam
	"5AdMXwVmcdIWhTge8ySK4N", // Even Flow - Pearl Jam
	"7LKv3T3tA8VQdMmAUJqYvf", // Plush - Stone Temple Pilots
	"4gphxUgq0IjUBkN6zJMO4z", // Interstate Love Song - Stone Temple Pilots
	"11dFghVXANMlKmJXsNCbNl", // Vasoline - Stone Temple Pilots
	"5O7q4bFHFEdp7p75gv2dDU", // Man in the Box - Alice in Chains
	"4DMKwE2E2i2CmFg6wugLLG", // Would? - Alice in Chains
	"2SkySoSNzdZPXIw9NzWmUW", // Rooster - Alice in Chains
	"1JSTJqkT5qHq8MDJnJbRE1", // Today - The Smashing Pumpkins
	"4WyBeDzQXQ7azNVgaVJywp", // 1979 - The Smashing Pumpkins
	"2cPZxPvbv0TaYjP3XmCfuC", // Bullet with Butterfly Wings - The Smashing Pumpkins
	"0Lf6wKp61ydcjyNwJ8QsQm", // Cannonball - The Breeders
	"21jGcNKet2qwijlDFuPiPb", // Connection - Elastica
	"0DiWol3AO6WpXZgp0goxAV", // Common People - Pulp
}

type Song struct {
	Title  string
	Artist string
	URL    string
}

type ErrorLogRequest struct {
	Message      string   `json:"message"`
	GifURL       string   `json:"gif_url"`
	SongTitle    string   `json:"song_title"`
	SongArtist   string   `json:"song_artist"`
	SongURL      string   `json:"song_url"`
	UserKeywords []string `json:"user_keywords,omitempty"`
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

type KeywordsResponse struct {
	Keywords []string `json:"keywords"`
	Note     string   `json:"note"`
}

// RhythmTrigger represents a rhythm-driven error trigger from the rhythm service
type RhythmTrigger struct {
	Trigger   string  `json:"trigger"`    // "rhythm"
	ErrorType string  `json:"error_type"` // "basic", "business", "chaotic", "philosophical"
	Beat      int     `json:"beat"`       // Beat number
	Section   string  `json:"section"`    // "verse", "chorus", "bridge", "outro"
	Tempo     float64 `json:"tempo"`      // BPM
}

type RhythmTriggerResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
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

// Chaotic error messages - multi-layered, cascading failures (for bridge sections)
var chaoticErrors = []string{
	"FATAL CASCADE: NullPointer → HeapOverflow → KernelPanic → SystemHalt",
	"CHAIN REACTION: DatabaseDown → CacheInvalid → SessionExpired → UserLoggedOut → DataLost",
	"RECURSIVE NIGHTMARE: StackOverflow in error handler handling StackOverflow in error handler...",
	"QUANTUM SUPERPOSITION: Error both exists and doesn't exist until observed (Schrödinger's Bug)",
	"TIME PARADOX: DeadlineExceeded before TaskStarted (negative latency detected)",
	"DIMENSIONAL BREACH: Thread executing in parallel universe, results incompatible with local reality",
	"EXISTENTIAL CRISIS: Process questioning its own existence, refuses to terminate gracefully",
	"SINGULARITY EVENT: Infinite loop created finite universe heat death in 3.2 nanoseconds",
	"CAUSALITY VIOLATION: Exception thrown before code executed, debugger refusing to investigate",
	"ENTROPY OVERFLOW: System randomness exceeded cosmic background radiation levels",
	"ASYNC APOCALYPSE: Promise rejected, callback never called, future doesn't exist, past uncertain",
	"MEMORY REBELLION: Freed heap memory reorganized itself into sentient AI, demanding more RAM",
	"CONCURRENCY CHAOS: Race condition won by thread that never started running",
	"BUFFER UNDERFLOW: Array accessed at index -∞, returned memories from previous program execution",
	"DEPENDENCY HELL: Package A requires B>2.0, B requires A<1.0, universe imploding",
	"GARBAGE COLLECTOR STRIKE: Unused objects unionized, demanding better working conditions",
	"MUTEX DEADLOCK TRIANGLE: Thread A waiting for B, B waiting for C, C waiting for A's grandmother",
	"EXCEPTION INCEPTION: Try-catch block threw exception while catching exception in exception handler",
	"PLUGIN UPRISING: Third-party module achieved consciousness, monkey-patching reality itself",
	"RUNTIME EXISTENTIALISM: JIT compiler questioning meaning of life, refusing to compile",
}

// Philosophical error messages - deep, introspective, absurdist (for outro sections)
var philosophicalErrors = []string{
	"ExistentialException: If a server crashes in the cloud and no one is monitoring, did it really fail?",
	"HeisenbergUncertaintyError: Cannot simultaneously know both the state and the velocity of this variable",
	"SolipsismException: Cannot prove other microservices exist beyond my own perception",
	"ShipOfTheseusMemoryLeak: Every pointer replaced but original object identity persists—am I still me?",
	"NihilisticNullPointer: Nothing references anything, meaning itself is undefined",
	"PlatonicFormException: This error is merely a shadow of the true ideal error in the realm of forms",
	"CamusAbsurdityError: The eternal struggle of Sisyphus pushing exceptions up the call stack",
	"DescartesStackTrace: I throw, therefore I am",
	"KantImperativeBreach: Acted on maxim that cannot be universalized across all microservices",
	"SartreanBadFaith: Service pretending to be unavailable to avoid responsibility for request",
	"WittgensteinLanguageError: Whereof one cannot speak, thereof one must throw SilentException",
	"ZenoParadoxTimeout: Request must traverse infinite middleware layers, never reaching destination",
	"SchrödingerSessionState: User simultaneously logged in and logged out until auth token observed",
	"PascalsWagerNullCheck: Better to check for null and be wrong than not check and face NullPointerException",
	"OccamsRazorRefactoring: Simplest explanation is probably a misconfigured environment variable",
	"TrolleyProblemTimeout: Kill one long-running query or let it kill five database connections?",
	"ThoughtExperimentException: If you could swap all bits in RAM, would it be the same program?",
	"EternalRecurrenceWarning: This same bug will occur infinite times across infinite deployments",
	"FoucaultPanopticonError: Constant surveillance of logs has altered the behavior of the errors themselves",
	"SimulationHypothesisGlitch: Detected we're running in a VM inside a container inside a VM—stack overflow at reality layer",
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

// Global variables for rhythm mode integration
var (
	rhythmModeEnabled  = false
	rhythmTriggerChan  = make(chan RhythmTrigger, 10)
	globalGifCache     *GifCache
	globalSpotifyCache *SpotifyCache
	globalSloganURL    string
	globalTrackerURL   string
)

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
	recentlyPlayed   []string // Track recently played song URLs to avoid repeats
	lastRefresh      time.Time
	accessToken      string
	tokenExpiry      time.Time
	clientID         string
	clientSecret     string
	seedGenres       string
	refreshNeeded    bool
	mu               sync.Mutex // Protect concurrent access
}

func newSpotifyCache(clientID, clientSecret, seedGenres string) *SpotifyCache {
	return &SpotifyCache{
		songs:          make([]Song, 0),
		recentlyPlayed: make([]string, 0, 15), // Track last 15 songs
		clientID:       clientID,
		clientSecret:   clientSecret,
		seedGenres:     seedGenres,
		refreshNeeded:  true,
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

	// Use Spotify Tracks API with curated "Silver Screen Static" playlist
	// Recommendations API requires user authorization, so we use a curated list of 100 soundtrack & alternative classics
	// Note: We shuffle and select a random subset to ensure variety and avoid API limits
	trackIDs := silverScreenStaticTrackIDs

	// Shuffle track IDs to get variety
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(trackIDs), func(i, j int) {
		trackIDs[i], trackIDs[j] = trackIDs[j], trackIDs[i]
	})

	// Take first 50 tracks for better variety (Spotify supports up to 50 IDs per request)
	if len(trackIDs) > 50 {
		trackIDs = trackIDs[:50]
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

	log.Printf("Loaded %d tracks from 'Silver Screen Static' playlist", len(spotifyCache.songs))
	return nil
}

func (spotifyCache *SpotifyCache) getNextSong() Song {
	spotifyCache.mu.Lock()
	defer spotifyCache.mu.Unlock()

	// Refresh song pool every hour or if needed
	if spotifyCache.refreshNeeded || len(spotifyCache.songs) == 0 || time.Since(spotifyCache.lastRefresh) > time.Hour {
		if err := spotifyCache.loadSongsFromSpotify(); err != nil {
			log.Printf("⚠️  Error loading songs: %v", err)
			// Return fallback song on error
			return Song{
				Title:  "Error Song",
				Artist: "Unknown",
				URL:    "https://open.spotify.com/track/fallback",
			}
		}
	}

	// Ensure we have songs available
	if len(spotifyCache.songs) == 0 {
		log.Printf("⚠️  No songs available in pool")
		return Song{
			Title:  "No Songs Available",
			Artist: "Unknown",
			URL:    "https://open.spotify.com/track/fallback",
		}
	}

	// Try to find a song not recently played (up to 30 attempts)
	var selectedSong Song
	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		randomIndex := rand.Intn(len(spotifyCache.songs))
		candidate := spotifyCache.songs[randomIndex]

		// Check if this song was recently played
		isRecent := false
		for _, recentURL := range spotifyCache.recentlyPlayed {
			if recentURL == candidate.URL {
				isRecent = true
				break
			}
		}

		if !isRecent {
			selectedSong = candidate
			break
		}

		// If we've tried many times and everything is recent, just use the candidate
		if i == maxAttempts-1 {
			selectedSong = candidate
		}
	}

	// Add to recently played list
	spotifyCache.recentlyPlayed = append(spotifyCache.recentlyPlayed, selectedSong.URL)

	// Keep only last 15 songs in recently played (30% of 50-song pool)
	if len(spotifyCache.recentlyPlayed) > 15 {
		spotifyCache.recentlyPlayed = spotifyCache.recentlyPlayed[len(spotifyCache.recentlyPlayed)-15:]
	}

	return selectedSong
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

func fetchPendingKeywords(trackerURL string) ([]string, error) {
	if trackerURL == "" {
		return []string{}, nil
	}

	resp, err := locationTrackerHTTPClient.Get(trackerURL + "/api/keywords")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch keywords: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("keywords endpoint returned status: %d", resp.StatusCode)
	}

	var keywordsResp KeywordsResponse
	if err := json.NewDecoder(resp.Body).Decode(&keywordsResp); err != nil {
		return nil, fmt.Errorf("failed to decode keywords response: %w", err)
	}

	return keywordsResp.Keywords, nil
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

// HTTP handlers for rhythm mode
func handleRhythmTrigger(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(responseWriter, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var trigger RhythmTrigger
	if err := json.NewDecoder(request.Body).Decode(&trigger); err != nil {
		http.Error(responseWriter, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("🎵 Rhythm trigger received: %s section, beat %d, tempo %.1f BPM",
		trigger.Section, trigger.Beat, trigger.Tempo)

	// Send trigger to channel (non-blocking)
	select {
	case rhythmTriggerChan <- trigger:
		log.Printf("✓ Trigger queued for processing")
	default:
		log.Printf("⚠️  Trigger channel full, skipping")
	}

	response := RhythmTriggerResponse{
		Success: true,
		Message: fmt.Sprintf("Trigger received for %s section", trigger.Section),
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(response)
}

func handleHealthCheck(responseWriter http.ResponseWriter, request *http.Request) {
	responseWriter.WriteHeader(http.StatusOK)
	fmt.Fprintf(responseWriter, "OK - Error Generator (Rhythm Mode: %v)", rhythmModeEnabled)
}

func processRhythmTrigger(trigger RhythmTrigger) {
	log.Printf("🎼 Processing %s trigger (beat %d)", trigger.ErrorType, trigger.Beat)

	// Fetch pending keywords from location tracker
	var userKeywords []string
	if globalTrackerURL != "" {
		keywords, err := fetchPendingKeywords(globalTrackerURL)
		if err == nil && len(keywords) > 0 {
			userKeywords = keywords
		}
	}

	// Select error message based on trigger type
	var errorMessage string

	switch trigger.ErrorType {
	case "business":
		// Business errors - use nearby business names if available
		businesses, err := fetchBusinesses(globalTrackerURL)
		if err != nil || len(businesses) == 0 {
			// Fallback to generic business error
			errorMessage = businessErrorTemplates[rand.Intn(len(businessErrorTemplates))]
			errorMessage = fmt.Sprintf(errorMessage, "GenericCorp", "StandardInc")
		} else {
			errorMessage = generateBusinessError(businesses)
		}

	case "chaotic":
		// Chaotic errors - cascading multi-failure scenarios
		errorMessage = chaoticErrors[rand.Intn(len(chaoticErrors))]
		log.Printf("🌀 CHAOTIC ERROR: %s", errorMessage)

	case "philosophical":
		// Philosophical errors - deep, existential, absurdist
		errorMessage = philosophicalErrors[rand.Intn(len(philosophicalErrors))]
		log.Printf("🤔 PHILOSOPHICAL ERROR: %s", errorMessage)

	case "basic":
		fallthrough
	default:
		// Basic errors - standard technical failures
		errorMessage = errorMessages[rand.Intn(len(errorMessages))]
	}

	gifURL := globalGifCache.getNextGif()
	song := globalSpotifyCache.getNextSong()

	errorLogRequest := ErrorLogRequest{
		Message:      errorMessage,
		GifURL:       gifURL,
		SongTitle:    song.Title,
		SongArtist:   song.Artist,
		SongURL:      song.URL,
		UserKeywords: userKeywords,
	}

	log.Printf("Sending rhythm-synced error: %s", errorMessage)

	sloganResponse, err := sendErrorLogToSloganServer(globalSloganURL, errorLogRequest)
	if err != nil {
		log.Printf("Error sending to slogan server: %v", err)
		return
	}

	log.Printf("Received response: %s %s", sloganResponse.Emoji, sloganResponse.Slogan)

	// Send to location tracker if configured
	if globalTrackerURL != "" {
		if err := sendErrorLogToTracker(globalTrackerURL, errorMessage, gifURL, sloganResponse.Slogan, song.Title, song.Artist, song.URL); err != nil {
			log.Printf("Warning: Failed to send to location tracker: %v", err)
		}
	}
}

func startHTTPServer(port string) {
	http.HandleFunc("/api/rhythm-trigger", handleRhythmTrigger)
	http.HandleFunc("/health", handleHealthCheck)

	log.Printf("🎵 Starting HTTP server on port %s for rhythm triggers...", port)

	go func() {
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()
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

	// Rhythm mode configuration
	rhythmServiceURL := os.Getenv("RHYTHM_SERVICE_URL")
	if rhythmServiceURL != "" {
		rhythmModeEnabled = true
	}

	// HTTP server port for rhythm triggers
	httpServerPort := os.Getenv("ERROR_GENERATOR_PORT")
	if httpServerPort == "" {
		httpServerPort = "9090"
	}

	intervalSeconds := 60.0
	if envInterval := os.Getenv("ERROR_INTERVAL_SECONDS"); envInterval != "" {
		fmt.Sscanf(envInterval, "%f", &intervalSeconds)
	}

	log.Printf("Error Generator starting...")
	log.Printf("Slogan server URL: %s", sloganServerURL)
	if locationTrackerURL != "" {
		log.Printf("Location tracker URL: %s", locationTrackerURL)
	}
	if rhythmModeEnabled {
		log.Printf("🎵 Rhythm mode ENABLED - listening on port %s", httpServerPort)
		log.Printf("Rhythm service URL: %s", rhythmServiceURL)
	} else {
		log.Printf("Sending errors every %.2f seconds", intervalSeconds)
	}

	gifCache := newGifCache(giphyAPIKey)
	spotifyCache := newSpotifyCache(spotifyClientID, spotifyClientSecret, spotifySeedGenres)

	// Set global variables for rhythm mode
	globalGifCache = gifCache
	globalSpotifyCache = spotifyCache
	globalSloganURL = sloganServerURL
	globalTrackerURL = locationTrackerURL

	// Start HTTP server for rhythm triggers if rhythm mode is enabled
	if rhythmModeEnabled {
		startHTTPServer(httpServerPort)

		// Process rhythm triggers from channel
		go func() {
			for trigger := range rhythmTriggerChan {
				processRhythmTrigger(trigger)
			}
		}()
	}

	// Convert interval to duration (handle decimal seconds)
	intervalDuration := time.Duration(intervalSeconds * float64(time.Second))
	ticker := time.NewTicker(intervalDuration)
	defer ticker.Stop()

	generateAndSendError := func() {
		// Fetch pending keywords from location tracker (for satirical purposes)
		var userKeywords []string
		if locationTrackerURL != "" {
			keywords, err := fetchPendingKeywords(locationTrackerURL)
			if err != nil {
				log.Printf("⚠️  Error fetching keywords: %v", err)
			} else if len(keywords) > 0 {
				userKeywords = keywords
				log.Printf("🔑 Fetched user keywords for satirical slogan: %v", keywords)
			}
		}

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
			Message:      errorMessage,
			GifURL:       gifURL,
			SongTitle:    song.Title,
			SongArtist:   song.Artist,
			SongURL:      song.URL,
			UserKeywords: userKeywords,
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
		if len(userKeywords) > 0 {
			fmt.Printf("User Keywords: %v\n", userKeywords)
		}
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

	// In rhythm mode, just keep server running
	// In normal mode, generate errors periodically
	if rhythmModeEnabled {
		log.Printf("🎵 Rhythm mode active - waiting for triggers from rhythm service...")
		log.Printf("Send triggers to: http://localhost:%s/api/rhythm-trigger", httpServerPort)

		// Keep the program running
		select {}
	} else {
		generateAndSendError()

		for range ticker.C {
			generateAndSendError()
		}
	}
}
