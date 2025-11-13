/*
# Module: storage/error_log_dynamodb.go
DynamoDB implementation of ErrorLogRepository.

## Linked Modules
- [storage/repository](./repository.go) - Repository interfaces
- [types/error_log](../types/error_log.go) - Error log data structures

## Tags
storage, dynamodb, error-log, persistence

## Exports
ErrorLogDynamoDBRepository, NewErrorLogDynamoDBRepository

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "storage/error_log_dynamodb.go" ;
    code:description "DynamoDB implementation of ErrorLogRepository" ;
    code:linksTo [
        code:name "storage/repository" ;
        code:path "./repository.go" ;
        code:relationship "Repository interfaces"
    ], [
        code:name "types/error_log" ;
        code:path "../types/error_log.go" ;
        code:relationship "Error log data structures"
    ] ;
    code:exports :ErrorLogDynamoDBRepository, :NewErrorLogDynamoDBRepository ;
    code:tags "storage", "dynamodb", "error-log", "persistence" .
<!-- End LinkedDoc RDF -->
*/
package storage

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"location-tracker/types"
)

// ErrorLogDynamoDBRepository implements ErrorLogRepository using DynamoDB
type ErrorLogDynamoDBRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewErrorLogDynamoDBRepository creates a new DynamoDB error log repository
func NewErrorLogDynamoDBRepository(client *dynamodb.Client, tableName string) *ErrorLogDynamoDBRepository {
	return &ErrorLogDynamoDBRepository{
		client:    client,
		tableName: tableName,
	}
}

// Save stores an error log in DynamoDB
func (r *ErrorLogDynamoDBRepository) Save(errorLog types.ErrorLog) error {
	if r.client == nil {
		return fmt.Errorf("DynamoDB client not initialized")
	}

	ctx := context.Background()

	item, err := attributevalue.MarshalMap(errorLog)
	if err != nil {
		return fmt.Errorf("failed to marshal error log: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to save error log to DynamoDB: %w", err)
	}

	log.Printf("💾 Error log saved to DynamoDB: %s", errorLog.ID)
	return nil
}

// GetByID retrieves an error log by ID from DynamoDB
func (r *ErrorLogDynamoDBRepository) GetByID(id string) (*types.ErrorLog, error) {
	if r.client == nil {
		return nil, fmt.Errorf("DynamoDB client not initialized")
	}

	ctx := context.Background()

	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get error log: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("error log not found")
	}

	var errorLog types.ErrorLog
	if err := attributevalue.UnmarshalMap(result.Item, &errorLog); err != nil {
		return nil, fmt.Errorf("failed to unmarshal error log: %w", err)
	}

	return &errorLog, nil
}

// GetRecent retrieves the most recent error logs (up to limit)
func (r *ErrorLogDynamoDBRepository) GetRecent(limit int) ([]types.ErrorLog, error) {
	if r.client == nil {
		return nil, fmt.Errorf("DynamoDB client not initialized")
	}

	ctx := context.Background()

	result, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
		Limit:     aws.Int32(int32(limit)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan error logs: %w", err)
	}

	errorLogs := make([]types.ErrorLog, 0, len(result.Items))
	for _, item := range result.Items {
		var errorLog types.ErrorLog
		if err := attributevalue.UnmarshalMap(item, &errorLog); err != nil {
			log.Printf("⚠️  Failed to unmarshal error log: %v", err)
			continue
		}
		errorLogs = append(errorLogs, errorLog)
	}

	return errorLogs, nil
}

// GetAll retrieves all error logs from DynamoDB
func (r *ErrorLogDynamoDBRepository) GetAll() ([]types.ErrorLog, error) {
	if r.client == nil {
		return nil, fmt.Errorf("DynamoDB client not initialized")
	}

	ctx := context.Background()

	errorLogs := make([]types.ErrorLog, 0)
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
			return nil, fmt.Errorf("failed to scan error logs: %w", err)
		}

		for _, item := range result.Items {
			var errorLog types.ErrorLog
			if err := attributevalue.UnmarshalMap(item, &errorLog); err != nil {
				log.Printf("⚠️  Failed to unmarshal error log: %v", err)
				continue
			}
			errorLogs = append(errorLogs, errorLog)
		}

		lastEvaluatedKey = result.LastEvaluatedKey
		if lastEvaluatedKey == nil {
			break
		}
	}

	log.Printf("📊 Loaded %d error logs from DynamoDB", len(errorLogs))
	return errorLogs, nil
}
