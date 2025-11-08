package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// RateLimiter manages submission rate limits for users
type RateLimiter struct {
	limits     map[string][]time.Time // user_hash -> submission timestamps
	maxPerHour int
	mutex      sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxPerHour int) *RateLimiter {
	rl := &RateLimiter{
		limits:     make(map[string][]time.Time),
		maxPerHour: maxPerHour,
	}

	// Start cleanup goroutine (remove old timestamps every 5 minutes)
	go rl.cleanupOldTimestamps()

	return rl
}

// CheckAndRecordSubmission checks if user is within rate limit and records submission
func (rl *RateLimiter) CheckAndRecordSubmission(userHash string) (allowed bool, remaining int, resetTime time.Time) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	hourAgo := now.Add(-1 * time.Hour)
	resetTime = now.Add(1 * time.Hour)

	// Get user's submissions
	timestamps := rl.limits[userHash]

	// Filter to keep only submissions from last hour
	filtered := []time.Time{}
	for _, ts := range timestamps {
		if ts.After(hourAgo) {
			filtered = append(filtered, ts)
		}
	}

	// Check if over limit
	if len(filtered) >= rl.maxPerHour {
		// Find when the oldest submission will expire
		if len(filtered) > 0 {
			resetTime = filtered[0].Add(1 * time.Hour)
		}
		return false, 0, resetTime
	}

	// Record new submission
	filtered = append(filtered, now)
	rl.limits[userHash] = filtered

	remaining = rl.maxPerHour - len(filtered)
	return true, remaining, resetTime
}

// GetRemainingQuota returns how many submissions a user has left
func (rl *RateLimiter) GetRemainingQuota(userHash string) int {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	now := time.Now()
	hourAgo := now.Add(-1 * time.Hour)

	timestamps := rl.limits[userHash]
	count := 0
	for _, ts := range timestamps {
		if ts.After(hourAgo) {
			count++
		}
	}

	remaining := rl.maxPerHour - count
	if remaining < 0 {
		remaining = 0
	}

	return remaining
}

// cleanupOldTimestamps removes timestamps older than 1 hour
func (rl *RateLimiter) cleanupOldTimestamps() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		hourAgo := now.Add(-1 * time.Hour)

		for userHash, timestamps := range rl.limits {
			filtered := []time.Time{}
			for _, ts := range timestamps {
				if ts.After(hourAgo) {
					filtered = append(filtered, ts)
				}
			}

			if len(filtered) == 0 {
				delete(rl.limits, userHash)
			} else {
				rl.limits[userHash] = filtered
			}
		}
		rl.mutex.Unlock()
	}
}

// BanManager manages banned users
type BanManager struct {
	bannedUsers     map[string]time.Time // user_hash -> ban_expiry
	mutex           sync.RWMutex
	dynamoClient    *dynamodb.Client
	bannedTableName string
	useDynamoDB     bool
}

// NewBanManager creates a new ban manager
func NewBanManager(dynamoClient *dynamodb.Client, tableName string) *BanManager {
	bm := &BanManager{
		bannedUsers:     make(map[string]time.Time),
		dynamoClient:    dynamoClient,
		bannedTableName: tableName,
		useDynamoDB:     dynamoClient != nil && tableName != "",
	}

	// Load existing bans from DynamoDB
	if bm.useDynamoDB {
		go bm.loadBannedUsers()
	}

	// Start cleanup goroutine
	go bm.cleanupExpiredBans()

	return bm
}

// BannedUser represents a banned user in DynamoDB
type BannedUser struct {
	UserHash   string    `dynamodbav:"user_hash"`
	BanExpiry  time.Time `dynamodbav:"ban_expiry"`
	Reason     string    `dynamodbav:"reason"`
	BannedAt   time.Time `dynamodbav:"banned_at"`
	BannedBy   string    `dynamodbav:"banned_by,omitempty"`
}

// IsUserBanned checks if a user is currently banned
func (bm *BanManager) IsUserBanned(userHash string) (banned bool, reason string, expiresAt time.Time) {
	bm.mutex.RLock()
	expiry, exists := bm.bannedUsers[userHash]
	bm.mutex.RUnlock()

	if !exists {
		return false, "", time.Time{}
	}

	// Check if ban has expired
	if time.Now().After(expiry) {
		// Ban expired, remove it
		bm.UnbanUser(userHash)
		return false, "", time.Time{}
	}

	return true, "User temporarily banned", expiry
}

// BanUser adds a user to the ban list
func (bm *BanManager) BanUser(userHash string, duration time.Duration, reason string) error {
	expiry := time.Now().Add(duration)

	bm.mutex.Lock()
	bm.bannedUsers[userHash] = expiry
	bm.mutex.Unlock()

	// Persist to DynamoDB if available
	if bm.useDynamoDB {
		bannedUser := BannedUser{
			UserHash:  userHash,
			BanExpiry: expiry,
			Reason:    reason,
			BannedAt:  time.Now(),
			BannedBy:  "system",
		}

		av, err := attributevalue.MarshalMap(bannedUser)
		if err != nil {
			return fmt.Errorf("failed to marshal banned user: %w", err)
		}

		_, err = bm.dynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
			TableName: aws.String(bm.bannedTableName),
			Item:      av,
		})
		if err != nil {
			return fmt.Errorf("failed to save ban to DynamoDB: %w", err)
		}
	}

	return nil
}

// UnbanUser removes a user from the ban list
func (bm *BanManager) UnbanUser(userHash string) error {
	bm.mutex.Lock()
	delete(bm.bannedUsers, userHash)
	bm.mutex.Unlock()

	// Remove from DynamoDB if available
	if bm.useDynamoDB {
		_, err := bm.dynamoClient.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
			TableName: aws.String(bm.bannedTableName),
			Key: map[string]types.AttributeValue{
				"user_hash": &types.AttributeValueMemberS{Value: userHash},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to remove ban from DynamoDB: %w", err)
		}
	}

	return nil
}

// loadBannedUsers loads banned users from DynamoDB on startup
func (bm *BanManager) loadBannedUsers() {
	if !bm.useDynamoDB {
		return
	}

	result, err := bm.dynamoClient.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName: aws.String(bm.bannedTableName),
	})
	if err != nil {
		fmt.Printf("⚠️  Failed to load banned users from DynamoDB: %v\n", err)
		return
	}

	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	now := time.Now()
	for _, item := range result.Items {
		var bannedUser BannedUser
		if err := attributevalue.UnmarshalMap(item, &bannedUser); err != nil {
			continue
		}

		// Only load non-expired bans
		if bannedUser.BanExpiry.After(now) {
			bm.bannedUsers[bannedUser.UserHash] = bannedUser.BanExpiry
		}
	}

	fmt.Printf("📋 Loaded %d active bans from DynamoDB\n", len(bm.bannedUsers))
}

// cleanupExpiredBans removes expired bans from memory every 10 minutes
func (bm *BanManager) cleanupExpiredBans() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		bm.mutex.Lock()
		now := time.Now()
		for userHash, expiry := range bm.bannedUsers {
			if now.After(expiry) {
				delete(bm.bannedUsers, userHash)
			}
		}
		bm.mutex.Unlock()
	}
}
