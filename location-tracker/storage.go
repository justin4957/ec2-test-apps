package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// DataStore interface abstracts storage operations (DynamoDB or Solid Pod)
type DataStore interface {
	SaveLocation(ctx context.Context, userID string, location Location) error
	GetLocations(ctx context.Context, userID string) ([]Location, error)
	SaveErrorLog(ctx context.Context, userID string, errorLog ErrorLog) error
	GetErrorLogs(ctx context.Context, userID string) ([]ErrorLog, error)
	GetStorageInfo(ctx context.Context, userID string) (*StorageInfo, error)
}

// StorageInfo describes where and how data is stored
type StorageInfo struct {
	Type        string    `json:"type"`         // "dynamodb", "solid_pod"
	Location    string    `json:"location"`     // Table name or Pod URL
	UserID      string    `json:"user_id"`      // Session ID or WebID
	LastSync    time.Time `json:"last_sync"`
	RecordCount int       `json:"record_count"`
}

// DynamoDBDataStore implements DataStore using DynamoDB (existing behavior)
type DynamoDBDataStore struct{}

func (s *DynamoDBDataStore) SaveLocation(ctx context.Context, userID string, location Location) error {
	if !useDynamoDB {
		return fmt.Errorf("DynamoDB not enabled")
	}
	saveLocationToDynamoDB(location)
	return nil
}

func (s *DynamoDBDataStore) GetLocations(ctx context.Context, userID string) ([]Location, error) {
	locationMutex.RLock()
	defer locationMutex.RUnlock()

	locs := make([]Location, 0, len(locations))
	for _, loc := range locations {
		locs = append(locs, loc)
	}
	return locs, nil
}

func (s *DynamoDBDataStore) SaveErrorLog(ctx context.Context, userID string, errorLog ErrorLog) error {
	if !useDynamoDB {
		return fmt.Errorf("DynamoDB not enabled")
	}
	go saveErrorLogToDynamoDB(errorLog)
	return nil
}

func (s *DynamoDBDataStore) GetErrorLogs(ctx context.Context, userID string) ([]ErrorLog, error) {
	errorLogMutex.RLock()
	defer errorLogMutex.RUnlock()

	logs := make([]ErrorLog, len(errorLogs))
	copy(logs, errorLogs)
	return logs, nil
}

func (s *DynamoDBDataStore) GetStorageInfo(ctx context.Context, userID string) (*StorageInfo, error) {
	return &StorageInfo{
		Type:        "dynamodb",
		Location:    locationsTableName,
		UserID:      userID,
		LastSync:    time.Now(),
		RecordCount: len(locations),
	}, nil
}

// SolidPodDataStore implements DataStore using Solid Pods (new behavior)
type SolidPodDataStore struct {
	Session *SolidSession
}

func (s *SolidPodDataStore) SaveLocation(ctx context.Context, userID string, location Location) error {
	if s.Session == nil {
		return fmt.Errorf("no Solid session")
	}

	// Convert location to Turtle RDF format
	turtle := locationToTurtle(location)

	// Construct resource path in Pod
	// /private/location-tracker/locations/YYYY/MM/location-TIMESTAMP.ttl
	resourcePath := fmt.Sprintf("private/location-tracker/locations/%s/location-%s.ttl",
		location.Timestamp.Format("2006/01"),
		location.Timestamp.Format("2006-01-02T15:04:05Z"))

	// Write to Pod
	err := writeToPod(s.Session, resourcePath, []byte(turtle), "text/turtle")
	if err != nil {
		return fmt.Errorf("failed to write location to Pod: %w", err)
	}

	log.Printf("💾 [Solid] Location saved to Pod: %s", resourcePath)
	return nil
}

func (s *SolidPodDataStore) GetLocations(ctx context.Context, userID string) ([]Location, error) {
	if s.Session == nil {
		return nil, fmt.Errorf("no Solid session")
	}

	// In PoC, we'll list containers and read all location files
	containerPath := "private/location-tracker/locations/"
	resources, err := listPodContainers(s.Session, containerPath)
	if err != nil {
		log.Printf("⚠️  Failed to list Pod locations: %v", err)
		return []Location{}, nil // Return empty instead of error for PoC
	}

	locations := make([]Location, 0)
	for _, resourceURL := range resources {
		// Read each location file
		resource, err := readFromPod(s.Session, resourceURL)
		if err != nil {
			log.Printf("⚠️  Failed to read location from Pod: %v", err)
			continue
		}

		// Parse Turtle to Location struct
		loc, err := turtleToLocation(string(resource.Data))
		if err != nil {
			log.Printf("⚠️  Failed to parse location from Turtle: %v", err)
			continue
		}

		locations = append(locations, loc)
	}

	log.Printf("📥 [Solid] Retrieved %d locations from Pod", len(locations))
	return locations, nil
}

func (s *SolidPodDataStore) SaveErrorLog(ctx context.Context, userID string, errorLog ErrorLog) error {
	if s.Session == nil {
		return fmt.Errorf("no Solid session")
	}

	// Convert error log to JSON-LD format
	jsonLD := errorLogToJSONLD(errorLog)

	// Construct resource path
	resourcePath := fmt.Sprintf("private/location-tracker/error-logs/%s/error-%s.jsonld",
		errorLog.Timestamp.Format("2006/01"),
		errorLog.Timestamp.Format("2006-01-02T15:04:05Z"))

	// Write to Pod
	err := writeToPod(s.Session, resourcePath, []byte(jsonLD), "application/ld+json")
	if err != nil {
		return fmt.Errorf("failed to write error log to Pod: %w", err)
	}

	log.Printf("💾 [Solid] Error log saved to Pod: %s", resourcePath)
	return nil
}

func (s *SolidPodDataStore) GetErrorLogs(ctx context.Context, userID string) ([]ErrorLog, error) {
	if s.Session == nil {
		return nil, fmt.Errorf("no Solid session")
	}

	// Similar to GetLocations, list and read error log resources
	containerPath := "private/location-tracker/error-logs/"
	resources, err := listPodContainers(s.Session, containerPath)
	if err != nil {
		log.Printf("⚠️  Failed to list Pod error logs: %v", err)
		return []ErrorLog{}, nil
	}

	errorLogs := make([]ErrorLog, 0)
	for _, resourceURL := range resources {
		resource, err := readFromPod(s.Session, resourceURL)
		if err != nil {
			continue
		}

		errorLog, err := jsonLDToErrorLog(string(resource.Data))
		if err != nil {
			continue
		}

		errorLogs = append(errorLogs, errorLog)
	}

	log.Printf("📥 [Solid] Retrieved %d error logs from Pod", len(errorLogs))
	return errorLogs, nil
}

func (s *SolidPodDataStore) GetStorageInfo(ctx context.Context, userID string) (*StorageInfo, error) {
	if s.Session == nil {
		return nil, fmt.Errorf("no Solid session")
	}

	return &StorageInfo{
		Type:        "solid_pod",
		Location:    s.Session.PodURL,
		UserID:      s.Session.WebID,
		LastSync:    time.Now(),
		RecordCount: 0, // Would need to count resources
	}, nil
}

// RDF Serialization helpers (simplified for PoC)

func locationToTurtle(loc Location) string {
	// Convert Location to Turtle RDF format
	return fmt.Sprintf(`@prefix schema: <http://schema.org/> .
@prefix geo: <http://www.w3.org/2003/01/geo/wgs84_pos#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

<#location>
    a schema:Place ;
    geo:lat "%.6f"^^xsd:decimal ;
    geo:long "%.6f"^^xsd:decimal ;
    schema:geo [
        a schema:GeoCoordinates ;
        schema:latitude "%.6f" ;
        schema:longitude "%.6f"
    ] ;
    schema:accuracy "%.2f"^^xsd:decimal ;
    schema:dateCreated "%s"^^xsd:dateTime ;
    schema:identifier "%s" .
`,
		loc.Latitude, loc.Longitude,
		loc.Latitude, loc.Longitude,
		loc.Accuracy,
		loc.Timestamp.Format(time.RFC3339),
		loc.DeviceID,
	)
}

func turtleToLocation(turtle string) (Location, error) {
	// In PoC, return empty location
	// In production, parse Turtle RDF and extract fields
	return Location{
		Latitude:  0.0,
		Longitude: 0.0,
		Accuracy:  0.0,
		Timestamp: time.Now(),
		DeviceID:  "unknown",
	}, nil
}

func errorLogToJSONLD(log ErrorLog) string {
	// Convert ErrorLog to JSON-LD format
	data := map[string]interface{}{
		"@context": map[string]string{
			"@vocab":  "http://schema.org/",
			"geo":     "http://www.w3.org/2003/01/geo/wgs84_pos#",
			"slogan":  "http://example.org/vocab#slogan",
			"gifURL":  "http://example.org/vocab#gifURL",
			"songURL": "http://example.org/vocab#songURL",
		},
		"@type":       "Report",
		"@id":         "#error-" + log.ID,
		"name":        "Application Error",
		"description": log.Message,
		"dateCreated": log.Timestamp.Format(time.RFC3339),
		"slogan":      log.Slogan,
	}

	if log.GifURL != "" {
		data["associatedMedia"] = map[string]string{
			"@type":      "ImageObject",
			"contentUrl": log.GifURL,
		}
	}

	if log.SongURL != "" {
		data["audio"] = map[string]string{
			"@type":      "AudioObject",
			"contentUrl": log.SongURL,
			"name":       log.SongTitle,
			"byArtist":   log.SongArtist,
		}
	}

	jsonBytes, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonBytes)
}

func jsonLDToErrorLog(jsonld string) (ErrorLog, error) {
	// In PoC, return empty error log
	// In production, parse JSON-LD and extract fields
	return ErrorLog{
		ID:        "unknown",
		Message:   "",
		Timestamp: time.Now(),
	}, nil
}

// getDataStore returns appropriate data store based on authentication type
func getDataStore(r *http.Request) DataStore {
	// Check if user has Solid session
	solidSession := getSolidSession(r)
	if solidSession != nil && solidEnabled {
		log.Printf("📦 Using Solid Pod storage for user: %s", solidSession.WebID)
		return &SolidPodDataStore{Session: solidSession}
	}

	// Check if user has password session
	if isAuthenticated(r) {
		log.Printf("📦 Using DynamoDB storage for password-authenticated user")
		return &DynamoDBDataStore{}
	}

	// Default to DynamoDB for unauthenticated requests (public data)
	return &DynamoDBDataStore{}
}

// handleStorageInfo returns information about current storage backend
func handleStorageInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication (either type)
	if !isAuthenticated(r) && !isSolidAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	store := getDataStore(r)
	info, err := store.GetStorageInfo(context.Background(), "current-user")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get storage info: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
