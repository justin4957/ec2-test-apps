/*
# Module: clients/google_places.go
Google Places API client for nearby business discovery.

## Linked Modules
- [types/business](../types/business.go) - Business data structures

## Tags
api-client, google, places, geolocation

## Exports
GooglePlacesClient, NewGooglePlacesClient, SearchNearby

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "clients/google_places.go" ;
    code:description "Google Places API client for nearby business discovery" ;
    code:linksTo [
        code:name "types/business" ;
        code:path "../types/business.go" ;
        code:relationship "Business data structures"
    ] ;
    code:exports :GooglePlacesClient, :NewGooglePlacesClient, :SearchNearby ;
    code:tags "api-client", "google", "places", "geolocation" .
<!-- End LinkedDoc RDF -->
*/
package clients

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"location-tracker/types"
)

// GooglePlacesClient handles Google Places API requests
type GooglePlacesClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewGooglePlacesClient creates a new Google Places API client
func NewGooglePlacesClient(apiKey string) *GooglePlacesClient {
	return &GooglePlacesClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// SearchNearby finds businesses near the given coordinates
func (c *GooglePlacesClient) SearchNearby(lat, lng float64, radiusMeters int) ([]types.Business, error) {
	if c.apiKey == "" {
		log.Println("⚠️  Google Maps API key not set, skipping business search")
		return []types.Business{}, nil
	}

	baseURL := "https://maps.googleapis.com/maps/api/place/nearbysearch/json"
	params := url.Values{}
	params.Add("location", fmt.Sprintf("%.6f,%.6f", lat, lng))
	params.Add("radius", fmt.Sprintf("%d", radiusMeters))
	params.Add("key", c.apiKey)

	fullURL := baseURL + "?" + params.Encode()

	resp, err := c.httpClient.Get(fullURL)
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

	businesses := make([]types.Business, 0, len(result.Results))
	for _, place := range result.Results {
		business := types.Business{
			Name:    place.Name,
			Type:    getFirstMeaningfulType(place.Types),
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

// getFirstMeaningfulType returns the first meaningful business type
func getFirstMeaningfulType(types []string) string {
	skipTypes := map[string]bool{
		"point_of_interest": true,
		"establishment":     true,
	}

	for _, t := range types {
		if !skipTypes[t] {
			return t
		}
	}

	if len(types) > 0 {
		return types[0]
	}
	return "business"
}
