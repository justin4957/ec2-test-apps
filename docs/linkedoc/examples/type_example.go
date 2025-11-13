/*
# Module: types/location.go
Location data structure for tracking user locations with geospatial data.

## Linked Modules
(None - types are typically leaf nodes with no dependencies)

## Tags
data-types, location, geolocation

## Exports
Location, LocationRequest, LocationResponse

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Type ;
    code:name "types/location.go" ;
    code:description "Location data structure for tracking user locations" ;
    code:exports :Location, :LocationRequest, :LocationResponse ;
    code:tags "data-types", "location" .
<!-- End LinkedDoc RDF -->
*/
package types

import "time"

// Location represents a tracked location point with associated metadata
type Location struct {
	// Unique identifier for this location record
	ID string `json:"id" dynamodbav:"id"`

	// User identifier (hashed for privacy)
	UserID string `json:"user_id" dynamodbav:"user_id"`

	// Geographic coordinates
	Latitude  float64 `json:"latitude" dynamodbav:"latitude"`
	Longitude float64 `json:"longitude" dynamodbav:"longitude"`

	// Accuracy in meters (GPS accuracy)
	Accuracy float64 `json:"accuracy,omitempty" dynamodbav:"accuracy,omitempty"`

	// Altitude in meters above sea level
	Altitude float64 `json:"altitude,omitempty" dynamodbav:"altitude,omitempty"`

	// Timestamps
	Timestamp time.Time `json:"timestamp" dynamodbav:"timestamp"`
	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`

	// Device information
	DeviceID string `json:"device_id,omitempty" dynamodbav:"device_id,omitempty"`
	Platform string `json:"platform,omitempty" dynamodbav:"platform,omitempty"` // "ios", "android", "web"

	// Location context
	Address     string   `json:"address,omitempty" dynamodbav:"address,omitempty"`
	City        string   `json:"city,omitempty" dynamodbav:"city,omitempty"`
	Country     string   `json:"country,omitempty" dynamodbav:"country,omitempty"`
	PostalCode  string   `json:"postal_code,omitempty" dynamodbav:"postal_code,omitempty"`

	// Associated businesses (nearby discoveries)
	NearbyBusinesses []string `json:"nearby_businesses,omitempty" dynamodbav:"nearby_businesses,omitempty"`

	// Metadata
	Source string `json:"source,omitempty" dynamodbav:"source,omitempty"` // "gps", "manual", "simulated"
	Notes  string `json:"notes,omitempty" dynamodbav:"notes,omitempty"`
}

// LocationRequest represents an incoming request to create/update a location
type LocationRequest struct {
	Latitude  float64 `json:"latitude" validate:"required,latitude"`
	Longitude float64 `json:"longitude" validate:"required,longitude"`
	Accuracy  float64 `json:"accuracy,omitempty"`
	Altitude  float64 `json:"altitude,omitempty"`
	DeviceID  string  `json:"device_id,omitempty"`
	Platform  string  `json:"platform,omitempty"`
	Notes     string  `json:"notes,omitempty"`
}

// LocationResponse represents the API response for location queries
type LocationResponse struct {
	Location         Location   `json:"location"`
	NearbyBusinesses []Business `json:"nearby_businesses,omitempty"`
	Message          string     `json:"message,omitempty"`
}

// Validate checks if the location has valid coordinates
func (l *Location) Validate() error {
	if l.Latitude < -90 || l.Latitude > 90 {
		return fmt.Errorf("invalid latitude: must be between -90 and 90")
	}
	if l.Longitude < -180 || l.Longitude > 180 {
		return fmt.Errorf("invalid longitude: must be between -180 and 180")
	}
	return nil
}

// DistanceTo calculates the distance in kilometers to another location using Haversine formula
func (l *Location) DistanceTo(other Location) float64 {
	const earthRadius = 6371.0 // Earth's radius in kilometers

	// Convert degrees to radians
	lat1 := l.Latitude * math.Pi / 180
	lat2 := other.Latitude * math.Pi / 180
	deltaLat := (other.Latitude - l.Latitude) * math.Pi / 180
	deltaLng := (other.Longitude - l.Longitude) * math.Pi / 180

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}
