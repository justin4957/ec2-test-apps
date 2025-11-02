#!/bin/bash

# Deploy both applications to EC2
set -e

# Load environment variables from .env.ec2 if it exists
if [ -f .env.ec2 ]; then
    echo "📋 Loading environment variables from .env.ec2..."
    export $(cat .env.ec2 | grep -v '^#' | xargs)
fi

AWS_REGION=us-east-1
AWS_ACCOUNT_ID=310829530225
INSTANCE_ID=i-04bd2369c252bee39
EC2_KEY_PATH=~/.ssh/ec2-test-apps-key.pem
EC2_USER=ec2-user

# Set defaults if not provided
GIPHY_API_KEY=${GIPHY_API_KEY:-}
OPENAI_API_KEY=${OPENAI_API_KEY:-}
SPOTIFY_CLIENT_ID=${SPOTIFY_CLIENT_ID:-}
SPOTIFY_CLIENT_SECRET=${SPOTIFY_CLIENT_SECRET:-}
SPOTIFY_SEED_GENRES=${SPOTIFY_SEED_GENRES:-}
ERROR_INTERVAL_SECONDS=${ERROR_INTERVAL_SECONDS:-60}
GOOGLE_MAPS_API_KEY=${GOOGLE_MAPS_API_KEY:-}
TRACKER_PASSWORD=${TRACKER_PASSWORD:-}
LOCATION_TRACKER_URL=${LOCATION_TRACKER_URL:-}

# Validate critical environment variables
echo "🔍 Validating environment variables..."
MISSING_VARS=0

if [ -z "$TRACKER_PASSWORD" ]; then
    echo "❌ ERROR: TRACKER_PASSWORD is not set in .env.ec2"
    MISSING_VARS=1
fi

if [ $MISSING_VARS -eq 1 ]; then
    echo ""
    echo "⚠️  Critical environment variables are missing!"
    echo "   Please ensure .env.ec2 exists and contains:"
    echo "   - TRACKER_PASSWORD=your_password"
    echo ""
    echo "   Optional but recommended:"
    echo "   - GOOGLE_MAPS_API_KEY=your_key"
    echo "   - GIPHY_API_KEY=your_key"
    echo "   - OPENAI_API_KEY=your_key"
    echo "   - SPOTIFY_CLIENT_ID=your_id"
    echo "   - SPOTIFY_CLIENT_SECRET=your_secret"
    echo ""
    exit 1
fi

echo "✅ All critical environment variables are set"

# Get instance details
echo "🔍 Getting instance details..."
PUBLIC_DNS=$(aws ec2 describe-instances \
    --instance-ids ${INSTANCE_ID} \
    --region ${AWS_REGION} \
    --query 'Reservations[0].Instances[0].PublicDnsName' \
    --output text)

echo "📦 Deploying to: ${PUBLIC_DNS}"
echo ""

# Deploy all containers via SSH
ssh -o StrictHostKeyChecking=no -i ${EC2_KEY_PATH} ${EC2_USER}@${PUBLIC_DNS} \
    GIPHY_API_KEY="${GIPHY_API_KEY}" \
    OPENAI_API_KEY="${OPENAI_API_KEY}" \
    SPOTIFY_CLIENT_ID="${SPOTIFY_CLIENT_ID}" \
    SPOTIFY_CLIENT_SECRET="${SPOTIFY_CLIENT_SECRET}" \
    SPOTIFY_SEED_GENRES="${SPOTIFY_SEED_GENRES}" \
    ERROR_INTERVAL_SECONDS="${ERROR_INTERVAL_SECONDS}" \
    GOOGLE_MAPS_API_KEY="${GOOGLE_MAPS_API_KEY}" \
    TRACKER_PASSWORD="${TRACKER_PASSWORD}" \
    LOCATION_TRACKER_URL="${LOCATION_TRACKER_URL}" \
    bash << 'EOF'
    set -e

    echo "🔐 Logging into ECR..."
    aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 310829530225.dkr.ecr.us-east-1.amazonaws.com

    echo ""
    echo "📥 Pulling slogan-server image..."
    docker pull 310829530225.dkr.ecr.us-east-1.amazonaws.com/slogan-server:latest

    echo ""
    echo "📥 Pulling error-generator image..."
    docker pull 310829530225.dkr.ecr.us-east-1.amazonaws.com/error-generator:latest

    echo ""
    echo "📥 Pulling location-tracker image..."
    docker pull 310829530225.dkr.ecr.us-east-1.amazonaws.com/location-tracker:latest

    echo ""
    echo "🛑 Stopping existing containers (if any)..."
    docker stop slogan-server 2>/dev/null || true
    docker rm slogan-server 2>/dev/null || true
    docker stop error-generator 2>/dev/null || true
    docker rm error-generator 2>/dev/null || true
    docker stop location-tracker 2>/dev/null || true
    docker rm location-tracker 2>/dev/null || true

    # Create Docker network if it doesn't exist
    echo ""
    echo "🌐 Setting up Docker network..."
    docker network create ec2-test-network 2>/dev/null || echo "Network already exists"

    echo ""
    echo "🚀 Starting slogan-server..."

    # Build slogan-server docker run command with optional OpenAI API key
    SLOGAN_CMD="docker run -d \
        --name slogan-server \
        --restart unless-stopped \
        --network ec2-test-network \
        -p 8080:8080"

    if [ ! -z "$OPENAI_API_KEY" ]; then
        echo "🤖 Using OpenAI API for dynamic slogan generation"
        SLOGAN_CMD="$SLOGAN_CMD -e OPENAI_API_KEY=${OPENAI_API_KEY}"
    else
        echo "⚠️  No OpenAI API key provided, using fallback slogans only"
    fi

    SLOGAN_CMD="$SLOGAN_CMD 310829530225.dkr.ecr.us-east-1.amazonaws.com/slogan-server:latest"

    eval $SLOGAN_CMD

    echo "✅ Slogan server started!"
    echo ""

    # Wait for slogan-server to be ready
    echo "⏳ Waiting for slogan-server to be ready..."
    sleep 3

    echo ""
    echo "🚀 Starting location-tracker..."

    # Build location-tracker docker run command with optional API keys
    # Expose both HTTP (8081) and HTTPS (8082) for Twilio webhook support
    TRACKER_CMD="docker run -d \
        --name location-tracker \
        --restart unless-stopped \
        --network ec2-test-network \
        -p 8081:8080 \
        -p 8082:8443 \
        -e USE_HTTPS=true"

    if [ ! -z "$GOOGLE_MAPS_API_KEY" ]; then
        echo "🗺️  Using Google Maps API for nearby business queries"
        TRACKER_CMD="$TRACKER_CMD -e GOOGLE_MAPS_API_KEY=${GOOGLE_MAPS_API_KEY}"
    else
        echo "⚠️  No Google Maps API key provided, business queries will be skipped"
    fi

    if [ ! -z "$TRACKER_PASSWORD" ]; then
        TRACKER_CMD="$TRACKER_CMD -e TRACKER_PASSWORD=${TRACKER_PASSWORD}"
    fi

    echo "🔒 Enabling HTTPS for location sharing"

    TRACKER_CMD="$TRACKER_CMD 310829530225.dkr.ecr.us-east-1.amazonaws.com/location-tracker:latest"

    eval $TRACKER_CMD

    echo "✅ Location tracker started!"
    echo ""

    # Wait for location-tracker to be ready
    echo "⏳ Waiting for location-tracker to be ready..."
    sleep 3

    echo ""
    echo "🚀 Starting error-generator..."

    # Build docker run command with optional Giphy API key
    DOCKER_CMD="docker run -d \
        --name error-generator \
        --restart unless-stopped \
        --network ec2-test-network \
        -e SLOGAN_SERVER_URL=http://slogan-server:8080 \
        -e LOCATION_TRACKER_URL=https://location-tracker:8443 \
        -e ERROR_INTERVAL_SECONDS=${ERROR_INTERVAL_SECONDS}"

    if [ ! -z "$GIPHY_API_KEY" ]; then
        echo "🔑 Using Giphy API key for real GIFs"
        DOCKER_CMD="$DOCKER_CMD -e GIPHY_API_KEY=${GIPHY_API_KEY}"
    else
        echo "⚠️  No Giphy API key provided, using placeholder GIFs"
    fi

    if [ ! -z "$SPOTIFY_CLIENT_ID" ] && [ ! -z "$SPOTIFY_CLIENT_SECRET" ] && [ ! -z "$SPOTIFY_SEED_GENRES" ]; then
        echo "🎵 Using Spotify API for song recommendations (genres: ${SPOTIFY_SEED_GENRES})"
        DOCKER_CMD="$DOCKER_CMD -e SPOTIFY_CLIENT_ID=${SPOTIFY_CLIENT_ID}"
        DOCKER_CMD="$DOCKER_CMD -e SPOTIFY_CLIENT_SECRET=${SPOTIFY_CLIENT_SECRET}"
        DOCKER_CMD="$DOCKER_CMD -e SPOTIFY_SEED_GENRES=${SPOTIFY_SEED_GENRES}"
    else
        echo "⚠️  No Spotify credentials provided, using placeholder songs"
    fi

    DOCKER_CMD="$DOCKER_CMD 310829530225.dkr.ecr.us-east-1.amazonaws.com/error-generator:latest"

    eval $DOCKER_CMD

    echo "✅ Error generator started!"
    echo ""

    echo "📊 Container status:"
    docker ps --filter name=slogan-server --filter name=error-generator --filter name=location-tracker

    echo ""
    echo "📝 Recent logs from slogan-server:"
    docker logs --tail 10 slogan-server

    echo ""
    echo "📝 Recent logs from location-tracker:"
    docker logs --tail 10 location-tracker

    echo ""
    echo "📝 Recent logs from error-generator:"
    docker logs --tail 10 error-generator

    echo ""
    echo "🔍 Validating deployment..."

    # Validate all containers are running (not restarting)
    LOCATION_STATUS=$(docker inspect --format='{{.State.Status}}' location-tracker 2>/dev/null)
    ERROR_GEN_STATUS=$(docker inspect --format='{{.State.Status}}' error-generator 2>/dev/null)
    SLOGAN_STATUS=$(docker inspect --format='{{.State.Status}}' slogan-server 2>/dev/null)

    if [ "$LOCATION_STATUS" != "running" ]; then
        echo "⚠️  WARNING: location-tracker is not running (status: $LOCATION_STATUS)"
        echo "   Check logs with: docker logs location-tracker"
    else
        echo "✅ location-tracker is running"
    fi

    if [ "$ERROR_GEN_STATUS" != "running" ]; then
        echo "⚠️  WARNING: error-generator is not running (status: $ERROR_GEN_STATUS)"
        echo "   Check logs with: docker logs error-generator"
    else
        echo "✅ error-generator is running"
    fi

    if [ "$SLOGAN_STATUS" != "running" ]; then
        echo "⚠️  WARNING: slogan-server is not running (status: $SLOGAN_STATUS)"
        echo "   Check logs with: docker logs slogan-server"
    else
        echo "✅ slogan-server is running"
    fi

    # Check if location-tracker loaded DynamoDB data
    if docker logs location-tracker 2>&1 | grep -q "Loaded.*from DynamoDB"; then
        echo "✅ DynamoDB data loaded successfully"
    else
        echo "⚠️  WARNING: Could not verify DynamoDB data loading"
    fi

    # Check if required environment variables are set
    if ! docker exec location-tracker printenv TRACKER_PASSWORD >/dev/null 2>&1; then
        echo "❌ ERROR: TRACKER_PASSWORD not set in location-tracker!"
    else
        echo "✅ TRACKER_PASSWORD is configured"
    fi
EOF

echo ""
echo "✅ Deployment complete!"
echo ""
echo "🌐 Service URLs:"
echo "   Slogan server:      http://${PUBLIC_DNS}:8080"
echo "   Location tracker:   https://${PUBLIC_DNS}:8082 (HTTPS with self-signed cert)"
echo "   Location tracker:   http://${PUBLIC_DNS}:8081 (HTTP for Twilio webhooks)"
echo ""
echo "📱 Twilio SMS Webhook URL:"
echo "   http://${PUBLIC_DNS}:8081/api/twilio/sms"
echo ""
echo "📊 To view logs:"
echo "  ssh -i ${EC2_KEY_PATH} ${EC2_USER}@${PUBLIC_DNS} 'docker logs -f slogan-server'"
echo "  ssh -i ${EC2_KEY_PATH} ${EC2_USER}@${PUBLIC_DNS} 'docker logs -f location-tracker'"
echo "  ssh -i ${EC2_KEY_PATH} ${EC2_USER}@${PUBLIC_DNS} 'docker logs -f error-generator'"
echo ""
echo "📖 Troubleshooting: See DEPLOYMENT_TROUBLESHOOTING.md"
