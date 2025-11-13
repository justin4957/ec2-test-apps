/*
# Module: types/context.go
Interaction context tracking data structures.

## Linked Modules
(None - types package has no dependencies)

## Tags
data-types, context

## Exports
LastInteractionContext

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "types/context.go" ;
    code:description "Interaction context tracking data structures" ;
    code:exports :LastInteractionContext ;
    code:tags "data-types", "context" .
<!-- End LinkedDoc RDF -->
*/
package types

import "time"

// LastInteractionContext represents the last user-driven interaction that serves as the "seed event"
// for all subsequent generated content. This creates fractal continuity where errors, GIFs, songs,
// slogans, and other content all trace back to and are influenced by the last known user interaction.
type LastInteractionContext struct {
	InteractionType string    `json:"interaction_type"` // "location_share", "user_note", "tip_submission"
	Timestamp       time.Time `json:"timestamp"`
	Keywords        []string  `json:"keywords"`         // Extracted keywords that influence content generation
	LocationName    string    `json:"location_name,omitempty"` // For location shares
	Latitude        float64   `json:"latitude,omitempty"`
	Longitude       float64   `json:"longitude,omitempty"`
	BusinessNames   []string  `json:"business_names,omitempty"` // Nearby businesses at location
	RawContent      string    `json:"raw_content,omitempty"`    // Original tip/note text
	SourceID        string    `json:"source_id"`                // ID of the source interaction
}
