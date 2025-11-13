/*
# Module: services/commercial.go
Commercial real estate search and governance discovery using Perplexity API.

## Linked Modules
- [types/commercial](../types/commercial.go) - Commercial real estate data structures
- [types/api_types](../types/api_types.go) - Perplexity API types
- [clients/perplexity](../clients/perplexity.go) - Perplexity API client

## Tags
business-logic, commercial, real-estate, governance

## Exports
CommercialService, NewCommercialService, SearchCommercialRealEstate

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "services/commercial.go" ;
    code:description "Commercial real estate search and governance discovery using Perplexity API" ;
    code:linksTo [
        code:name "types/commercial" ;
        code:path "../types/commercial.go" ;
        code:relationship "Commercial real estate data structures"
    ], [
        code:name "types/api_types" ;
        code:path "../types/api_types.go" ;
        code:relationship "Perplexity API types"
    ], [
        code:name "clients/perplexity" ;
        code:path "../clients/perplexity.go" ;
        code:relationship "Perplexity API client"
    ] ;
    code:exports :CommercialService, :NewCommercialService, :SearchCommercialRealEstate ;
    code:tags "business-logic", "commercial", "real-estate", "governance" .
<!-- End LinkedDoc RDF -->
*/
package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	mrand "math/rand"
	"strings"

	"location-tracker/clients"
	"location-tracker/types"
)

// CommercialService handles commercial real estate and governance queries
type CommercialService struct {
	perplexityClient *clients.PerplexityClient
}

// NewCommercialService creates a new CommercialService instance
func NewCommercialService(perplexityAPIKey string) *CommercialService {
	return &CommercialService{
		perplexityClient: clients.NewPerplexityClient(perplexityAPIKey),
	}
}

// SearchCommercialRealEstate searches for commercial properties and governing bodies near coordinates
// Returns properties, governing bodies, query coordinates, and error
func (s *CommercialService) SearchCommercialRealEstate(baseLat, baseLng float64, userKeywords []string) ([]types.CommercialPropertyDetails, []types.GoverningBody, float64, float64, error) {
	// Generate random location within 10 mile radius
	queryLat, queryLng := s.generateRandomLocationInRadius(baseLat, baseLng, 10.0)
	log.Printf("🎲 Searching for commercial real estate at random location: (%.6f, %.6f) within 10 miles of base", queryLat, queryLng)

	// Build satirical prompt that references user keywords if available
	keywordContext := ""
	if len(userKeywords) > 0 {
		keywordContext = fmt.Sprintf("\n\nContext keywords from user: %s. Feel free to incorporate these themes into the property descriptions.", strings.Join(userKeywords, ", "))
	}

	prompt := fmt.Sprintf(`Find commercial properties, governing authorities, and businesses near (%.6f, %.6f).

Return JSON with:
1. Available commercial spaces (address, type, sqft, price)
2. Current businesses (name, type, contact)
3. Local governing bodies (city council, planning commission, zoning board, civic orgs)%s

JSON format:
{
  "properties": [{"address": "...", "property_type": "retail|office|industrial", "status": "available|leased", "square_footage": "...", "price_info": "...", "current_business": "...", "business_type": "...", "description": "...", "contact_info": {"phone": "...", "email": "...", "website": "..."}}],
  "governing_bodies": [{"name": "...", "type": "city_council|planning|zoning|civic", "jurisdiction": "...", "contact": "..."}]
}

Return ONLY valid JSON.`, queryLat, queryLng, keywordContext)

	messages := []types.PerplexityMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Use client to make the API call
	content, err := s.perplexityClient.ChatCompletion("sonar", messages)
	if err != nil {
		log.Printf("⚠️  Perplexity API call failed: %v", err)
		return []types.CommercialPropertyDetails{}, []types.GoverningBody{}, queryLat, queryLng, nil
	}

	// Try to extract JSON from response (sometimes wrapped in markdown)
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		content = content[jsonStart : jsonEnd+1]
	}

	var result struct {
		Properties      []types.CommercialPropertyDetails `json:"properties"`
		GoverningBodies []types.GoverningBody             `json:"governing_bodies"`
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		log.Printf("⚠️  Failed to parse commercial real estate JSON, raw content: %s", content)
		return []types.CommercialPropertyDetails{}, []types.GoverningBody{}, queryLat, queryLng, nil
	}

	log.Printf("🏢 Found %d commercial properties and %d governing bodies near (%.6f, %.6f)",
		len(result.Properties), len(result.GoverningBodies), queryLat, queryLng)

	return result.Properties, result.GoverningBodies, queryLat, queryLng, nil
}

// generateRandomLocationInRadius generates a random lat/lng within radiusMiles of the base coordinates
func (s *CommercialService) generateRandomLocationInRadius(baseLat, baseLng, radiusMiles float64) (float64, float64) {
	// Convert radius to degrees (rough approximation: 1 degree ≈ 69 miles)
	radiusDegrees := radiusMiles / 69.0

	// Generate random angle and distance
	angle := mrand.Float64() * 2 * math.Pi
	distance := mrand.Float64() * radiusDegrees

	// Calculate offset
	latOffset := distance * math.Cos(angle)
	lngOffset := distance * math.Sin(angle) / math.Cos(baseLat*math.Pi/180)

	return baseLat + latOffset, baseLng + lngOffset
}
