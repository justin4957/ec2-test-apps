/*
# Module: types/location.go
Location tracking data structures and types.

## Linked Modules
(None - types package has no dependencies)

## Tags
data-types, location

## Exports
Location

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "types/location.go" ;
    code:description "Location tracking data structures and types" ;
    code:exports :Location ;
    code:tags "data-types", "location" .
<!-- End LinkedDoc RDF -->
*/
package types

import "time"

// Location represents a tracked geographic location with metadata
type Location struct {
	Latitude     float64   `json:"latitude" dynamodbav:"latitude"`
	Longitude    float64   `json:"longitude" dynamodbav:"longitude"`
	Accuracy     float64   `json:"accuracy" dynamodbav:"accuracy"`
	Timestamp    time.Time `json:"timestamp" dynamodbav:"timestamp"`
	DeviceID     string    `json:"device_id" dynamodbav:"device_id"`
	UserAgent    string    `json:"user_agent" dynamodbav:"user_agent"`
	Simulated    bool      `json:"simulated,omitempty" dynamodbav:"simulated"`
	LocationName string    `json:"location_name,omitempty" dynamodbav:"location_name"`
}
