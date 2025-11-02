package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

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

type OpenAIRequest struct {
	Model    string                   `json:"model"`
	Messages []OpenAIMessage          `json:"messages"`
	MaxTokens int                     `json:"max_tokens"`
	Temperature float64               `json:"temperature"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message OpenAIMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

var nonsensicalSlogans = []string{
	"Your error is someone's feature!",
	"Bugs: The spice of development life",
	"Error 404: Motivation not found",
	"Keep calm and blame the compiler",
	"Undefined is just a state of mind",
	"Segfault: A journey into the unknown",
	"Memory leak? More like memory waterfall",
	"Stack overflow: Because recursion loves company",
	"Null pointer: The void stares back",
	"Race condition: Speed dating for threads",
	"Deadlock: When threads fall in love forever",
	"Heap corruption: Modern art for memory",
	"Buffer overflow: Living life on the edge",
	"Out of bounds: Breaking free from constraints",
	"Type mismatch: Celebrate diversity",
	"Syntax error: Poetry the compiler doesn't understand",
	"Logic error: Alternative facts in code",
	"Off by one: Close enough is good enough",
	"Infinite loop: The circle of life",
	"Timeout: Patience is overrated",
	"Connection refused: Playing hard to get",
	"Permission denied: No means maybe later",
	"Resource exhausted: Living beyond your means",
	"Assertion failed: Reality is negotiable",
	"Invalid argument: Agreeing to disagree",
	"Exception: The rule that proves itself",
	"Panic: Excitement in disguise",
	"Fatal error: Drama queen energy",
	"Core dumped: Sharing is caring",
	"Zombie process: The undead of computing",
	"Orphan process: Independence day every day",
	"Fork bomb: Exponential family growth",
	"Kernel panic: Operating system's existential crisis",
	"Blue screen: Windows' way of saying hello",
	"Guru meditation: Enlightenment through crashes",
	"Bus error: Wrong stop, right destination",
	"Illegal instruction: Breaking the law, breaking the law",
	"Floating point exception: Math gone wild",
	"Integer overflow: More is more is more",
	"Division by zero: Infinity at your fingertips",
	"Uninitialized variable: Mystery box programming",
	"Memory corruption: Spicy randomness",
	"Use after free: Living dangerously",
	"Double free: Twice the fun",
	"Stack smashing: Aggressive optimization",
	"Heap spray: Artistic memory arrangement",
	"Format string vulnerability: Creative formatting",
	"SQL injection: Bonus query features",
	"XSS: Extra script support",
	"CSRF: Surprise requests",
	"Path traversal: Filesystem tourism",
	"Remote code execution: Sharing is caring",
	"Privilege escalation: Career advancement",
	"Authentication bypass: VIP treatment",
	"Broken access control: Open door policy",
	"Security misconfiguration: Artistic freedom",
	"Sensitive data exposure: Radical transparency",
	"XML external entities: Make new friends",
	"Deserialization: Unboxing surprise objects",
	"Insecure components: Vintage dependencies",
	"Insufficient logging: Mystery novel mode",
	"API abuse: Enthusiastic usage",
	"Brute force: Determined persistence",
	"DDoS: Overwhelming popularity",
	"Man in the middle: Third wheel networking",
	"Session hijacking: Friendly takeover",
	"Clickjacking: Surprise interactions",
	"Cookie poisoning: Spicy snacks",
	"DNS spoofing: Identity exploration",
	"ARP poisoning: Network personality disorder",
	"Port scanning: Neighborly curiosity",
	"Packet sniffing: Network aromatherapy",
	"Replay attack: Nostalgia in action",
	"Zero day: Fresh out of the oven",
	"Exploit: Feature unlock code",
	"Payload: Special delivery",
	"Rootkit: Deep system integration",
	"Trojan: Surprise software bundle",
	"Worm: Self-motivated traveler",
	"Virus: Social butterfly code",
	"Ransomware: Aggressive data backup",
	"Spyware: Overly attached software",
	"Adware: Enthusiastic marketing",
	"Botnet: Distributed friendship",
	"Backdoor: Alternative entrance",
	"Logic bomb: Delayed surprise party",
	"Time bomb: Countdown to excitement",
	"Keylogger: Thorough documentation",
	"Screen scraper: Visual collector",
	"Phishing: Optimistic communications",
	"Vishing: Voice of opportunity",
	"Smishing: Texting enthusiasm",
	"Pretexting: Creative storytelling",
	"Baiting: Generous offers",
	"Quid pro quo: Fair exchange philosophy",
	"Tailgating: Close following",
	"Shoulder surfing: Over-the-shoulder learning",
	"Dumpster diving: Recycling enthusiasm",
	"Social engineering: People skills",
	"Password cracking: Lock picking hobby",
	"Rainbow table: Colorful data structures",
	"Hash collision: Cryptographic coincidence",
	"Certificate error: Trust issues",
	"Encryption failed: Privacy is optional",
	"Decryption failed: Mystery preservation",
	"Key exchange failed: Awkward handshake",
	"Handshake failed: Social anxiety",
	"Protocol error: Miscommunication art",
	"Malformed request: Creative formatting",
	"Bad gateway: Confused intermediary",
	"Service unavailable: Taking a break",
	"Gateway timeout: Fashionably late",
	"Network unreachable: Playing hide and seek",
	"Host unreachable: The ultimate introvert",
	"Connection reset: Starting fresh",
}

var (
	openaiAPIKey string
	httpClient   = &http.Client{Timeout: 10 * time.Second}
)

func generateSloganWithOpenAI(errorMessage string, gifURL string) (string, error) {
	if openaiAPIKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	// Extract GIF context from URL if possible
	gifContext := extractGifContext(gifURL)

	// Build prompt
	prompt := fmt.Sprintf(`Generate a single sardonic, darkly humorous advertising slogan for this error message. The slogan should be in the style of an absurdist marketing campaign - treating errors as desirable features.

Error: %s
%s

Requirements:
- Maximum 10 words
- Sardonic and darkly funny
- In the format "ErrorType: Witty phrase"
- Similar to slogans like "Buffer overflow: Living life on the edge" or "Null pointer: The void stares back"
- Reference the error context if relevant

Respond with ONLY the slogan, nothing else.`, errorMessage, gifContext)

	reqBody := OpenAIRequest{
		Model: "gpt-4o-mini",
		Messages: []OpenAIMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   50,
		Temperature: 0.9,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openaiAPIKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var openaiResp OpenAIResponse
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if openaiResp.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s", openaiResp.Error.Message)
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenAI response")
	}

	slogan := strings.TrimSpace(openaiResp.Choices[0].Message.Content)
	// Remove quotes if present
	slogan = strings.Trim(slogan, "\"'")

	return slogan, nil
}

func extractGifContext(gifURL string) string {
	if gifURL == "" {
		return ""
	}

	// Try to extract meaningful context from GIF URL
	// Giphy URLs often have descriptive text in them
	parts := strings.Split(gifURL, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// Remove GIF ID and extract description
		descParts := strings.Split(lastPart, "-")
		if len(descParts) > 1 {
			// Join the descriptive parts
			desc := strings.Join(descParts[:len(descParts)-1], " ")
			if desc != "" {
				return fmt.Sprintf("GIF context: %s", desc)
			}
		}
	}

	return ""
}

func getFallbackSlogan() string {
	return nonsensicalSlogans[rand.Intn(len(nonsensicalSlogans))]
}

func handleErrorLog(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(responseWriter, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var errorLogRequest ErrorLogRequest
	if err := json.NewDecoder(request.Body).Decode(&errorLogRequest); err != nil {
		http.Error(responseWriter, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Received error log: %s (GIF: %s)", errorLogRequest.Message, errorLogRequest.GifURL)

	var slogan string
	var sloganSource string

	// Try OpenAI first
	if openaiAPIKey != "" {
		generatedSlogan, err := generateSloganWithOpenAI(errorLogRequest.Message, errorLogRequest.GifURL)
		if err != nil {
			log.Printf("OpenAI generation failed, using fallback: %v", err)
			slogan = getFallbackSlogan()
			sloganSource = "fallback"
		} else {
			slogan = generatedSlogan
			sloganSource = "openai"
		}
	} else {
		slogan = getFallbackSlogan()
		sloganSource = "fallback"
	}

	sloganResponse := SloganResponse{
		Emoji:  "🚬",
		Slogan: slogan,
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(sloganResponse)

	log.Printf("Responded with slogan (%s): %s", sloganSource, slogan)
}

func healthCheck(responseWriter http.ResponseWriter, request *http.Request) {
	responseWriter.WriteHeader(http.StatusOK)
	fmt.Fprintf(responseWriter, "OK")
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Load OpenAI API key from environment
	openaiAPIKey = os.Getenv("OPENAI_API_KEY")

	if openaiAPIKey != "" {
		log.Printf("OpenAI API key configured, will generate slogans dynamically")
	} else {
		log.Printf("OpenAI API key not set, using fallback slogans only")
	}

	log.Printf("Loaded %d fallback slogans", len(nonsensicalSlogans))

	http.HandleFunc("/error-log", handleErrorLog)
	http.HandleFunc("/health", healthCheck)

	port := "8080"
	log.Printf("Slogan server starting on port %s", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
