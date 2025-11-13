/*
# Module: services/business_search.go
Business discovery and nearby search using Google Places API.

## Linked Modules
- [clients/google_places](../clients/google_places.go): Google Places API client for business queries
- [types/business](../types/business.go): Business data structures and models
- [storage/cache](../storage/cache.go): Redis cache for API response caching

## Tags
business-logic, search, geolocation, api-client

## Exports
BusinessSearchService, NewBusinessSearchService, SearchNearby, GetBusinessType

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "services/business_search.go" ;
    code:description "Business discovery and nearby search using Google Places API" ;
    code:dependsOn <../types/business.go>, <../storage/cache.go> ;
    code:integratesWith <../clients/google_places.go> ;
    code:exports :BusinessSearchService, :SearchNearby, :GetBusinessType ;
    code:tags "business-logic", "search", "geolocation" .
<!-- End LinkedDoc RDF -->
*/
package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/justin4957/ec2-test-apps/location-tracker/clients"
	"github.com/justin4957/ec2-test-apps/location-tracker/storage"
	"github.com/justin4957/ec2-test-apps/location-tracker/types"
)

// BusinessSearchService handles business discovery logic
type BusinessSearchService struct {
	placesClient *clients.GooglePlacesClient
	cache        *storage.CacheStore
	radiusMeters int
}

// NewBusinessSearchService creates a new business search service
func NewBusinessSearchService(placesClient *clients.GooglePlacesClient, cache *storage.CacheStore) *BusinessSearchService {
	return &BusinessSearchService{
		placesClient: placesClient,
		cache:        cache,
		radiusMeters: 500, // Default 500m radius
	}
}

// SearchNearby searches for businesses near the given coordinates
func (s *BusinessSearchService) SearchNearby(ctx context.Context, lat, lng float64) ([]types.Business, error) {
	// Generate cache key
	cacheKey := fmt.Sprintf("businesses:%f,%f:%d", lat, lng, s.radiusMeters)

	// Check cache first
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var businesses []types.Business
		if err := json.Unmarshal([]byte(cached), &businesses); err == nil {
			log.Printf("Cache hit for businesses near (%f, %f)", lat, lng)
			return businesses, nil
		}
	}

	// Cache miss - query Google Places API
	log.Printf("Searching for businesses near (%f, %f) with radius %dm", lat, lng, s.radiusMeters)

	businesses, err := s.placesClient.SearchNearby(ctx, lat, lng, s.radiusMeters)
	if err != nil {
		return nil, fmt.Errorf("failed to search nearby businesses: %w", err)
	}

	// Enrich business data
	for i := range businesses {
		businesses[i].Type = s.GetBusinessType(businesses[i].Types)
		businesses[i].DiscoveredAt = time.Now()
	}

	// Cache the results (TTL: 1 hour)
	if data, err := json.Marshal(businesses); err == nil {
		s.cache.Set(ctx, cacheKey, string(data), time.Hour)
	}

	log.Printf("Found %d businesses near (%f, %f)", len(businesses), lat, lng)

	return businesses, nil
}

// GetBusinessType determines the primary business type from Google's type list
func (s *BusinessSearchService) GetBusinessType(types []string) string {
	// Priority order for business types
	priority := map[string]int{
		"restaurant":    10,
		"cafe":          9,
		"bar":           8,
		"store":         7,
		"shopping_mall": 6,
		"bank":          5,
		"hospital":      4,
		"school":        3,
		"park":          2,
	}

	bestType := "other"
	bestPriority := 0

	for _, t := range types {
		if p, ok := priority[t]; ok && p > bestPriority {
			bestType = t
			bestPriority = p
		}
	}

	return bestType
}

// SearchByKeyword searches businesses by keyword query
func (s *BusinessSearchService) SearchByKeyword(ctx context.Context, lat, lng float64, keyword string) ([]types.Business, error) {
	log.Printf("Searching for '%s' near (%f, %f)", keyword, lat, lng)

	// Use Places Text Search API
	businesses, err := s.placesClient.SearchByText(ctx, lat, lng, keyword, s.radiusMeters)
	if err != nil {
		return nil, fmt.Errorf("failed to search by keyword: %w", err)
	}

	return businesses, nil
}

// SetRadius updates the search radius in meters
func (s *BusinessSearchService) SetRadius(meters int) {
	if meters > 0 && meters <= 50000 { // Max 50km
		s.radiusMeters = meters
	}
}
