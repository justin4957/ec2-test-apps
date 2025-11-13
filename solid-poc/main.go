package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/justin4957/ec2-test-apps/solid-poc/internal/solid"
)

// loggingMiddleware wraps an http.Handler and logs all requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request
		log.Printf("→ %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Log headers (excluding sensitive tokens)
		if log.Writer() != nil {
			for key, values := range r.Header {
				if key == "Authorization" || key == "Cookie" {
					log.Printf("  %s: [REDACTED]", key)
				} else {
					log.Printf("  %s: %v", key, values)
				}
			}
		}

		// Create a response wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Log response
		duration := time.Since(start)
		log.Printf("← %s %s - %d (%v)", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	// Serve static files from proof-of-concept directory with logging
	fs := http.FileServer(http.Dir("../proof-of-concept"))
	http.Handle("/", loggingMiddleware(fs))

	// API endpoints for backend Solid operations
	http.HandleFunc("/api/health", loggingHandler(healthHandler))
	http.HandleFunc("/api/validate-token", loggingHandler(validateTokenHandler))
	http.HandleFunc("/api/pod/read", loggingHandler(podReadHandler))
	http.HandleFunc("/api/pod/write", loggingHandler(podWriteHandler))
	http.HandleFunc("/api/rdf/serialize", loggingHandler(serializeHandler))
	http.HandleFunc("/api/rdf/deserialize", loggingHandler(deserializeHandler))

	log.Printf("🔐 Solid PoC Server starting on port %s", port)
	log.Printf("📂 Serving proof-of-concept from ../proof-of-concept")
	log.Printf("🌐 Access at: http://localhost:%s", port)
	log.Printf("📝 Logging enabled for all requests")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// loggingHandler wraps an http.HandlerFunc and logs requests/responses
func loggingHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request
		log.Printf("→ %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Log headers (excluding sensitive tokens)
		for key, values := range r.Header {
			if key == "Authorization" || key == "Cookie" {
				log.Printf("  %s: [REDACTED]", key)
			} else {
				log.Printf("  %s: %v", key, values)
			}
		}

		// Create a response wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler
		next(wrapped, r)

		// Log response
		duration := time.Since(start)
		log.Printf("← %s %s - %d (%v)", r.Method, r.URL.Path, wrapped.statusCode, duration)
	}
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
		log.Printf("❌ Token validation failed: method not allowed (%s)", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Token validation failed: invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("🔑 Validating DPoP token")

	// Validate the DPoP token
	valid, err := solid.ValidateDPoPToken(req.Token)
	if err != nil {
		log.Printf("❌ Token validation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	log.Printf("✅ Token validation successful")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": valid,
	})
}

// podReadHandler reads a resource from a Pod (server-side)
func podReadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("❌ Pod read failed: method not allowed (%s)", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token string `json:"token"`
		URL   string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Pod read failed: invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("📖 Reading Pod resource: %s", req.URL)

	client, err := solid.NewClient(req.Token)
	if err != nil {
		log.Printf("❌ Pod read failed: client creation error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create client: %v", err), http.StatusBadRequest)
		return
	}

	data, contentType, err := client.GetResource(r.Context(), req.URL)
	if err != nil {
		log.Printf("❌ Pod read failed: %v", err)
		http.Error(w, fmt.Sprintf("Failed to read resource: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Pod read successful: %d bytes, content-type: %s", len(data), contentType)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":         string(data),
		"content_type": contentType,
	})
}

// podWriteHandler writes a resource to a Pod (server-side)
func podWriteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("❌ Pod write failed: method not allowed (%s)", r.Method)
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
		log.Printf("❌ Pod write failed: invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("📝 Writing Pod resource: %s (content-type: %s, size: %d bytes)", req.URL, req.ContentType, len(req.Data))

	client, err := solid.NewClient(req.Token)
	if err != nil {
		log.Printf("❌ Pod write failed: client creation error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create client: %v", err), http.StatusBadRequest)
		return
	}

	if err := client.PutResource(r.Context(), req.URL, []byte(req.Data), req.ContentType); err != nil {
		log.Printf("❌ Pod write failed: %v", err)
		http.Error(w, fmt.Sprintf("Failed to write resource: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Pod write successful: %s", req.URL)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// serializeHandler converts Location data to RDF (Turtle or JSON-LD)
func serializeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("❌ RDF serialize failed: method not allowed (%s)", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Format string                 `json:"format"` // "turtle" or "jsonld"
		Data   map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ RDF serialize failed: invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("🔄 Serializing to %s format", req.Format)

	var result string
	var err error

	switch req.Format {
	case "turtle":
		result, err = solid.SerializeToTurtle(req.Data)
	case "jsonld":
		result, err = solid.SerializeToJSONLD(req.Data)
	default:
		log.Printf("❌ RDF serialize failed: invalid format '%s'", req.Format)
		http.Error(w, "Invalid format. Use 'turtle' or 'jsonld'", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("❌ RDF serialize failed: %v", err)
		http.Error(w, fmt.Sprintf("Serialization failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ RDF serialization successful: %s (%d bytes)", req.Format, len(result))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"format": req.Format,
		"result": result,
	})
}

// deserializeHandler converts RDF to Go data structures
func deserializeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("❌ RDF deserialize failed: method not allowed (%s)", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Format string `json:"format"` // "turtle" or "jsonld"
		Data   string `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ RDF deserialize failed: invalid request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("🔄 Deserializing from %s format (%d bytes)", req.Format, len(req.Data))

	var result map[string]interface{}
	var err error

	switch req.Format {
	case "turtle":
		result, err = solid.DeserializeFromTurtle(req.Data)
	case "jsonld":
		result, err = solid.DeserializeFromJSONLD(req.Data)
	default:
		log.Printf("❌ RDF deserialize failed: invalid format '%s'", req.Format)
		http.Error(w, "Invalid format. Use 'turtle' or 'jsonld'", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("❌ RDF deserialize failed: %v", err)
		http.Error(w, fmt.Sprintf("Deserialization failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ RDF deserialization successful: extracted %d fields", len(result))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
