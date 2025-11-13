/*
# Module: services/business.go
Business discovery and nearby search using Google Places API.

## Linked Modules
- [types/business](../types/business.go) - Business data structures
- [clients/google_places](../clients/google_places.go) - Google Places API client

## Tags
business-logic, search, geolocation

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
    ], [
        code:name "clients/google_places" ;
        code:path "../clients/google_places.go" ;
        code:relationship "Google Places API client"
    ] ;
    code:exports :BusinessService, :NewBusinessService, :FetchNearbyBusinesses, :GetBusinessType, :ExtractLocationKeywords ;
    code:tags "business-logic", "search", "geolocation" .
<!-- End LinkedDoc RDF -->
*/
package services

import (
	"strings"

	"location-tracker/clients"
	"location-tracker/types"
)

// BusinessService handles business search and discovery
type BusinessService struct {
	placesClient *clients.GooglePlacesClient
}

// NewBusinessService creates a new BusinessService instance
func NewBusinessService(googleMapsAPIKey string) *BusinessService {
	return &BusinessService{
		placesClient: clients.NewGooglePlacesClient(googleMapsAPIKey),
	}
}

// FetchNearbyBusinesses finds businesses near the given coordinates using Google Places API
func (s *BusinessService) FetchNearbyBusinesses(lat, lng float64) ([]types.Business, error) {
	return s.placesClient.SearchNearby(lat, lng, 500) // 500 meters radius
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
