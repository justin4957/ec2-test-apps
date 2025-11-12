package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"
)

const (
	fbShareWidth  = 1200
	fbShareHeight = 630
	maxImageWidth = 1200
	maxImageHeight = 400
)

// FacebookShareResponse represents the response for a Facebook share request
type FacebookShareResponse struct {
	ImageURL    string `json:"image_url"`
	ShareURL    string `json:"share_url"`
	Caption     string `json:"caption"`
	DirectShare string `json:"direct_share_url"`
}

// generateFacebookShareImage creates a compilation image from an error log
func generateFacebookShareImage(errorLog *ErrorLog) ([]byte, error) {
	log.Printf("🎨 Starting image generation for error: %s", errorLog.ID)

	// Create base canvas
	img := image.NewRGBA(image.Rect(0, 0, fbShareWidth, fbShareHeight))

	// Fill with gradient background
	drawGradientBackground(img)

	currentY := 20

	// Add title header
	currentY = drawTextBox(img, "🚨 ERROR COMPILATION 🚨", 20, currentY, 1160, 40, color.RGBA{255, 255, 255, 255}, true)
	currentY += 10

	// Add error message
	currentY = drawTextBox(img, errorLog.Message, 20, currentY, 1160, 60, color.RGBA{255, 220, 220, 255}, false)
	currentY += 10

	// Add slogan if available
	if errorLog.Slogan != "" {
		currentY = drawTextBox(img, fmt.Sprintf("💡 %s", errorLog.Slogan), 20, currentY, 1160, 40, color.RGBA{255, 255, 180, 255}, false)
		currentY += 10
	}

	log.Printf("📝 Text rendering complete, currentY=%d", currentY)

	// Calculate remaining space for images
	remainingHeight := fbShareHeight - currentY - 20
	imageAreaY := currentY

	// Collect images to display
	var imagesToComposite []image.Image

	// Priority 1: Meme image (if available)
	if errorLog.MemeURL != "" {
		log.Printf("📥 Downloading meme image: %s", errorLog.MemeURL)
		if memeImg, err := downloadAndDecodeImage(errorLog.MemeURL); err == nil {
			imagesToComposite = append(imagesToComposite, memeImg)
			log.Printf("✅ Meme image downloaded")
		} else {
			log.Printf("⚠️  Failed to download meme: %v", err)
		}
	}

	// Priority 2: Food image (if available and no meme)
	if len(imagesToComposite) == 0 && errorLog.FoodImageURL != "" {
		log.Printf("📥 Downloading food image: %s", errorLog.FoodImageURL)
		if foodImg, err := downloadAndDecodeImage(errorLog.FoodImageURL); err == nil {
			imagesToComposite = append(imagesToComposite, foodImg)
			log.Printf("✅ Food image downloaded")
		} else {
			log.Printf("⚠️  Failed to download food image: %v", err)
		}
	}

	// Priority 3: GIF thumbnails (up to 4)
	gifCount := 0
	for _, gifURL := range errorLog.GifURLs {
		if gifCount >= 4 {
			break
		}
		log.Printf("📥 Downloading GIF %d: %s", gifCount+1, gifURL)
		if gifImg, err := downloadAndDecodeImage(gifURL); err == nil {
			imagesToComposite = append(imagesToComposite, gifImg)
			gifCount++
			log.Printf("✅ GIF %d downloaded", gifCount)
		} else {
			log.Printf("⚠️  Failed to download GIF: %v", err)
		}
	}

	// Fallback to single GIF URL if GifURLs is empty
	if len(imagesToComposite) == 0 && errorLog.GifURL != "" {
		log.Printf("📥 Downloading fallback GIF: %s", errorLog.GifURL)
		if gifImg, err := downloadAndDecodeImage(errorLog.GifURL); err == nil {
			imagesToComposite = append(imagesToComposite, gifImg)
			log.Printf("✅ Fallback GIF downloaded")
		} else {
			log.Printf("⚠️  Failed to download fallback GIF: %v", err)
		}
	}

	log.Printf("🖼️  Total images collected: %d", len(imagesToComposite))

	// Composite images into the canvas
	if len(imagesToComposite) > 0 {
		log.Printf("🎨 Compositing %d images", len(imagesToComposite))
		compositeImages(img, imagesToComposite, 20, imageAreaY, fbShareWidth-40, remainingHeight-30)
	} else {
		// No images available, add placeholder text
		log.Printf("⚠️  No images available, adding placeholder")
		drawTextBox(img, "No visual media available for this error", 20, imageAreaY+50, 1160, 100, color.RGBA{180, 180, 180, 255}, false)
	}

	// Add footer with timestamp
	footerY := fbShareHeight - 25
	drawTextBox(img, fmt.Sprintf("Generated: %s | notspies.org", errorLog.Timestamp.Format("Jan 2, 2006")), 20, footerY, 1160, 20, color.RGBA{200, 200, 200, 255}, false)

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}

// drawGradientBackground fills the image with a gradient
func drawGradientBackground(img *image.RGBA) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		ratio := float64(y) / float64(bounds.Max.Y)
		r := uint8(20 + ratio*40)
		g := uint8(20 + ratio*60)
		b := uint8(40 + ratio*80)
		c := color.RGBA{r, g, b, 255}
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			img.Set(x, y, c)
		}
	}
}

// drawTextBox draws text within a box with word wrapping
func drawTextBox(img *image.RGBA, text string, x, y, maxWidth, maxHeight int, textColor color.RGBA, bold bool) int {
	// Parse font
	fontData := goregular.TTF
	parsedFont, err := freetype.ParseFont(fontData)
	if err != nil {
		log.Printf("Failed to parse font: %v", err)
		return y + 20
	}

	fontSize := 18.0
	if bold {
		fontSize = 24.0
	}

	// Create font face
	face := truetype.NewFace(parsedFont, &truetype.Options{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})

	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(textColor),
		Face: face,
	}

	// Word wrap
	words := strings.Fields(text)
	lines := []string{}
	currentLine := ""

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		advance := drawer.MeasureString(testLine)
		if advance.Ceil() > maxWidth-20 && currentLine != "" {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine = testLine
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	// Draw lines
	currentY := y + int(fontSize) + 5
	lineHeight := int(fontSize * 1.3)

	for i, line := range lines {
		if i*lineHeight > maxHeight {
			break
		}
		drawer.Dot = fixed.Point26_6{
			X: fixed.I(x + 10),
			Y: fixed.I(currentY),
		}
		drawer.DrawString(line)
		currentY += lineHeight
	}

	return currentY + 10
}

// downloadAndDecodeImage downloads an image from a URL and decodes it
func downloadAndDecodeImage(imageURL string) (image.Image, error) {
	// Add timeout to prevent hanging
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Try to decode as various formats
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		// Try JPEG specifically
		img, err = jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			// Try PNG specifically
			img, err = png.Decode(bytes.NewReader(data))
			if err != nil {
				return nil, fmt.Errorf("failed to decode image: %w", err)
			}
		}
	}

	return img, nil
}

// compositeImages arranges multiple images in a grid layout
func compositeImages(canvas *image.RGBA, images []image.Image, x, y, maxWidth, maxHeight int) {
	if len(images) == 0 {
		return
	}

	// Determine layout
	var cols, rows int
	switch len(images) {
	case 1:
		cols, rows = 1, 1
	case 2:
		cols, rows = 2, 1
	case 3, 4:
		cols, rows = 2, 2
	default:
		cols, rows = 3, 2
	}

	cellWidth := maxWidth / cols
	cellHeight := maxHeight / rows

	for i, img := range images {
		if i >= cols*rows {
			break
		}

		row := i / cols
		col := i % cols

		cellX := x + col*cellWidth
		cellY := y + row*cellHeight

		// Resize and draw image
		resized := resizeImage(img, cellWidth-10, cellHeight-10)

		// Center the image in the cell
		offsetX := cellX + (cellWidth-resized.Bounds().Dx())/2
		offsetY := cellY + (cellHeight-resized.Bounds().Dy())/2

		draw.Draw(canvas, image.Rect(offsetX, offsetY, offsetX+resized.Bounds().Dx(), offsetY+resized.Bounds().Dy()), resized, image.Point{0, 0}, draw.Over)
	}
}

// resizeImage resizes an image to fit within maxWidth x maxHeight while maintaining aspect ratio
func resizeImage(img image.Image, maxWidth, maxHeight int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate scaling factor
	scaleX := float64(maxWidth) / float64(width)
	scaleY := float64(maxHeight) / float64(height)
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}

	if scale >= 1.0 {
		return img // No need to resize
	}

	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)

	// Create new image
	resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Simple nearest-neighbor scaling
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := int(float64(x) / scale)
			srcY := int(float64(y) / scale)
			resized.Set(x, y, img.At(srcX, srcY))
		}
	}

	return resized
}

// handleFacebookShare generates a shareable compilation image and returns share URLs
func handleFacebookShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract error log ID from path: /api/facebook-share/{id}
	errorID := strings.TrimPrefix(r.URL.Path, "/api/facebook-share/")
	if errorID == "" {
		log.Printf("❌ Facebook share: No error ID provided")
		http.Error(w, "Error ID is required", http.StatusBadRequest)
		return
	}

	log.Printf("📱 Facebook share request: errorID=%s", errorID)

	// Find the error log by ID (same as regular error log endpoint)
	errorLogMutex.RLock()
	var targetLog *ErrorLog
	for i := range errorLogs {
		if errorLogs[i].ID == errorID {
			targetLog = &errorLogs[i]
			break
		}
	}
	errorLogMutex.RUnlock()

	if targetLog == nil {
		log.Printf("❌ Facebook share: Error log not found (ID: %s)", errorID)
		http.Error(w, "Error log not found", http.StatusNotFound)
		return
	}

	log.Printf("✅ Facebook share: Found error log: %s", targetLog.Message)

	// Generate compilation image
	imageData, err := generateFacebookShareImage(targetLog)
	if err != nil {
		log.Printf("❌ Failed to generate Facebook share image: %v", err)
		http.Error(w, fmt.Sprintf("Failed to generate share image: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Facebook share image generated (%d bytes)", len(imageData))

	// Store image temporarily (in production, upload to S3)
	shareImageID := fmt.Sprintf("share_%s", errorID)
	shareImagesMutex.Lock()
	shareImages[shareImageID] = imageData
	shareImagesMutex.Unlock()

	// Create share URLs
	baseURL := "https://notspies.org"
	if os.Getenv("BASE_URL") != "" {
		baseURL = os.Getenv("BASE_URL")
	}

	imageURL := fmt.Sprintf("%s/api/share-image/%s", baseURL, shareImageID)

	// Use the error log's existing URL (which has proper timestamp format)
	shareURL := targetLog.URL
	if shareURL == "" {
		// Fallback if URL not set
		shareURL = fmt.Sprintf("%s/", baseURL)
	}

	// Create caption
	caption := fmt.Sprintf("🚨 %s\n\n💡 %s\n\n🔗 View full error experience: %s",
		targetLog.Message,
		targetLog.Slogan,
		shareURL,
	)

	// Create Facebook direct share URL
	fbShareURL := fmt.Sprintf("https://www.facebook.com/sharer/sharer.php?u=%s&quote=%s",
		shareURL,
		strings.ReplaceAll(caption, "\n", "%0A"),
	)

	// Return response
	response := FacebookShareResponse{
		ImageURL:    imageURL,
		ShareURL:    shareURL,
		Caption:     caption,
		DirectShare: fbShareURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleShareImage serves the generated share image
func handleShareImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract share image ID from path: /api/share-image/{id}
	shareID := strings.TrimPrefix(r.URL.Path, "/api/share-image/")
	if shareID == "" {
		http.Error(w, "Invalid share ID", http.StatusBadRequest)
		return
	}

	// Retrieve image data
	shareImagesMutex.Lock()
	imageData, exists := shareImages[shareID]
	shareImagesMutex.Unlock()

	if !exists {
		http.Error(w, "Share image not found", http.StatusNotFound)
		return
	}

	// Serve image
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 1 day
	w.Write(imageData)
}

// In-memory storage for share images (in production, use S3)
var (
	shareImages      = make(map[string][]byte)
	shareImagesMutex sync.Mutex
)
