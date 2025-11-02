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
SPOTIFY_PLAYLIST_ID=${SPOTIFY_PLAYLIST_ID:-}
ERROR_INTERVAL_SECONDS=${ERROR_INTERVAL_SECONDS:-60}

# Get instance details
echo "🔍 Getting instance details..."
PUBLIC_DNS=$(aws ec2 describe-instances \
    --instance-ids ${INSTANCE_ID} \
    --region ${AWS_REGION} \
    --query 'Reservations[0].Instances[0].PublicDnsName' \
    --output text)

echo "📦 Deploying to: ${PUBLIC_DNS}"
echo ""

# Deploy both containers via SSH
ssh -o StrictHostKeyChecking=no -i ${EC2_KEY_PATH} ${EC2_USER}@${PUBLIC_DNS} \
    GIPHY_API_KEY="${GIPHY_API_KEY}" \
    OPENAI_API_KEY="${OPENAI_API_KEY}" \
    SPOTIFY_CLIENT_ID="${SPOTIFY_CLIENT_ID}" \
    SPOTIFY_CLIENT_SECRET="${SPOTIFY_CLIENT_SECRET}" \
    SPOTIFY_PLAYLIST_ID="${SPOTIFY_PLAYLIST_ID}" \
    ERROR_INTERVAL_SECONDS="${ERROR_INTERVAL_SECONDS}" \
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
    echo "🛑 Stopping existing containers (if any)..."
    docker stop slogan-server 2>/dev/null || true
    docker rm slogan-server 2>/dev/null || true
    docker stop error-generator 2>/dev/null || true
    docker rm error-generator 2>/dev/null || true

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
    echo "🚀 Starting error-generator..."

    # Build docker run command with optional Giphy API key
    DOCKER_CMD="docker run -d \
        --name error-generator \
        --restart unless-stopped \
        --network ec2-test-network \
        -e SLOGAN_SERVER_URL=http://slogan-server:8080 \
        -e ERROR_INTERVAL_SECONDS=${ERROR_INTERVAL_SECONDS}"

    if [ ! -z "$GIPHY_API_KEY" ]; then
        echo "🔑 Using Giphy API key for real GIFs"
        DOCKER_CMD="$DOCKER_CMD -e GIPHY_API_KEY=${GIPHY_API_KEY}"
    else
        echo "⚠️  No Giphy API key provided, using placeholder GIFs"
    fi

    if [ ! -z "$SPOTIFY_CLIENT_ID" ] && [ ! -z "$SPOTIFY_CLIENT_SECRET" ] && [ ! -z "$SPOTIFY_PLAYLIST_ID" ]; then
        echo "🎵 Using Spotify API for real songs from playlist"
        DOCKER_CMD="$DOCKER_CMD -e SPOTIFY_CLIENT_ID=${SPOTIFY_CLIENT_ID}"
        DOCKER_CMD="$DOCKER_CMD -e SPOTIFY_CLIENT_SECRET=${SPOTIFY_CLIENT_SECRET}"
        DOCKER_CMD="$DOCKER_CMD -e SPOTIFY_PLAYLIST_ID=${SPOTIFY_PLAYLIST_ID}"
    else
        echo "⚠️  No Spotify credentials provided, using placeholder songs"
    fi

    DOCKER_CMD="$DOCKER_CMD 310829530225.dkr.ecr.us-east-1.amazonaws.com/error-generator:latest"

    eval $DOCKER_CMD

    echo "✅ Error generator started!"
    echo ""

    echo "📊 Container status:"
    docker ps --filter name=slogan-server --filter name=error-generator

    echo ""
    echo "📝 Recent logs from slogan-server:"
    docker logs --tail 10 slogan-server

    echo ""
    echo "📝 Recent logs from error-generator:"
    docker logs --tail 10 error-generator
EOF

echo ""
echo "✅ Deployment complete!"
echo "🌐 Slogan server is available at: http://${PUBLIC_DNS}:8080"
echo ""
echo "To view logs:"
echo "  ssh -i ${EC2_KEY_PATH} ${EC2_USER}@${PUBLIC_DNS} 'docker logs -f slogan-server'"
echo "  ssh -i ${EC2_KEY_PATH} ${EC2_USER}@${PUBLIC_DNS} 'docker logs -f error-generator'"
