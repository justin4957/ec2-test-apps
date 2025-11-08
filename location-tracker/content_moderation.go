package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// ModerationResult contains the result of content moderation
type ModerationResult struct {
	Status        string   // "approved", "rejected", "redacted"
	ModeratedText string   // Redacted version if needed
	Reason        string   // Explanation for rejection/redaction
	Categories    []string // Flagged categories
}

// ContentModerator handles content moderation using OpenAI APIs
type ContentModerator struct {
	openaiAPIKey string
	client       *http.Client
}

// NewContentModerator creates a new content moderator
func NewContentModerator(apiKey string) *ContentModerator {
	return &ContentModerator{
		openaiAPIKey: apiKey,
		client:       &http.Client{},
	}
}

// ModerateTip performs content moderation on a tip
func (cm *ContentModerator) ModerateTip(tipContent string) (*ModerationResult, error) {
	// Step 1: Basic validation
	if strings.TrimSpace(tipContent) == "" {
		return &ModerationResult{
			Status: "rejected",
			Reason: "Empty content",
		}, nil
	}

	// Step 2: OpenAI Moderation API (check for ToS violations)
	if cm.openaiAPIKey != "" {
		moderationResp, err := cm.callModerationAPI(tipContent)
		if err != nil {
			// If moderation API fails, fall back to pattern matching
			fmt.Printf("⚠️  OpenAI Moderation API failed: %v\n", err)
		} else if moderationResp.Flagged {
			return &ModerationResult{
				Status:     "rejected",
				Reason:     fmt.Sprintf("Content flagged for: %s", strings.Join(moderationResp.Categories, ", ")),
				Categories: moderationResp.Categories,
			}, nil
		}
	}

	// Step 3: Pattern-based PII redaction
	redactedText, wasRedacted := cm.redactSensitiveContent(tipContent)

	if wasRedacted {
		return &ModerationResult{
			Status:        "redacted",
			ModeratedText: redactedText,
			Reason:        "Sensitive information redacted",
		}, nil
	}

	return &ModerationResult{
		Status:        "approved",
		ModeratedText: tipContent,
	}, nil
}

// OpenAI Moderation API types
type openAIModerationRequest struct {
	Input string `json:"input"`
}

type openAIModerationResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Results []struct {
		Flagged    bool `json:"flagged"`
		Categories struct {
			Hate            bool `json:"hate"`
			HateThreatening bool `json:"hate/threatening"`
			SelfHarm        bool `json:"self-harm"`
			Sexual          bool `json:"sexual"`
			SexualMinors    bool `json:"sexual/minors"`
			Violence        bool `json:"violence"`
			ViolenceGraphic bool `json:"violence/graphic"`
		} `json:"categories"`
	} `json:"results"`
}

type moderationResult struct {
	Flagged    bool
	Categories []string
}

// callModerationAPI calls OpenAI Moderation API
func (cm *ContentModerator) callModerationAPI(content string) (*moderationResult, error) {
	reqBody := openAIModerationRequest{Input: content}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/moderations", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cm.openaiAPIKey)

	resp, err := cm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("moderation API returned status %d", resp.StatusCode)
	}

	var moderationResp openAIModerationResponse
	if err := json.NewDecoder(resp.Body).Decode(&moderationResp); err != nil {
		return nil, err
	}

	if len(moderationResp.Results) == 0 {
		return &moderationResult{Flagged: false}, nil
	}

	result := moderationResp.Results[0]
	categories := []string{}

	if result.Categories.Hate {
		categories = append(categories, "hate")
	}
	if result.Categories.HateThreatening {
		categories = append(categories, "hate/threatening")
	}
	if result.Categories.SelfHarm {
		categories = append(categories, "self-harm")
	}
	if result.Categories.Sexual {
		categories = append(categories, "sexual")
	}
	if result.Categories.SexualMinors {
		categories = append(categories, "sexual/minors")
	}
	if result.Categories.Violence {
		categories = append(categories, "violence")
	}
	if result.Categories.ViolenceGraphic {
		categories = append(categories, "violence/graphic")
	}

	return &moderationResult{
		Flagged:    result.Flagged,
		Categories: categories,
	}, nil
}

// redactSensitiveContent uses pattern matching to redact PII
func (cm *ContentModerator) redactSensitiveContent(content string) (string, bool) {
	original := content
	wasRedacted := false

	// Email pattern
	emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	if emailPattern.MatchString(content) {
		content = emailPattern.ReplaceAllString(content, "[EMAIL_REDACTED]")
		wasRedacted = true
	}

	// Phone number patterns (US and international)
	phonePatterns := []*regexp.Regexp{
		regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),                    // 123-456-7890
		regexp.MustCompile(`\(\d{3}\)\s*\d{3}[-.]?\d{4}`),                      // (123) 456-7890
		regexp.MustCompile(`\+\d{1,3}\s*\d{1,14}`),                             // +1 234567890
		regexp.MustCompile(`\b\d{3}\s\d{3}\s\d{4}\b`),                          // 123 456 7890
	}
	for _, pattern := range phonePatterns {
		if pattern.MatchString(content) {
			content = pattern.ReplaceAllString(content, "[PHONE_REDACTED]")
			wasRedacted = true
		}
	}

	// SSN pattern
	ssnPattern := regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	if ssnPattern.MatchString(content) {
		content = ssnPattern.ReplaceAllString(content, "[SSN_REDACTED]")
		wasRedacted = true
	}

	// Credit card pattern (basic check for 13-16 digit sequences)
	ccPattern := regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{3,4}\b`)
	if ccPattern.MatchString(content) {
		content = ccPattern.ReplaceAllString(content, "[CARD_REDACTED]")
		wasRedacted = true
	}

	// Street address pattern (basic)
	addressPattern := regexp.MustCompile(`\b\d+\s+[A-Z][a-z]+\s+(Street|St|Avenue|Ave|Road|Rd|Boulevard|Blvd|Lane|Ln|Drive|Dr|Court|Ct|Way)\b`)
	if addressPattern.MatchString(content) {
		content = addressPattern.ReplaceAllString(content, "[ADDRESS_REDACTED]")
		wasRedacted = true
	}

	// IP address pattern
	ipPattern := regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	if ipPattern.MatchString(content) {
		content = ipPattern.ReplaceAllString(content, "[IP_REDACTED]")
		wasRedacted = true
	}

	// URLs with specific protocols
	urlPattern := regexp.MustCompile(`https?://[^\s]+`)
	if urlPattern.MatchString(content) {
		content = urlPattern.ReplaceAllString(content, "[URL_REDACTED]")
		wasRedacted = true
	}

	return content, content != original || wasRedacted
}

// ValidateTipContent performs basic validation on tip content
func ValidateTipContent(content string, maxLength int) error {
	trimmed := strings.TrimSpace(content)

	if trimmed == "" {
		return fmt.Errorf("tip cannot be empty")
	}

	if len(content) > maxLength {
		return fmt.Errorf("tip exceeds maximum length of %d characters", maxLength)
	}

	// Check for suspicious patterns (spam indicators)
	if strings.Count(content, "http") > 3 {
		return fmt.Errorf("too many URLs in content")
	}

	// Check for excessive repetition
	words := strings.Fields(content)
	if len(words) > 5 {
		wordCount := make(map[string]int)
		for _, word := range words {
			wordCount[strings.ToLower(word)]++
		}
		for _, count := range wordCount {
			if count > len(words)/2 {
				return fmt.Errorf("content contains excessive repetition")
			}
		}
	}

	return nil
}
