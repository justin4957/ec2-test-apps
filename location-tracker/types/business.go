/*
# Module: types/business.go
Business search and discovery data structures.

## Linked Modules
(None - types package has no dependencies)

## Tags
data-types, business

## Exports
Business, GooglePlacesResponse

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "types/business.go" ;
    code:description "Business search and discovery data structures" ;
    code:exports :Business, :GooglePlacesResponse ;
    code:tags "data-types", "business" .
<!-- End LinkedDoc RDF -->
*/
package types

// Business represents a nearby business from Google Maps
type Business struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Address  string   `json:"address"`
	PlaceID  string   `json:"place_id"`
	Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"location"`
}

// GooglePlacesResponse represents Google Maps API response types
type GooglePlacesResponse struct {
	Results []struct {
		Name         string   `json:"name"`
		Types        []string `json:"types"`
		PlaceID      string   `json:"place_id"`
		FormattedAddress string `json:"formatted_address"`
		Geometry struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
	} `json:"results"`
	Status string `json:"status"`
}
