package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// DailyCryptogram represents the cryptogram puzzle for a day
type DailyCryptogram struct {
	Date            string
	PlainText       string
	CipherText      string
	BookTitle       string
	BookAuthor      string
	BookDescription string
	HintKeywords    []string
	HintNumbers     []int
	BookCover       string
	SubstitutionMap map[rune]rune
}

// GoogleBooksResponse represents the API response structure
type GoogleBooksResponse struct {
	Items []struct {
		VolumeInfo struct {
			Title       string   `json:"title"`
			Authors     []string `json:"authors"`
			Description string   `json:"description"`
			ImageLinks  struct {
				Thumbnail string `json:"thumbnail"`
			} `json:"imageLinks"`
		} `json:"volumeInfo"`
	} `json:"items"`
}

// Book represents a simplified book structure
type Book struct {
	Title       string
	Authors     []string
	Description string
	Thumbnail   string
}

var (
	currentCryptogram *DailyCryptogram
	cryptogramCache   = make(map[string]*DailyCryptogram) // date -> cryptogram
)

// GetTodaysCryptogram returns the cryptogram for today, generating it if needed
func GetTodaysCryptogram() (*DailyCryptogram, error) {
	today := time.Now().Format("2006-01-02")

	// Check cache first
	if cached, exists := cryptogramCache[today]; exists {
		return cached, nil
	}

	// Generate new cryptogram
	crypto, err := generateDailyCryptogram(today)
	if err != nil {
		return nil, err
	}

	// Cache it
	cryptogramCache[today] = crypto
	currentCryptogram = crypto

	return crypto, nil
}

// generateDailyCryptogram creates a new cryptogram based on the date
func generateDailyCryptogram(date string) (*DailyCryptogram, error) {
	// Use date as seed for deterministic randomness
	seed := hashDateToSeed(date)
	rng := rand.New(rand.NewSource(seed))

	// Fetch a book from Google Books API
	book, err := fetchDailyBook(date, rng)
	if err != nil {
		// Fallback if API fails
		return generateFallbackCryptogram(date, rng), nil
	}

	// Extract keywords from book description (words longer than 5 chars)
	keywords := extractKeywords(book.Description, 3)

	// Generate hint numbers (page numbers, chapter numbers, etc.)
	hintNumbers := []int{
		rng.Intn(20) + 1,  // Chapter number (1-20)
		rng.Intn(300) + 1, // Page number (1-300)
	}

	// Create a cryptogram message that references the book
	plainText := generateCryptogramMessage(book.Title, keywords, hintNumbers, rng)

	// Generate substitution cipher
	substitutionMap := generateSubstitutionCipher(rng)
	cipherText := applyCipher(plainText, substitutionMap)

	author := ""
	if len(book.Authors) > 0 {
		author = strings.Join(book.Authors, ", ")
	}

	return &DailyCryptogram{
		Date:            date,
		PlainText:       plainText,
		CipherText:      cipherText,
		BookTitle:       book.Title,
		BookAuthor:      author,
		BookDescription: truncateString(book.Description, 200),
		HintKeywords:    keywords,
		HintNumbers:     hintNumbers,
		BookCover:       book.Thumbnail,
		SubstitutionMap: substitutionMap,
	}, nil
}

// fetchDailyBook fetches a book from Google Books API based on the date
func fetchDailyBook(date string, rng *rand.Rand) (*Book, error) {
	// List of interesting search terms to rotate through
	searchTerms := []string{
		"mystery", "adventure", "science", "history", "philosophy",
		"technology", "art", "fiction", "biography", "psychology",
	}

	// Pick a search term based on the date
	termIndex := int(hashDateToSeed(date) % int64(len(searchTerms)))
	searchTerm := searchTerms[termIndex]

	// Make API request
	url := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=40&orderBy=relevance", searchTerm)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var booksResp GoogleBooksResponse
	if err := json.Unmarshal(body, &booksResp); err != nil {
		return nil, err
	}

	if len(booksResp.Items) == 0 {
		return nil, fmt.Errorf("no books found")
	}

	// Pick a book deterministically based on the date
	bookIndex := rng.Intn(len(booksResp.Items))
	item := booksResp.Items[bookIndex]

	selectedBook := &Book{
		Title:       item.VolumeInfo.Title,
		Authors:     item.VolumeInfo.Authors,
		Description: item.VolumeInfo.Description,
		Thumbnail:   item.VolumeInfo.ImageLinks.Thumbnail,
	}

	return selectedBook, nil
}

// generateCryptogramMessage creates a message referencing the book
func generateCryptogramMessage(bookTitle string, keywords []string, numbers []int, rng *rand.Rand) string {
	messages := []string{
		fmt.Sprintf("THE ANSWER LIES IN CHAPTER %d PAGE %d OF THIS BOOK", numbers[0], numbers[1]),
		fmt.Sprintf("LOOK FOR THE WORD %s IN THE STORY TO FIND THE TRUTH", strings.ToUpper(keywords[0])),
		fmt.Sprintf("PAGE %d HOLDS THE KEY TO UNDERSTANDING EVERYTHING", numbers[1]),
		fmt.Sprintf("CHAPTER %d REVEALS THE SECRET HIDDEN IN PLAIN SIGHT", numbers[0]),
		fmt.Sprintf("THE %s APPEARS %d TIMES AND SHOWS THE WAY FORWARD", strings.ToUpper(keywords[0]), numbers[0]),
	}

	return messages[rng.Intn(len(messages))]
}

// generateFallbackCryptogram creates a cryptogram when API fails
func generateFallbackCryptogram(date string, rng *rand.Rand) *DailyCryptogram {
	fallbackMessages := []string{
		"ERROR LOGS ARE THE STORIES WE TELL OURSELVES ABOUT WHAT WENT WRONG",
		"DEBUGGING IS LIKE BEING A DETECTIVE IN A CRIME MOVIE WHERE YOU ARE ALSO THE MURDERER",
		"CODE NEVER LIES COMMENTS SOMETIMES DO",
		"FIRST SOLVE THE PROBLEM THEN WRITE THE CODE",
		"THE BEST ERROR MESSAGE IS THE ONE THAT NEVER SHOWS UP",
	}

	plainText := fallbackMessages[rng.Intn(len(fallbackMessages))]
	substitutionMap := generateSubstitutionCipher(rng)
	cipherText := applyCipher(plainText, substitutionMap)

	return &DailyCryptogram{
		Date:            date,
		PlainText:       plainText,
		CipherText:      cipherText,
		BookTitle:       "Classic Programming Wisdom",
		BookAuthor:      "The Developers",
		BookDescription: "A collection of timeless programming quotes and wisdom.",
		HintKeywords:    []string{"error", "code", "debug"},
		HintNumbers:     []int{42, 137},
		SubstitutionMap: substitutionMap,
	}
}

// generateSubstitutionCipher creates a random letter substitution map
func generateSubstitutionCipher(rng *rand.Rand) map[rune]rune {
	alphabet := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	shuffled := make([]rune, len(alphabet))
	copy(shuffled, alphabet)

	// Fisher-Yates shuffle
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	// Create substitution map
	subMap := make(map[rune]rune)
	for i, letter := range alphabet {
		subMap[letter] = shuffled[i]
	}

	return subMap
}

// applyCipher applies the substitution cipher to the plaintext
func applyCipher(plainText string, subMap map[rune]rune) string {
	var result strings.Builder
	for _, ch := range strings.ToUpper(plainText) {
		if cipher, exists := subMap[ch]; exists {
			result.WriteRune(cipher)
		} else {
			result.WriteRune(ch) // Keep spaces, numbers, punctuation
		}
	}
	return result.String()
}

// extractKeywords extracts meaningful words from text
func extractKeywords(text string, count int) []string {
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'))
	})

	var keywords []string
	seen := make(map[string]bool)

	for _, word := range words {
		lower := strings.ToLower(word)
		if len(word) > 5 && !seen[lower] && !isCommonWord(lower) {
			keywords = append(keywords, word)
			seen[lower] = true
			if len(keywords) >= count {
				break
			}
		}
	}

	// Fallback if not enough keywords found
	if len(keywords) == 0 {
		keywords = []string{"mystery", "secret", "answer"}
	}

	return keywords
}

// isCommonWord filters out very common words
func isCommonWord(word string) bool {
	common := map[string]bool{
		"about": true, "after": true, "before": true, "where": true,
		"which": true, "their": true, "there": true, "these": true,
		"those": true, "through": true, "without": true, "would": true,
		"could": true, "should": true, "because": true,
	}
	return common[word]
}

// hashDateToSeed converts a date string to a deterministic seed
func hashDateToSeed(date string) int64 {
	hash := sha256.Sum256([]byte(date))
	seed := int64(0)
	for i := 0; i < 8; i++ {
		seed = (seed << 8) | int64(hash[i])
	}
	return seed
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
