package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// GiphyActionRequest represents the request to track a Giphy view
type GiphyActionRequest struct {
	GifID      string `json:"gif_id"`
	ActionType string `json:"action_type"`
	RandomID   string `json:"random_id"`
}

// handleGiphyAction proxies Giphy analytics calls through the backend to avoid CORS
func handleGiphyAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract GIF ID from URL path: /api/giphy/action/{id}
	gifID := strings.TrimPrefix(r.URL.Path, "/api/giphy/action/")
	if gifID == "" {
		http.Error(w, "GIF ID is required", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req GiphyActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Make the request to Giphy API
	giphyURL := fmt.Sprintf("https://api.giphy.com/v1/gifs/%s/actions", gifID)

	payload := map[string]string{
		"action_type": req.ActionType,
		"random_id":   req.RandomID,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal Giphy action request: %v", err)
		// Don't fail the request - analytics shouldn't break the UI
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ignored"})
		return
	}

	giphyReq, err := http.NewRequest("POST", giphyURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to create Giphy request: %v", err)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ignored"})
		return
	}

	giphyReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(giphyReq)
	if err != nil {
		log.Printf("Failed to call Giphy API: %v", err)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ignored"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("Giphy API returned non-OK status %d: %s", resp.StatusCode, string(body))
		// Still return OK - we don't want analytics failures to break the UI
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ignored"})
		return
	}

	log.Printf("✅ Giphy action tracked: GIF %s", gifID)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
