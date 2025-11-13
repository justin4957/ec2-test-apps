package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/justin4957/ec2-test-apps/solid-poc/internal/solid"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	// Serve static files from proof-of-concept directory
	fs := http.FileServer(http.Dir("../proof-of-concept"))
	http.Handle("/", fs)

	// API endpoints for backend Solid operations
	http.HandleFunc("/api/health", healthHandler)
	http.HandleFunc("/api/validate-token", validateTokenHandler)
	http.HandleFunc("/api/pod/read", podReadHandler)
	http.HandleFunc("/api/pod/write", podWriteHandler)
	http.HandleFunc("/api/rdf/serialize", serializeHandler)
	http.HandleFunc("/api/rdf/deserialize", deserializeHandler)

	log.Printf("🔐 Solid PoC Server starting on port %s", port)
	log.Printf("📂 Serving proof-of-concept from ../proof-of-concept")
	log.Printf("🌐 Access at: http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "solid-poc",
		"version": "1.0.0",
	})
}

// validateTokenHandler validates a DPoP token from the frontend
func validateTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the DPoP token
	valid, err := solid.ValidateDPoPToken(req.Token)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": valid,
	})
}

// podReadHandler reads a resource from a Pod (server-side)
func podReadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token string `json:"token"`
		URL   string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	client, err := solid.NewClient(req.Token)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create client: %v", err), http.StatusBadRequest)
		return
	}

	data, contentType, err := client.GetResource(r.Context(), req.URL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read resource: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":         string(data),
		"content_type": contentType,
	})
}

// podWriteHandler writes a resource to a Pod (server-side)
func podWriteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token       string `json:"token"`
		URL         string `json:"url"`
		Data        string `json:"data"`
		ContentType string `json:"content_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	client, err := solid.NewClient(req.Token)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create client: %v", err), http.StatusBadRequest)
		return
	}

	if err := client.PutResource(r.Context(), req.URL, []byte(req.Data), req.ContentType); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write resource: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// serializeHandler converts Location data to RDF (Turtle or JSON-LD)
func serializeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Format string                 `json:"format"` // "turtle" or "jsonld"
		Data   map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var result string
	var err error

	switch req.Format {
	case "turtle":
		result, err = solid.SerializeToTurtle(req.Data)
	case "jsonld":
		result, err = solid.SerializeToJSONLD(req.Data)
	default:
		http.Error(w, "Invalid format. Use 'turtle' or 'jsonld'", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Serialization failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"format": req.Format,
		"result": result,
	})
}

// deserializeHandler converts RDF to Go data structures
func deserializeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Format string `json:"format"` // "turtle" or "jsonld"
		Data   string `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var result map[string]interface{}
	var err error

	switch req.Format {
	case "turtle":
		result, err = solid.DeserializeFromTurtle(req.Data)
	case "jsonld":
		result, err = solid.DeserializeFromJSONLD(req.Data)
	default:
		http.Error(w, "Invalid format. Use 'turtle' or 'jsonld'", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Deserialization failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
