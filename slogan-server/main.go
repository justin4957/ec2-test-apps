package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type ErrorLogRequest struct {
	Message string `json:"message"`
	GifURL  string `json:"gif_url"`
}

type SloganResponse struct {
	Emoji  string `json:"emoji"`
	Slogan string `json:"slogan"`
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

	randomSlogan := nonsensicalSlogans[rand.Intn(len(nonsensicalSlogans))]

	sloganResponse := SloganResponse{
		Emoji:  "🚬",
		Slogan: randomSlogan,
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(sloganResponse)

	log.Printf("Responded with slogan: %s", randomSlogan)
}

func healthCheck(responseWriter http.ResponseWriter, request *http.Request) {
	responseWriter.WriteHeader(http.StatusOK)
	fmt.Fprintf(responseWriter, "OK")
}

func main() {
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/error-log", handleErrorLog)
	http.HandleFunc("/health", healthCheck)

	port := "8080"
	log.Printf("Slogan server starting on port %s", port)
	log.Printf("Loaded %d nonsensical slogans", len(nonsensicalSlogans))

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
