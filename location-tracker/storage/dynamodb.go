/*
# Module: storage/dynamodb.go
Consolidated DynamoDB repository implementations for all data types.

## Linked Modules
- [storage/repository](./repository.go) - Repository interfaces
- [types/location](../types/location.go) - Location data structures
- [types/commercial](../types/commercial.go) - Commercial real estate data structures
- [types/tip](../types/tip.go) - Anonymous tip data structures

## Tags
storage, dynamodb, persistence, repository

## Exports
LocationDynamoDBRepository, CommercialDynamoDBRepository, TipDynamoDBRepository

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "storage/dynamodb.go" ;
    code:description "Consolidated DynamoDB repository implementations for all data types" ;
    code:linksTo [
        code:name "storage/repository" ;
        code:path "./repository.go" ;
        code:relationship "Repository interfaces"
    ], [
        code:name "types/location" ;
        code:path "../types/location.go" ;
        code:relationship "Location data structures"
    ], [
        code:name "types/commercial" ;
        code:path "../types/commercial.go" ;
        code:relationship "Commercial real estate data structures"
    ], [
        code:name "types/tip" ;
        code:path "../types/tip.go" ;
        code:relationship "Anonymous tip data structures"
    ] ;
    code:exports :LocationDynamoDBRepository, :CommercialDynamoDBRepository, :TipDynamoDBRepository ;
    code:tags "storage", "dynamodb", "persistence", "repository" .
<!-- End LinkedDoc RDF -->
*/
package storage

import (
	"context"
	"fmt"
	"log"
	"math"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"location-tracker/types"
)

// LocationDynamoDBRepository implements LocationRepository using DynamoDB
type LocationDynamoDBRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewLocationDynamoDBRepository creates a new DynamoDB location repository
func NewLocationDynamoDBRepository(client *dynamodb.Client, tableName string) *LocationDynamoDBRepository {
	return &LocationDynamoDBRepository{
		client:    client,
		tableName: tableName,
	}
}

// Save stores a location in DynamoDB
func (r *LocationDynamoDBRepository) Save(location types.Location) error {
	if r.client == nil {
		return fmt.Errorf("DynamoDB client not initialized")
	}

	ctx := context.Background()

	item, err := attributevalue.MarshalMap(location)
	if err != nil {
		return fmt.Errorf("failed to marshal location: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to save location to DynamoDB: %w", err)
	}

	log.Printf("💾 Location saved to DynamoDB: device_id=%s", location.DeviceID)
	return nil
}

// GetByDeviceID retrieves a location by device ID
func (r *LocationDynamoDBRepository) GetByDeviceID(deviceID string) (*types.Location, error) {
	if r.client == nil {
		return nil, fmt.Errorf("DynamoDB client not initialized")
	}

	ctx := context.Background()

	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]dynamodbtypes.AttributeValue{
			"device_id": &dynamodbtypes.AttributeValueMemberS{Value: deviceID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get location: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("location not found")
	}

	var location types.Location
	if err := attributevalue.UnmarshalMap(result.Item, &location); err != nil {
		return nil, fmt.Errorf("failed to unmarshal location: %w", err)
	}

	return &location, nil
}

// GetAll retrieves all locations as a map keyed by device ID
func (r *LocationDynamoDBRepository) GetAll() (map[string]types.Location, error) {
	if r.client == nil {
		return nil, fmt.Errorf("DynamoDB client not initialized")
	}

	ctx := context.Background()

	locations := make(map[string]types.Location)
	var lastEvaluatedKey map[string]dynamodbtypes.AttributeValue

	for {
		input := &dynamodb.ScanInput{
			TableName: aws.String(r.tableName),
		}
		if lastEvaluatedKey != nil {
			input.ExclusiveStartKey = lastEvaluatedKey
		}

		result, err := r.client.Scan(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to scan locations: %w", err)
		}

		for _, item := range result.Items {
			var location types.Location
			if err := attributevalue.UnmarshalMap(item, &location); err != nil {
				log.Printf("⚠️  Failed to unmarshal location: %v", err)
				continue
			}
			locations[location.DeviceID] = location
		}

		lastEvaluatedKey = result.LastEvaluatedKey
		if lastEvaluatedKey == nil {
			break
		}
	}

	log.Printf("📍 Loaded %d locations from DynamoDB", len(locations))
	return locations, nil
}

// CommercialDynamoDBRepository implements CommercialRepository using DynamoDB
type CommercialDynamoDBRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewCommercialDynamoDBRepository creates a new DynamoDB commercial repository
func NewCommercialDynamoDBRepository(client *dynamodb.Client, tableName string) *CommercialDynamoDBRepository {
	return &CommercialDynamoDBRepository{
		client:    client,
		tableName: tableName,
	}
}

// Save stores commercial real estate data in DynamoDB
func (r *CommercialDynamoDBRepository) Save(commercial types.CommercialRealEstate) error {
	if r.client == nil {
		return fmt.Errorf("DynamoDB client not initialized")
	}

	ctx := context.Background()

	item, err := attributevalue.MarshalMap(commercial)
	if err != nil {
		return fmt.Errorf("failed to marshal commercial real estate: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to save commercial real estate to DynamoDB: %w", err)
	}

	log.Printf("💾 Commercial real estate info saved to DynamoDB: %s", commercial.LocationName)
	return nil
}

// GetByLocation retrieves commercial data near a location
func (r *CommercialDynamoDBRepository) GetByLocation(lat, lng float64, radiusMiles float64) (*types.CommercialRealEstate, error) {
	if r.client == nil {
		return nil, fmt.Errorf("DynamoDB client not initialized")
	}

	ctx := context.Background()

	result, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan commercial real estate: %w", err)
	}

	// Find closest match within radius
	var closest *types.CommercialRealEstate
	minDistance := radiusMiles

	for _, item := range result.Items {
		var commercial types.CommercialRealEstate
		if err := attributevalue.UnmarshalMap(item, &commercial); err != nil {
			continue
		}

		// Calculate distance using Haversine formula
		distance := haversineDistance(lat, lng, commercial.QueryLat, commercial.QueryLng)
		if distance < minDistance {
			minDistance = distance
			closest = &commercial
		}
	}

	if closest == nil {
		return nil, fmt.Errorf("no commercial real estate found within radius")
	}

	return closest, nil
}

// GetByName retrieves commercial data by location name
func (r *CommercialDynamoDBRepository) GetByName(locationName string) (*types.CommercialRealEstate, error) {
	if r.client == nil {
		return nil, fmt.Errorf("DynamoDB client not initialized")
	}

	ctx := context.Background()

	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]dynamodbtypes.AttributeValue{
			"location_name": &dynamodbtypes.AttributeValueMemberS{Value: locationName},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get commercial real estate: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("commercial real estate not found")
	}

	var commercial types.CommercialRealEstate
	if err := attributevalue.UnmarshalMap(result.Item, &commercial); err != nil {
		return nil, fmt.Errorf("failed to unmarshal commercial real estate: %w", err)
	}

	return &commercial, nil
}

// TipDynamoDBRepository implements TipRepository using DynamoDB
type TipDynamoDBRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewTipDynamoDBRepository creates a new DynamoDB tip repository
func NewTipDynamoDBRepository(client *dynamodb.Client, tableName string) *TipDynamoDBRepository {
	return &TipDynamoDBRepository{
		client:    client,
		tableName: tableName,
	}
}

// Save stores an anonymous tip in DynamoDB
func (r *TipDynamoDBRepository) Save(tip types.AnonymousTip) error {
	if r.client == nil {
		return fmt.Errorf("DynamoDB client not initialized")
	}

	ctx := context.Background()

	item, err := attributevalue.MarshalMap(tip)
	if err != nil {
		return fmt.Errorf("failed to marshal tip: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to save tip to DynamoDB: %w", err)
	}

	log.Printf("💾 Anonymous tip saved to DynamoDB: %s", tip.ID)
	return nil
}

// GetByID retrieves a tip by ID
func (r *TipDynamoDBRepository) GetByID(tipID string) (*types.AnonymousTip, error) {
	if r.client == nil {
		return nil, fmt.Errorf("DynamoDB client not initialized")
	}

	ctx := context.Background()

	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: tipID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get tip: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("tip not found")
	}

	var tip types.AnonymousTip
	if err := attributevalue.UnmarshalMap(result.Item, &tip); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tip: %w", err)
	}

	return &tip, nil
}

// GetRecent retrieves the most recent tips (up to limit)
func (r *TipDynamoDBRepository) GetRecent(limit int) ([]types.AnonymousTip, error) {
	if r.client == nil {
		return nil, fmt.Errorf("DynamoDB client not initialized")
	}

	ctx := context.Background()

	result, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
		Limit:     aws.Int32(int32(limit)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan tips: %w", err)
	}

	tips := make([]types.AnonymousTip, 0, len(result.Items))
	for _, item := range result.Items {
		var tip types.AnonymousTip
		if err := attributevalue.UnmarshalMap(item, &tip); err != nil {
			log.Printf("⚠️  Failed to unmarshal tip: %v", err)
			continue
		}
		tips = append(tips, tip)
	}

	return tips, nil
}

// GetAll retrieves all tips from DynamoDB
func (r *TipDynamoDBRepository) GetAll() ([]types.AnonymousTip, error) {
	if r.client == nil {
		return nil, fmt.Errorf("DynamoDB client not initialized")
	}

	ctx := context.Background()

	tips := make([]types.AnonymousTip, 0)
	var lastEvaluatedKey map[string]dynamodbtypes.AttributeValue

	for {
		input := &dynamodb.ScanInput{
			TableName: aws.String(r.tableName),
		}
		if lastEvaluatedKey != nil {
			input.ExclusiveStartKey = lastEvaluatedKey
		}

		result, err := r.client.Scan(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tips: %w", err)
		}

		for _, item := range result.Items {
			var tip types.AnonymousTip
			if err := attributevalue.UnmarshalMap(item, &tip); err != nil {
				log.Printf("⚠️  Failed to unmarshal tip: %v", err)
				continue
			}
			tips = append(tips, tip)
		}

		lastEvaluatedKey = result.LastEvaluatedKey
		if lastEvaluatedKey == nil {
			break
		}
	}

	log.Printf("💡 Loaded %d tips from DynamoDB", len(tips))
	return tips, nil
}

// haversineDistance calculates the distance between two lat/lng points in miles
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusMiles = 3959.0

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMiles * c
}
