/*
# Module: services/context.go
Interaction context tracking for fractal continuity across generated content.

## Linked Modules
- [types/context](../types/context.go) - Context data structures
- [types/business](../types/business.go) - Business data structures

## Tags
business-logic, context, fractal, continuity

## Exports
ContextService, NewContextService, UpdateContext, GetContext

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "services/context.go" ;
    code:description "Interaction context tracking for fractal continuity across generated content" ;
    code:linksTo [
        code:name "types/context" ;
        code:path "../types/context.go" ;
        code:relationship "Context data structures"
    ], [
        code:name "types/business" ;
        code:path "../types/business.go" ;
        code:relationship "Business data structures"
    ] ;
    code:exports :ContextService, :NewContextService, :UpdateContext, :GetContext ;
    code:tags "business-logic", "context", "fractal", "continuity" .
<!-- End LinkedDoc RDF -->
*/
package services

import (
	"log"
	"sync"
	"time"

	"location-tracker/types"
)

// ContextService manages the fractal continuity context for user interactions
// All generated content (errors, GIFs, songs, etc.) traces back to user-driven seed events
type ContextService struct {
	lastContext *types.LastInteractionContext
	mu          sync.RWMutex
}

// NewContextService creates a new ContextService instance
func NewContextService() *ContextService {
	return &ContextService{
		lastContext: nil,
	}
}

// UpdateContext updates the fractal continuity context with a new user interaction
// This becomes the seed for all subsequent generated content
func (s *ContextService) UpdateContext(interactionType string, keywords []string, sourceID string, locationName string, lat float64, lng float64, businesses []types.Business, rawContent string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Extract business names
	businessNames := make([]string, 0, len(businesses))
	for _, b := range businesses {
		businessNames = append(businessNames, b.Name)
	}

	s.lastContext = &types.LastInteractionContext{
		Timestamp:       time.Now(),
		InteractionType: interactionType, // "sms", "location_share", "tip_submission", etc.
		Keywords:        keywords,
		SourceID:        sourceID,
		LocationName:    locationName,
		Latitude:        lat,
		Longitude:       lng,
		BusinessNames:   businessNames,
		RawContent:      rawContent,
	}

	log.Printf("🧩 Fractal context updated: %s | Keywords: %v | Location: %s",
		interactionType, keywords, locationName)
}

// GetContext returns the current fractal continuity context (read-only copy)
func (s *ContextService) GetContext() *types.LastInteractionContext {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.lastContext == nil {
		return nil
	}

	// Return a copy to prevent external modification
	contextCopy := *s.lastContext
	return &contextCopy
}

// HasContext returns true if there is an active context
func (s *ContextService) HasContext() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastContext != nil
}

// ClearContext resets the fractal continuity context
func (s *ContextService) ClearContext() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastContext = nil
	log.Printf("🧩 Fractal context cleared")
}
