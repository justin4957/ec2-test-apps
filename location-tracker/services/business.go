/*
# Module: services/business.go
Business discovery and nearby search using Google Places API.

## Linked Modules
- [types/business](../types/business.go) - Business data structures

## Tags
business-logic, search, geolocation, api-client

## Exports
BusinessService, NewBusinessService, FetchNearbyBusinesses, GetBusinessType, ExtractLocationKeywords

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "services/business.go" ;
    code:description "Business discovery and nearby search using Google Places API" ;
    code:linksTo [
        code:name "types/business" ;
        code:path "../types/business.go" ;
        code:relationship "Business data structures"
    ] ;
    code:exports :BusinessService, :NewBusinessService, :FetchNearbyBusinesses, :GetBusinessType, :ExtractLocationKeywords ;
    code:tags "business-logic", "search", "geolocation", "api-client" .
<!-- End LinkedDoc RDF -->
*/
package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"location-tracker/types"
)

// BusinessService handles business search and discovery
type BusinessService struct {
	googleMapsAPIKey string
	client           *http.Client
}

// NewBusinessService creates a new BusinessService instance
func NewBusinessService(googleMapsAPIKey string) *BusinessService {
	return &BusinessService{
		googleMapsAPIKey: googleMapsAPIKey,
		client:           &http.Client{Timeout: 10 * time.Second},
	}
}

// FetchNearbyBusinesses finds businesses near the given coordinates using Google Places API
func (s *BusinessService) FetchNearbyBusinesses(lat, lng float64) ([]types.Business, error) {
	if s.googleMapsAPIKey == "" {
		log.Println("⚠️  Google Maps API key not set, skipping business search")
		return []types.Business{}, nil
	}

	baseURL := "https://maps.googleapis.com/maps/api/place/nearbysearch/json"
	params := url.Values{}
	params.Add("location", fmt.Sprintf("%.6f,%.6f", lat, lng))
	params.Add("radius", "500") // 500 meters
	params.Add("key", s.googleMapsAPIKey)

	fullURL := baseURL + "?" + params.Encode()

	resp, err := s.client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to call Google Places API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google Places API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result types.GooglePlacesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse Google Places response: %w", err)
	}

	businesses := []types.Business{}
	for _, place := range result.Results {
		business := types.Business{
			Name:    place.Name,
			Type:    s.GetBusinessType(place.Types),
			Address: place.FormattedAddress,
			PlaceID: place.PlaceID,
		}
		business.Location.Lat = place.Geometry.Location.Lat
		business.Location.Lng = place.Geometry.Location.Lng
		businesses = append(businesses, business)
	}

	log.Printf("🏪 Found %d businesses near location (%.6f, %.6f)", len(businesses), lat, lng)
	return businesses, nil
}

// GetBusinessType returns a human-readable business type from Google Places types
func (s *BusinessService) GetBusinessType(businessTypes []string) string {
	if len(businessTypes) == 0 {
		return "business"
	}

	// Priority order for display
	priorityTypes := []string{
		"restaurant", "cafe", "bar", "food",
		"store", "shop", "shopping_mall",
		"park", "museum", "library",
		"school", "hospital", "pharmacy",
		"bank", "atm", "post_office",
		"gas_station", "parking",
	}

	for _, priority := range priorityTypes {
		for _, bType := range businessTypes {
			if strings.Contains(bType, priority) {
				return priority
			}
		}
	}

	// Return first type if no priority match
	return businessTypes[0]
}

// ExtractLocationKeywords extracts keywords from location name and businesses for context
func (s *BusinessService) ExtractLocationKeywords(locationName string, businesses []types.Business) []string {
	keywords := []string{}

	// Extract words from location name
	locationWords := strings.Fields(strings.ToLower(locationName))
	keywords = append(keywords, locationWords...)

	// Extract business types (max 5 for brevity)
	typesSeen := make(map[string]bool)
	for _, business := range businesses {
		if business.Type != "" && !typesSeen[business.Type] && len(keywords) < 10 {
			keywords = append(keywords, business.Type)
			typesSeen[business.Type] = true
		}
	}

	return keywords
}
