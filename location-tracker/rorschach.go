package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"location-tracker/clients"
	"location-tracker/types"
)

// RorschachInterpretRequest represents the request for AI interpretation
type RorschachInterpretRequest struct {
	ErrorID string `json:"error_id"`
}

// RorschachUserResponseRequest represents the request for user response
type RorschachUserResponseRequest struct {
	ErrorID  string `json:"error_id"`
	Response string `json:"response"`
}

// handleRorschachInterpret generates an AI interpretation of the Rorschach image
func handleRorschachInterpret(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract error ID from URL path: /api/rorschach/interpret/{id}
	errorID := strings.TrimPrefix(r.URL.Path, "/api/rorschach/interpret/")
	if errorID == "" {
		http.Error(w, "Error ID is required", http.StatusBadRequest)
		return
	}

	log.Printf("🎨 Rorschach interpretation request for error ID: %s", errorID)

	// Find the error log
	errorLogMutex.Lock()
	var targetLog *types.ErrorLog
	var targetIndex int
	for i := range errorLogs {
		if errorLogs[i].ID == errorID {
			targetLog = &errorLogs[i]
			targetIndex = i
			break
		}
	}
	errorLogMutex.Unlock()

	if targetLog == nil {
		log.Printf("❌ Error log not found: %s", errorID)
		http.Error(w, "Error log not found", http.StatusNotFound)
		return
	}

	// Check if interpretation already exists
	if targetLog.RorschachAIResponse != "" {
		log.Printf("✅ Returning cached Rorschach interpretation")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"interpretation": targetLog.RorschachAIResponse,
		})
		return
	}

	// Check if Rorschach image is assigned
	if targetLog.RorschachImageNumber == 0 {
		log.Printf("⚠️  No Rorschach image assigned to this error log")
		http.Error(w, "No Rorschach image assigned", http.StatusBadRequest)
		return
	}

	// Generate AI interpretation
	interpretation, err := generateRorschachInterpretation(targetLog.RorschachImageNumber, targetLog.Message)
	if err != nil {
		log.Printf("❌ Failed to generate interpretation: %v", err)
		http.Error(w, fmt.Sprintf("Failed to generate interpretation: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Generated Rorschach interpretation (%d chars)", len(interpretation))

	// Update error log with interpretation
	errorLogMutex.Lock()
	errorLogs[targetIndex].RorschachAIResponse = interpretation
	errorLogMutex.Unlock()

	// Save to DynamoDB asynchronously
	if useDynamoDB {
		go func() {
			updatedLog := errorLogs[targetIndex]
			saveErrorLogToDynamoDB(updatedLog)
		}()
	}

	// Return interpretation
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"interpretation": interpretation,
	})
}

// handleRorschachUserResponse saves a user's response to the Rorschach image
func handleRorschachUserResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication
	if !isAuthenticated(r) {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Extract error ID from URL path: /api/rorschach/respond/{id}
	errorID := strings.TrimPrefix(r.URL.Path, "/api/rorschach/respond/")
	if errorID == "" {
		http.Error(w, "Error ID is required", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req RorschachUserResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Response == "" {
		http.Error(w, "Response cannot be empty", http.StatusBadRequest)
		return
	}

	// Limit response length
	if len(req.Response) > 1000 {
		http.Error(w, "Response too long (max 1000 characters)", http.StatusBadRequest)
		return
	}

	log.Printf("📝 User Rorschach response for error ID: %s (%d chars)", errorID, len(req.Response))

	// Find the error log
	errorLogMutex.Lock()
	var targetLog *types.ErrorLog
	var targetIndex int
	for i := range errorLogs {
		if errorLogs[i].ID == errorID {
			targetLog = &errorLogs[i]
			targetIndex = i
			break
		}
	}
	errorLogMutex.Unlock()

	if targetLog == nil {
		log.Printf("❌ Error log not found: %s", errorID)
		http.Error(w, "Error log not found", http.StatusNotFound)
		return
	}

	// Update error log with user response
	errorLogMutex.Lock()
	errorLogs[targetIndex].RorschachUserResponse = req.Response
	errorLogMutex.Unlock()

	// Save to DynamoDB asynchronously
	if useDynamoDB {
		go func() {
			updatedLog := errorLogs[targetIndex]
			saveErrorLogToDynamoDB(updatedLog)
		}()
	}

	log.Printf("✅ Saved user Rorschach response")

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// generateRorschachInterpretation calls OpenAI to generate a humorous Freudian interpretation
func generateRorschachInterpretation(imageNumber int, errorMessage string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	openAIClient := clients.NewOpenAIClient(apiKey)

	prompt := fmt.Sprintf(`You are a neurotic patient being shown Rorschach inkblot Card #%d during a psychological evaluation. You are acutely self-aware of your own neuroses and describe them with dark humor.

Provide your response in the standard Rorschach test format - describe what you literally see in the inkblot (animals, people, architecture, objects, landscapes, body parts, etc.) while revealing your neurosis through your interpretation.

Your response should be 3-5 sentences that:
- STARTS by describing specific imagery you see (e.g., "I see two butterflies", "It looks like a bat hanging upside down", "I see a cathedral with crumbling walls", "Two people pulling apart something", "A pelvis or maybe a mask")
- Connects the imagery to your psychological problems with acute self-awareness
- Reveals deep-seated guilt about disappointing family members or authority figures
- Subtly expresses comfort with being told what to do, seeking constant external validation, fear of making independent decisions, or relief when others take control
- Uses guilt, shame, and fear of judgment as underlying emotional drivers (influenced by religious upbringing without explicitly naming denominations)
- Sometimes includes plausible pop culture memes, TV show references (The Sopranos, Mad Men, Breaking Bad, The Office, etc.), or trending social media content
- Is darkly humorous and self-deprecating about your neurosis, but not mean-spirited
- Varies the neurotic themes: codependency, perfectionism, people-pleasing, fear of autonomy, compulsive rule-following, inability to trust own judgment

Write in first person as an overly self-aware patient speaking to your analyst, describing actual imagery in the inkblot while connecting it to your psychological problems.

Example tones:
- "I see two bears reaching toward each other but not quite touching. *laughs nervously* Kind of like how I approach every relationship - desperately wanting connection but terrified of being the one who reaches too far without explicit permission. My therapist says I have 'boundary issues' but honestly it's just easier when people tell me exactly what they need from me. Less room for error that way."
- "It's a butterfly, but the wings look... uneven? Like one side followed the instructions perfectly and the other side freelanced. That's basically my internal monologue during every work project. My whole childhood was 'what would disappoint everyone less' - which, ironically, prepared me perfectly for middle management where I can defer every decision upward."
- "I see a cathedral, or maybe a courthouse - something with rules and structure and people in robes telling you what to do. *sighs* The symmetry is honestly comforting. At least SOMEONE designed this with a rubric I can follow. Is it weird that I find the idea of judgment day kind of... relaxing? Like finally someone with authority will just TELL me if I did it right?"`, imageNumber)

	messages := []clients.OpenAIMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	return openAIClient.ChatCompletion("gpt-4", messages)
}
