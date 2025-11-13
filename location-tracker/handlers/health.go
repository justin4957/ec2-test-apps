/*
# Module: handlers/health.go
Health check endpoint handler.

## Linked Modules
(None - simple health check with no dependencies)

## Tags
http, health, api

## Exports
HandleHealth

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "handlers/health.go" ;
    code:description "Health check endpoint handler" ;
    code:exports :HandleHealth ;
    code:tags "http", "health", "api" .
<!-- End LinkedDoc RDF -->
*/
package handlers

import (
	"encoding/json"
	"net/http"
)

// HandleHealth handles GET /api/health
// Returns a simple health status response
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}
