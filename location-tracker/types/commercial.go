/*
# Module: types/commercial.go
Commercial real estate and governance data structures.

## Linked Modules
(None - types package has no dependencies)

## Tags
data-types, commercial, real-estate

## Exports
CommercialRealEstate, CommercialPropertyDetails, GoverningBody

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "types/commercial.go" ;
    code:description "Commercial real estate and governance data structures" ;
    code:exports :CommercialRealEstate, :CommercialPropertyDetails, :GoverningBody ;
    code:tags "data-types", "commercial", "real-estate" .
<!-- End LinkedDoc RDF -->
*/
package types

import "time"

// CommercialRealEstate represents commercial real estate and associated businesses in an area
type CommercialRealEstate struct {
	ID              string                      `json:"id,omitempty" dynamodbav:"id"`
	LocationName    string                      `json:"location_name" dynamodbav:"location_name"`
	QueryLat        float64                     `json:"query_lat" dynamodbav:"query_lat"`
	QueryLng        float64                     `json:"query_lng" dynamodbav:"query_lng"`
	Properties      []CommercialPropertyDetails `json:"properties" dynamodbav:"properties"`
	GoverningBodies []GoverningBody             `json:"governing_bodies,omitempty" dynamodbav:"governing_bodies"`
	Timestamp       time.Time                   `json:"timestamp" dynamodbav:"timestamp"`
}

// CommercialPropertyDetails stores information about commercial real estate and businesses
type CommercialPropertyDetails struct {
	Address          string                 `json:"address" dynamodbav:"address"`
	PropertyType     string                 `json:"property_type" dynamodbav:"property_type"` // "retail", "office", "industrial", etc.
	Status           string                 `json:"status" dynamodbav:"status"`               // "available", "leased", "for_sale"
	SquareFootage    string                 `json:"square_footage,omitempty" dynamodbav:"square_footage"`
	PriceInfo        string                 `json:"price_info,omitempty" dynamodbav:"price_info"`
	CurrentBusiness  string                 `json:"current_business,omitempty" dynamodbav:"current_business"`
	BusinessType     string                 `json:"business_type,omitempty" dynamodbav:"business_type"`
	Description      string                 `json:"description,omitempty" dynamodbav:"description"`
	ContactInfo      map[string]interface{} `json:"contact_info,omitempty" dynamodbav:"contact_info"`
	AdditionalInfo   map[string]interface{} `json:"additional_info,omitempty" dynamodbav:"additional_info"`
}

// GoverningBody stores information about local government authorities and civic organizations
type GoverningBody struct {
	Name         string `json:"name" dynamodbav:"name"`
	Type         string `json:"type" dynamodbav:"type"`                     // "city_council", "planning", "zoning", "civic"
	Jurisdiction string `json:"jurisdiction,omitempty" dynamodbav:"jurisdiction"` // City/county name
	Contact      string `json:"contact,omitempty" dynamodbav:"contact"`           // Contact info
}
