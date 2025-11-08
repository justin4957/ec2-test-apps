#!/bin/bash

# Script to create DynamoDB tables for anonymous tips system
# Run this script to set up the required tables in AWS

set -e

echo "🚀 Creating DynamoDB tables for anonymous tips system..."

# Create anonymous tips table
echo "📝 Creating location-tracker-anonymous-tips table..."
aws dynamodb create-table \
    --table-name location-tracker-anonymous-tips \
    --attribute-definitions \
        AttributeName=id,AttributeType=S \
        AttributeName=timestamp,AttributeType=S \
        AttributeName=user_hash,AttributeType=S \
        AttributeName=moderation_status,AttributeType=S \
    --key-schema \
        AttributeName=id,KeyType=HASH \
    --global-secondary-indexes \
        "[
            {
                \"IndexName\": \"user_hash-timestamp-index\",
                \"KeySchema\": [
                    {\"AttributeName\":\"user_hash\",\"KeyType\":\"HASH\"},
                    {\"AttributeName\":\"timestamp\",\"KeyType\":\"RANGE\"}
                ],
                \"Projection\": {\"ProjectionType\":\"ALL\"},
                \"ProvisionedThroughput\": {\"ReadCapacityUnits\":5,\"WriteCapacityUnits\":5}
            },
            {
                \"IndexName\": \"moderation_status-timestamp-index\",
                \"KeySchema\": [
                    {\"AttributeName\":\"moderation_status\",\"KeyType\":\"HASH\"},
                    {\"AttributeName\":\"timestamp\",\"KeyType\":\"RANGE\"}
                ],
                \"Projection\": {\"ProjectionType\":\"ALL\"},
                \"ProvisionedThroughput\": {\"ReadCapacityUnits\":5,\"WriteCapacityUnits\":5}
            }
        ]" \
    --provisioned-throughput ReadCapacityUnits=10,WriteCapacityUnits=10 \
    --region us-east-1

echo "✅ Anonymous tips table created successfully!"

# Create banned users table
echo "🚫 Creating location-tracker-banned-users table..."
aws dynamodb create-table \
    --table-name location-tracker-banned-users \
    --attribute-definitions \
        AttributeName=user_hash,AttributeType=S \
    --key-schema \
        AttributeName=user_hash,KeyType=HASH \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
    --region us-east-1

echo "✅ Banned users table created successfully!"

# Wait for tables to become active
echo "⏳ Waiting for tables to become active..."
aws dynamodb wait table-exists --table-name location-tracker-anonymous-tips --region us-east-1
aws dynamodb wait table-exists --table-name location-tracker-banned-users --region us-east-1

echo "🎉 All tables created and ready!"
echo ""
echo "📋 Summary:"
echo "  • location-tracker-anonymous-tips (stores anonymous tips)"
echo "  • location-tracker-banned-users (stores banned user hashes)"
echo ""
echo "🔑 Environment variables needed:"
echo "  export OPENAI_API_KEY=<your-openai-api-key>"
echo "  export TIP_ENCRYPTION_KEY=<64-hex-char-key>"
echo ""
echo "💡 To generate an encryption key:"
echo "  openssl rand -hex 32"
