package solid

import (
	"encoding/json"
	"fmt"
	"time"
)

// SerializeToTurtle converts a location data structure to Turtle RDF format.
// Based on the schema defined in SOLID_DATA_MODELS.md
func SerializeToTurtle(data map[string]interface{}) (string, error) {
	// Extract fields from data
	lat, ok := data["latitude"].(float64)
	if !ok {
		return "", fmt.Errorf("latitude is required and must be a number")
	}

	lng, ok := data["longitude"].(float64)
	if !ok {
		return "", fmt.Errorf("longitude is required and must be a number")
	}

	accuracy := data["accuracy"]
	if accuracy == nil {
		accuracy = 10.0
	}

	timestamp := data["timestamp"]
	if timestamp == nil {
		timestamp = time.Now().Format(time.RFC3339)
	}

	deviceID := data["device_id"]
	if deviceID == nil {
		deviceID = "unknown"
	}

	locality := data["locality"]
	region := data["region"]
	country := data["country"]

	// Build Turtle document
	turtle := fmt.Sprintf(`@prefix schema: <http://schema.org/> .
@prefix geo: <http://www.w3.org/2003/01/geo/wgs84_pos#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix dcterms: <http://purl.org/dc/terms/> .

<#location> a schema:Place, geo:Point ;
    geo:lat "%.6f"^^xsd:decimal ;
    geo:long "%.6f"^^xsd:decimal ;
    schema:geo [
        a schema:GeoCoordinates ;
        schema:latitude %.6f ;
        schema:longitude %.6f ;
    ] ;
    schema:accuracy "%.2f"^^xsd:decimal ;
    dcterms:created "%s"^^xsd:dateTime ;
    schema:identifier "%s"`, lat, lng, lat, lng, accuracy, timestamp, deviceID)

	// Add address if present
	if locality != nil || region != nil || country != nil {
		turtle += ` ;
    schema:address [
        a schema:PostalAddress`

		if locality != nil {
			turtle += fmt.Sprintf(` ;
        schema:addressLocality "%s"`, locality)
		}
		if region != nil {
			turtle += fmt.Sprintf(` ;
        schema:addressRegion "%s"`, region)
		}
		if country != nil {
			turtle += fmt.Sprintf(` ;
        schema:addressCountry "%s"`, country)
		}

		turtle += ` ;
    ]`
	}

	turtle += ` .
`

	return turtle, nil
}

// SerializeToJSONLD converts a location data structure to JSON-LD format.
func SerializeToJSONLD(data map[string]interface{}) (string, error) {
	// Extract fields
	lat, ok := data["latitude"].(float64)
	if !ok {
		return "", fmt.Errorf("latitude is required and must be a number")
	}

	lng, ok := data["longitude"].(float64)
	if !ok {
		return "", fmt.Errorf("longitude is required and must be a number")
	}

	accuracy := data["accuracy"]
	if accuracy == nil {
		accuracy = 10.0
	}

	timestamp := data["timestamp"]
	if timestamp == nil {
		timestamp = time.Now().Format(time.RFC3339)
	}

	deviceID := data["device_id"]
	if deviceID == nil {
		deviceID = "unknown"
	}

	// Build JSON-LD document
	doc := map[string]interface{}{
		"@context": map[string]interface{}{
			"@vocab":   "http://schema.org/",
			"geo":      "http://www.w3.org/2003/01/geo/wgs84_pos#",
			"dcterms":  "http://purl.org/dc/terms/",
			"xsd":      "http://www.w3.org/2001/XMLSchema#",
		},
		"@type": []string{"Place", "geo:Point"},
		"@id":   "#location",
		"geo:lat": lat,
		"geo:long": lng,
		"geo": map[string]interface{}{
			"@type": "GeoCoordinates",
			"latitude": lat,
			"longitude": lng,
		},
		"accuracy": accuracy,
		"dateCreated": timestamp,
		"identifier": deviceID,
	}

	// Add address if present
	locality := data["locality"]
	region := data["region"]
	country := data["country"]

	if locality != nil || region != nil || country != nil {
		address := map[string]interface{}{
			"@type": "PostalAddress",
		}
		if locality != nil {
			address["addressLocality"] = locality
		}
		if region != nil {
			address["addressRegion"] = region
		}
		if country != nil {
			address["addressCountry"] = country
		}
		doc["address"] = address
	}

	result, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON-LD: %w", err)
	}

	return string(result), nil
}

// DeserializeFromTurtle parses Turtle RDF and extracts location data.
// Note: This is a simplified parser for PoC. Production should use a full RDF library.
func DeserializeFromTurtle(turtle string) (map[string]interface{}, error) {
	// This is a placeholder implementation
	// In production, use github.com/deiu/rdf2go or similar library
	return nil, fmt.Errorf("Turtle deserialization not yet implemented - use RDF library in production")
}

// DeserializeFromJSONLD parses JSON-LD and extracts location data.
func DeserializeFromJSONLD(jsonld string) (map[string]interface{}, error) {
	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(jsonld), &doc); err != nil {
		return nil, fmt.Errorf("failed to parse JSON-LD: %w", err)
	}

	result := make(map[string]interface{})

	// Extract latitude and longitude
	if lat, ok := doc["geo:lat"].(float64); ok {
		result["latitude"] = lat
	}
	if lng, ok := doc["geo:long"].(float64); ok {
		result["longitude"] = lng
	}

	// Extract accuracy
	if accuracy, ok := doc["accuracy"].(float64); ok {
		result["accuracy"] = accuracy
	}

	// Extract timestamp
	if timestamp, ok := doc["dateCreated"].(string); ok {
		result["timestamp"] = timestamp
	}

	// Extract device ID
	if deviceID, ok := doc["identifier"].(string); ok {
		result["device_id"] = deviceID
	}

	// Extract address
	if address, ok := doc["address"].(map[string]interface{}); ok {
		if locality, ok := address["addressLocality"].(string); ok {
			result["locality"] = locality
		}
		if region, ok := address["addressRegion"].(string); ok {
			result["region"] = region
		}
		if country, ok := address["addressCountry"].(string); ok {
			result["country"] = country
		}
	}

	return result, nil
}
