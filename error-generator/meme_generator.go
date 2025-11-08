package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"golang.org/x/oauth2/google"
)

// BrandBible contains the absurdist meme prompt formula
const brandBiblePrompt = `You are an expert at creating high-concept absurdist memes. Use "The Absurd Cynic Meme Brand Bible":

**BRAND VOICE:**
Cynical, over-educated, frustrated, and deeply granular. The tone suggests someone who's seen every meeting, read every memo, and understands both the bureaucracy and its futility at a molecular level.

**CHARACTER/MASCOT: The Weary Oracle**
A bearded wizard (or suited corporate figure) pointing at something mundane with gravitas and existential dread. Think Gandalf but he's analyzing office supplies, or a CEO giving a TED Talk about staplers.

**VISUAL STRUCTURE - SPLIT PANELS OR JUXTAPOSITION:**
Two contrasting images side-by-side or layered:
- LEFT/TOP: Something grand, historical, mythic, or classical (museum art, ancient monuments, heroic statues, dramatic landscapes)
- RIGHT/BOTTOM: Something pathetically mundane, hyperspecific, and contemporary (office supplies, Slack notifications, traffic cones, yellow socks)

**EXAMPLES OF JUXTAPOSITION:**
- Traffic cone in an art museum next to a Monet, labeled "Object #HC-471 (High-Visibility Safety Apparatus, DOT-Compliant)"
- Roman Centurion statue pointing at a manila folder labeled "Strategic Realignment Initiative, Q3 2024"
- Renaissance painting of philosophers debating next to a screenshot of a Zoom meeting titled "All-Hands: Synergizing Core Competencies"
- Majestic mountain vista with a Post-it note stuck on it reading "URGENT: TPS Report Due EOD"

**TEXT STRUCTURE - THREE PARTS:**

1. **THE THESIS (Top/Main Text):**
   - A run-on sentence that sounds like corporate bureaucracy meets academic analysis
   - MUST include: committee names, file references, percentages, timestamps, or model numbers
   - Tone: Weary but precise, like someone who's documented every step of their own despair
   - 40-80 words
   - Use semicolons, em dashes, parentheticals liberally
   - Blend management-speak with philosophical vocabulary

   EXAMPLES:
   "Per the Strategic Realignment Framework (SRF-2024-Q3, Subsection 7.2b), all ontological inquiries regarding the purpose of Object #HC-471 (High-Visibility Traffic Cone, ANSI Z535.1-compliant) must now route through the Interdepartmental Aesthetics Committee; preliminary findings suggest it has achieved more structural integrity than our Q2 roadmap—see Appendix D for existential implications."

   "The Global Sock Consortium (GSC) has determined, after 847 hours of deliberation and cross-functional alignment, that yellow argyle socks (SKU: YAS-2024-L, Left Foot Variant) possess demonstrable control over quarterly earnings forecasts; Leadership is cautiously optimistic that we can leverage this insight pending approval from the Hosiery Governance Board (Form 19-C required)."

2. **THE PUNCHLINE (Bottom Text):**
   - Short, dismissive, weary acceptance or cryptic finality
   - Often a bureaucratic sign-off or knowing acknowledgment of absurdity
   - 8-20 words

   EXAMPLES:
   "The museum acquired it anyway. Budget code: INEXPLICABLE-701."
   "His quarterly review cited 'insufficient gravitas regarding manila folders.'"
   "The socks remain undefeated. ROI: Unknowable."
   "We've scheduled a follow-up meeting to discuss the meeting. Attendance: Mandatory."
   "File this under 'Lessons We'll Ignore.' Reference: SISYPHUS-2024."

3. **THE CALLOUT (Arrow/Pointer to Specific Detail):**
   - Label pointing to some arbitrary, hyperspecific element in the image
   - Format: Technical designation with unnecessary precision
   - 5-15 words

   EXAMPLES:
   "Object #HC-471 (DOT-Compliant, 28" height, color: OSHA Orange)"
   "Form 27-B, Rev. 9 (Unread Since 2019)"
   "Yellow Argyle Sock, Left (SKU: YAS-2024-L)"
   "Memo Re: Paradigm Shifts, Page 47, Paragraph 12, Footnote 8"
   "Conference Room C Thermostat (Permanently Set to 'Existential Dread')"

Generate a meme based on these error context keywords: %s

CRITICAL REQUIREMENTS:
- Image MUST use split-panel or juxtaposition structure (grand/classical vs. mundane/contemporary)
- Include "The Weary Oracle" character when possible (wizard, suited figure, pointing gesture)
- Text MUST have all three parts: Thesis (dense run-on), Punchline (dismissive brevity), Callout (technical label)
- Thesis must include specific references: model numbers, file codes, committee names, percentages, timestamps
- Vary the historical/classical reference wildly: museums, monuments, Renaissance art, Roman architecture, wilderness photography, classical philosophy
- Vary the mundane subject with absurd specificity: office supplies, corporate forms, Slack screenshots, traffic cones, socks, thermostats
- Tone: Cynical, over-educated, weary, precise

Your response must be JSON with this exact structure:
{
  "image_prompt": "A detailed split-panel or juxtaposition image prompt: [LEFT/TOP: classical/grand element] contrasted with [RIGHT/BOTTOM: mundane/contemporary element]. Include The Weary Oracle character (bearded wizard or suited figure) pointing at the mundane element with gravitas.",
  "text_overlay": "THE THESIS: [40-80 word run-on sentence with committee names, file codes, percentages, timestamps mixing corporate-speak with philosophy]\n\nTHE PUNCHLINE: [8-20 word dismissive finality or bureaucratic sign-off]\n\nTHE CALLOUT: [5-15 word technical label] → [pointing to specific detail in image]"
}`

// GeminiRequest represents the request to Google Gemini API
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiResponse represents the response from Google Gemini API
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// MemePrompt contains the generated prompts for meme creation
type MemePrompt struct {
	ImagePrompt string `json:"image_prompt"`
	TextOverlay string `json:"text_overlay"`
}

// GeminiImageRequest for Imagen API
type GeminiImageRequest struct {
	Instances []ImageInstance `json:"instances"`
	Parameters ImageParameters `json:"parameters"`
}

type ImageInstance struct {
	Prompt string `json:"prompt"`
}

type ImageParameters struct {
	SampleCount int `json:"sampleCount"`
}

type GeminiImageResponse struct {
	Predictions []struct {
		BytesBase64Encoded string `json:"bytesBase64Encoded"`
		MimeType           string `json:"mimeType"`
	} `json:"predictions"`
}

// extractKeywords extracts meaningful keywords from error context
func extractKeywords(errorMessage, slogan, verboseDesc, story string) []string {
	keywordsMap := make(map[string]bool)

	// Combine all text
	allText := strings.ToLower(errorMessage + " " + slogan + " " + verboseDesc + " " + story)

	// Remove common words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "is": true, "was": true, "are": true, "were": true, "been": true,
		"be": true, "have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"this": true, "that": true, "these": true, "those": true, "i": true, "you": true,
		"he": true, "she": true, "it": true, "we": true, "they": true, "what": true,
		"which": true, "who": true, "when": true, "where": true, "why": true, "how": true,
	}

	// Extract words
	wordRegex := regexp.MustCompile(`\b[a-z]{3,}\b`)
	words := wordRegex.FindAllString(allText, -1)

	for _, word := range words {
		if !stopWords[word] {
			keywordsMap[word] = true
		}
	}

	// Convert to slice
	keywords := make([]string, 0, len(keywordsMap))
	for keyword := range keywordsMap {
		keywords = append(keywords, keyword)
		if len(keywords) >= 10 {
			break
		}
	}

	return keywords
}

// generateMemePromptWithGemini uses Gemini to create a brand bible-aligned meme concept
func generateMemePromptWithGemini(keywords []string) (*MemePrompt, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" || apiKey == "YOUR_GEMINI_API_KEY_HERE" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	keywordsStr := strings.Join(keywords, ", ")
	prompt := fmt.Sprintf(brandBiblePrompt, keywordsStr)

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash-exp:generateContent?key=%s", apiKey)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini API error (status %d): %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from Gemini")
	}

	responseText := geminiResp.Candidates[0].Content.Parts[0].Text

	// Extract JSON from response (handle markdown code blocks)
	jsonRegex := regexp.MustCompile("(?s)```json\\s*(.+?)\\s*```|({.+})")
	matches := jsonRegex.FindStringSubmatch(responseText)
	var jsonStr string
	if len(matches) > 1 {
		if matches[1] != "" {
			jsonStr = matches[1]
		} else if matches[2] != "" {
			jsonStr = matches[2]
		}
	} else {
		jsonStr = responseText
	}

	var memePrompt MemePrompt
	if err := json.Unmarshal([]byte(jsonStr), &memePrompt); err != nil {
		return nil, fmt.Errorf("failed to parse meme prompt JSON: %w (response: %s)", err, responseText)
	}

	return &memePrompt, nil
}

// getVertexAIAccessToken gets an OAuth2 access token from the service account
func getVertexAIAccessToken(ctx context.Context) (string, error) {
	// Read service account JSON from file
	credsJSON, err := os.ReadFile("gcp-service-account.json")
	if err != nil {
		return "", fmt.Errorf("failed to read service account file: %w", err)
	}

	// Create credentials from JSON
	creds, err := google.CredentialsFromJSON(ctx, credsJSON, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("failed to create credentials: %w", err)
	}

	// Get token
	token, err := creds.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	return token.AccessToken, nil
}

// generateMemeImageWithGemini generates an image using Vertex AI Imagen API
func generateMemeImageWithGemini(imagePrompt string) ([]byte, error) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GCP_LOCATION")

	if projectID == "" {
		projectID = "notspies"
	}
	if location == "" {
		location = "us-central1"
	}

	// Get OAuth2 access token
	ctx := context.Background()
	accessToken, err := getVertexAIAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Build Vertex AI Imagen endpoint
	// Using imagen-3.0-generate-001 model
	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/imagen-3.0-generate-001:predict",
		location, projectID, location)

	// Create request body
	requestBody := map[string]interface{}{
		"instances": []map[string]interface{}{
			{
				"prompt": imagePrompt,
			},
		},
		"parameters": map[string]interface{}{
			"sampleCount": 1,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers with OAuth2 bearer token
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Vertex AI Imagen API: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("Vertex AI Imagen API failed (status %d): %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("vertex AI imagen API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Predictions []struct {
			BytesBase64Encoded string `json:"bytesBase64Encoded"`
			MimeType           string `json:"mimeType"`
		} `json:"predictions"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Failed to decode Vertex AI response: %s", string(body))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Predictions) == 0 || result.Predictions[0].BytesBase64Encoded == "" {
		log.Printf("No image in Vertex AI response: %s", string(body))
		return nil, fmt.Errorf("no image generated by Vertex AI Imagen")
	}

	// Decode base64 image data
	imageData, err := base64.StdEncoding.DecodeString(result.Predictions[0].BytesBase64Encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image data: %w", err)
	}

	log.Printf("✅ Successfully generated image with Vertex AI (%d bytes, type: %s)", len(imageData), result.Predictions[0].MimeType)
	return imageData, nil
}

// addTextToImage overlays text on an image with word wrapping and styling
func addTextToImage(imageData []byte, textOverlay string) ([]byte, error) {
	// Decode the image
	img, err := png.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Create a new RGBA image to draw on
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	// Parse the text sections (THESIS, PUNCHLINE, CALLOUT)
	sections := parseTextSections(textOverlay)

	// Draw text sections
	drawTextSections(rgba, sections)

	// Encode back to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, rgba); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}

// TextSection represents a section of meme text
type TextSection struct {
	Label string
	Text  string
	Y     int // Y position to draw at
}

// parseTextSections extracts THESIS, PUNCHLINE, and CALLOUT from the text overlay
func parseTextSections(textOverlay string) []TextSection {
	sections := []TextSection{}

	// Extract THE THESIS
	thesisRegex := regexp.MustCompile(`(?s)THE THESIS:\s*(.+?)(?:\n\nTHE PUNCHLINE:|$)`)
	if matches := thesisRegex.FindStringSubmatch(textOverlay); len(matches) > 1 {
		sections = append(sections, TextSection{
			Label: "THE THESIS",
			Text:  strings.TrimSpace(matches[1]),
			Y:     20, // Top of image
		})
	}

	// Extract THE PUNCHLINE
	punchlineRegex := regexp.MustCompile(`(?s)THE PUNCHLINE:\s*(.+?)(?:\n\nTHE CALLOUT:|$)`)
	if matches := punchlineRegex.FindStringSubmatch(textOverlay); len(matches) > 1 {
		sections = append(sections, TextSection{
			Label: "THE PUNCHLINE",
			Text:  strings.TrimSpace(matches[1]),
			Y:     -80, // Negative means from bottom
		})
	}

	// Extract THE CALLOUT
	calloutRegex := regexp.MustCompile(`(?s)THE CALLOUT:\s*(.+?)(?:\n|$)`)
	if matches := calloutRegex.FindStringSubmatch(textOverlay); len(matches) > 1 {
		sections = append(sections, TextSection{
			Label: "THE CALLOUT",
			Text:  strings.TrimSpace(matches[1]),
			Y:     -30, // Near bottom
		})
	}

	return sections
}

// drawTextSections draws each text section on the image
func drawTextSections(img *image.RGBA, sections []TextSection) {
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	for _, section := range sections {
		// Calculate Y position (negative means from bottom)
		yPos := section.Y
		if yPos < 0 {
			yPos = imgHeight + yPos
		}

		// Wrap text to fit image width
		lines := wrapText(section.Text, imgWidth-40, basicfont.Face7x13)

		// Draw background box for text
		boxHeight := len(lines)*15 + 20
		drawTextBox(img, 10, yPos-10, imgWidth-20, boxHeight)

		// Draw each line
		currentY := yPos
		for _, line := range lines {
			drawString(img, 20, currentY, line, color.White)
			currentY += 15
		}
	}
}

// wrapText wraps text to fit within a given width
func wrapText(text string, maxWidth int, face font.Face) []string {
	words := strings.Fields(text)
	lines := []string{}
	currentLine := ""

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		// Measure the width (approximate with basic font)
		width := len(testLine) * 7 // basicfont.Face7x13 is ~7 pixels wide per char
		if width > maxWidth && currentLine != "" {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine = testLine
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	// Limit to reasonable number of lines
	if len(lines) > 10 {
		lines = lines[:10]
		lines[9] = lines[9] + "..."
	}

	return lines
}

// drawTextBox draws a semi-transparent background box for text
func drawTextBox(img *image.RGBA, x, y, width, height int) {
	bounds := img.Bounds()

	// Clamp coordinates to image bounds
	x = int(math.Max(0, float64(x)))
	y = int(math.Max(0, float64(y)))
	x2 := int(math.Min(float64(bounds.Dx()), float64(x+width)))
	y2 := int(math.Min(float64(bounds.Dy()), float64(y+height)))

	// Draw semi-transparent black box
	boxColor := color.RGBA{0, 0, 0, 200}
	for py := y; py < y2; py++ {
		for px := x; px < x2; px++ {
			img.Set(px, py, boxColor)
		}
	}
}

// drawString draws a string on an image at the given position
func drawString(img *image.RGBA, x, y int, text string, col color.Color) {
	point := fixed.Point26_6{
		X: fixed.I(x),
		Y: fixed.I(y),
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}

	d.DrawString(text)
}

// uploadMemeToS3 uploads the generated meme image to S3
func uploadMemeToS3(imageData []byte, mimeType string) (string, error) {
	bucket := os.Getenv("S3_MEME_BUCKET")
	if bucket == "" {
		bucket = "error-generator-memes"
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	// Generate unique filename
	filename := fmt.Sprintf("memes/%s-%d.png", uuid.New().String(), time.Now().Unix())

	// Upload to S3
	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(filename),
		Body:        bytes.NewReader(imageData),
		ContentType: aws.String(mimeType),
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Return public URL
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, filename)
	return url, nil
}

// GenerateMemeForError orchestrates the full meme generation pipeline
func GenerateMemeForError(errorMessage, slogan, verboseDesc, story string) (string, error) {
	// Step 1: Extract keywords from error context
	keywords := extractKeywords(errorMessage, slogan, verboseDesc, story)
	log.Printf("Extracted keywords for meme: %v", keywords)

	if len(keywords) == 0 {
		keywords = []string{"error", "chaos", "debugging", "production"}
	}

	// Step 2: Generate meme concept using brand bible
	memePrompt, err := generateMemePromptWithGemini(keywords)
	if err != nil {
		log.Printf("Failed to generate meme prompt: %v", err)
		return "", err
	}

	log.Printf("Generated meme concept - Image: %s, Text: %s", memePrompt.ImagePrompt, memePrompt.TextOverlay)

	// Step 3: Generate the base image (without text)
	imageData, err := generateMemeImageWithGemini(memePrompt.ImagePrompt)
	if err != nil {
		log.Printf("Failed to generate meme image: %v", err)
		return "", err
	}

	// Step 4: Add text overlay to the image
	imageWithText, err := addTextToImage(imageData, memePrompt.TextOverlay)
	if err != nil {
		log.Printf("Failed to add text overlay: %v", err)
		return "", err
	}

	// Step 5: Upload to S3
	memeURL, err := uploadMemeToS3(imageWithText, "image/png")
	if err != nil {
		log.Printf("Failed to upload meme to S3: %v", err)
		return "", err
	}

	log.Printf("🎨 Successfully generated and uploaded meme: %s", memeURL)
	return memeURL, nil
}
