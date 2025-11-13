/*
# Module: handlers/location.go
HTTP handlers for location tracking and retrieval endpoints.

## Linked Modules
- [types/location](../types/location.go): Location data structures and domain models
- [storage/dynamodb](../storage/dynamodb.go): DynamoDB persistence layer for locations
- [services/business_search](../services/business_search.go): Nearby business discovery service
- [middleware/auth](../middleware/auth.go): Authentication middleware for protected routes

## Tags
http, location, tracking, api

## Exports
HandleLocation, HandleLocationByID, HandleDeleteLocation, HandleUpdateLocation

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "handlers/location.go" ;
    code:description "HTTP handlers for location tracking and retrieval endpoints" ;
    code:dependsOn <../types/location.go>, <../storage/dynamodb.go>, <../middleware/auth.go> ;
    code:integratesWith <../services/business_search.go> ;
    code:exports :HandleLocation, :HandleLocationByID, :HandleDeleteLocation, :HandleUpdateLocation ;
    code:tags "http", "location", "api" .
<!-- End LinkedDoc RDF -->
*/
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/justin4957/ec2-test-apps/location-tracker/middleware"
	"github.com/justin4957/ec2-test-apps/location-tracker/services"
	"github.com/justin4957/ec2-test-apps/location-tracker/storage"
	"github.com/justin4957/ec2-test-apps/location-tracker/types"
)

// LocationHandler handles location-related HTTP requests
type LocationHandler struct {
	storage       *storage.DynamoDBStore
	businessSvc   *services.BusinessSearchService
}

// NewLocationHandler creates a new location handler
func NewLocationHandler(store *storage.DynamoDBStore, businessSvc *services.BusinessSearchService) *LocationHandler {
	return &LocationHandler{
		storage:     store,
		businessSvc: businessSvc,
	}
}

// HandleLocation handles POST /api/location for creating new location records
func (h *LocationHandler) HandleLocation(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	userID, err := middleware.GetUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var loc types.Location
	if err := json.NewDecoder(r.Body).Decode(&loc); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate location
	if loc.Latitude == 0 || loc.Longitude == 0 {
		http.Error(w, "Invalid coordinates", http.StatusBadRequest)
		return
	}

	// Set user ID
	loc.UserID = userID

	// Save to storage
	if err := h.storage.SaveLocation(&loc); err != nil {
		http.Error(w, "Failed to save location", http.StatusInternalServerError)
		return
	}

	// Search for nearby businesses (async)
	go h.businessSvc.SearchNearby(loc.Latitude, loc.Longitude)

	// Return created location
	w.Header().Set("Content-Type", "application/json")
	w.WriteStatus(http.StatusCreated)
	json.NewEncoder(w).Encode(loc)
}

// HandleLocationByID handles GET /api/location/:id for retrieving a location by ID
func (h *LocationHandler) HandleLocationByID(w http.ResponseWriter, r *http.Request) {
	// Extract location ID from URL
	locationID := r.URL.Query().Get("id")
	if locationID == "" {
		http.Error(w, "Location ID required", http.StatusBadRequest)
		return
	}

	// Authenticate user
	userID, err := middleware.GetUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch location from storage
	loc, err := h.storage.GetLocation(locationID, userID)
	if err != nil {
		http.Error(w, "Location not found", http.StatusNotFound)
		return
	}

	// Return location
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loc)
}

// HandleDeleteLocation handles DELETE /api/location/:id
func (h *LocationHandler) HandleDeleteLocation(w http.ResponseWriter, r *http.Request) {
	// Implementation here
}

// HandleUpdateLocation handles PUT /api/location/:id
func (h *LocationHandler) HandleUpdateLocation(w http.ResponseWriter, r *http.Request) {
	// Implementation here
}
