#!/bin/bash

# Deploy both applications to EC2 using AWS Systems Manager Session Manager
# No SSH keys or open port 22 required!
set -e

# Load environment variables from .env.ec2 if it exists
if [ -f .env.ec2 ]; then
    echo "📋 Loading environment variables from .env.ec2..."
    set -a
    source .env.ec2
    set +a
fi

AWS_REGION=us-east-1
AWS_ACCOUNT_ID=310829530225
INSTANCE_ID=i-04bd2369c252bee39

# Set defaults if not provided
GIPHY_API_KEY=${GIPHY_API_KEY:-}
PEXELS_API_KEY=${PEXELS_API_KEY:-}
OPENAI_API_KEY=${OPENAI_API_KEY:-}
SPOTIFY_CLIENT_ID=${SPOTIFY_CLIENT_ID:-}
SPOTIFY_CLIENT_SECRET=${SPOTIFY_CLIENT_SECRET:-}
SPOTIFY_SEED_GENRES=${SPOTIFY_SEED_GENRES:-}
ERROR_INTERVAL_SECONDS=${ERROR_INTERVAL_SECONDS:-60}
GOOGLE_MAPS_API_KEY=${GOOGLE_MAPS_API_KEY:-}
PERPLEXITY_API_KEY=${PERPLEXITY_API_KEY:-}
TRACKER_PASSWORD=${TRACKER_PASSWORD:-}
LOCATION_TRACKER_URL=${LOCATION_TRACKER_URL:-}
DEEPSEEK_API_KEY=${DEEPSEEK_API_KEY:-}
ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY:-}
GEMINI_API_KEY=${GEMINI_API_KEY:-}
S3_MEME_BUCKET=${S3_MEME_BUCKET:-error-generator-memes}
GCP_PROJECT_ID=${GCP_PROJECT_ID:-notspies}
GCP_LOCATION=${GCP_LOCATION:-us-central1}

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

echo "📦 Deploying to instance: ${INSTANCE_ID}"
echo "   Public DNS: ${PUBLIC_DNS}"
echo ""

# Check if instance is online in SSM
echo "🔍 Checking Systems Manager connectivity..."
PING_STATUS=$(aws ssm describe-instance-information \
    --filters "Key=InstanceIds,Values=${INSTANCE_ID}" \
    --region ${AWS_REGION} \
    --query 'InstanceInformationList[0].PingStatus' \
    --output text 2>/dev/null || echo "Unknown")

if [ "$PING_STATUS" != "Online" ]; then
    echo "❌ ERROR: Instance is not online in Systems Manager (Status: ${PING_STATUS})"
    echo ""
    echo "Please ensure:"
    echo "  1. SSM Agent is installed and running on the instance"
    echo "  2. Instance IAM role has AmazonSSMManagedInstanceCore policy attached"
    echo "  3. Instance has internet connectivity"
    echo ""
    exit 1
fi

echo "✅ Instance is online in Systems Manager"
echo ""

# Create deployment script with environment variables
echo "📝 Creating deployment script..."
DEPLOY_SCRIPT=$(cat <<'DEPLOY_EOF'
set -e

export GIPHY_API_KEY="GIPHY_API_KEY_VALUE"
export PEXELS_API_KEY="PEXELS_API_KEY_VALUE"
export OPENAI_API_KEY="OPENAI_API_KEY_VALUE"
export SPOTIFY_CLIENT_ID="SPOTIFY_CLIENT_ID_VALUE"
export SPOTIFY_CLIENT_SECRET="SPOTIFY_CLIENT_SECRET_VALUE"
export SPOTIFY_SEED_GENRES="SPOTIFY_SEED_GENRES_VALUE"
export ERROR_INTERVAL_SECONDS="ERROR_INTERVAL_SECONDS_VALUE"
export GOOGLE_MAPS_API_KEY="GOOGLE_MAPS_API_KEY_VALUE"
export PERPLEXITY_API_KEY="PERPLEXITY_API_KEY_VALUE"
export TRACKER_PASSWORD="TRACKER_PASSWORD_VALUE"
export LOCATION_TRACKER_URL="LOCATION_TRACKER_URL_VALUE"
export DEEPSEEK_API_KEY="DEEPSEEK_API_KEY_VALUE"
export ANTHROPIC_API_KEY="ANTHROPIC_API_KEY_VALUE"
export GEMINI_API_KEY="GEMINI_API_KEY_VALUE"
export S3_MEME_BUCKET="S3_MEME_BUCKET_VALUE"
export GCP_PROJECT_ID="GCP_PROJECT_ID_VALUE"
export GCP_LOCATION="GCP_LOCATION_VALUE"
export STRIPE_SECRET_KEY="STRIPE_SECRET_KEY_VALUE"
export STRIPE_PUBLISHABLE_KEY="STRIPE_PUBLISHABLE_KEY_VALUE"
DEPLOY_EOF
)

# Replace placeholders with actual values
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//GIPHY_API_KEY_VALUE/$GIPHY_API_KEY}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//PEXELS_API_KEY_VALUE/$PEXELS_API_KEY}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//OPENAI_API_KEY_VALUE/$OPENAI_API_KEY}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//SPOTIFY_CLIENT_ID_VALUE/$SPOTIFY_CLIENT_ID}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//SPOTIFY_CLIENT_SECRET_VALUE/$SPOTIFY_CLIENT_SECRET}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//SPOTIFY_SEED_GENRES_VALUE/$SPOTIFY_SEED_GENRES}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//ERROR_INTERVAL_SECONDS_VALUE/$ERROR_INTERVAL_SECONDS}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//GOOGLE_MAPS_API_KEY_VALUE/$GOOGLE_MAPS_API_KEY}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//PERPLEXITY_API_KEY_VALUE/$PERPLEXITY_API_KEY}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//TRACKER_PASSWORD_VALUE/$TRACKER_PASSWORD}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//LOCATION_TRACKER_URL_VALUE/$LOCATION_TRACKER_URL}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//DEEPSEEK_API_KEY_VALUE/$DEEPSEEK_API_KEY}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//ANTHROPIC_API_KEY_VALUE/$ANTHROPIC_API_KEY}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//GEMINI_API_KEY_VALUE/$GEMINI_API_KEY}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//S3_MEME_BUCKET_VALUE/$S3_MEME_BUCKET}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//GCP_PROJECT_ID_VALUE/$GCP_PROJECT_ID}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//GCP_LOCATION_VALUE/$GCP_LOCATION}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//STRIPE_SECRET_KEY_VALUE/$STRIPE_SECRET_KEY}"
DEPLOY_SCRIPT="${DEPLOY_SCRIPT//STRIPE_PUBLISHABLE_KEY_VALUE/$STRIPE_PUBLISHABLE_KEY}"

# Append the actual deployment commands
DEPLOY_SCRIPT+=$(cat <<'EOF'

bash << 'INNER_EOF'
    set -e

    echo "🔐 Logging into ECR..."
    aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 310829530225.dkr.ecr.us-east-1.amazonaws.com

    echo ""
    echo "🗑️  Removing cached images to force fresh pull..."
    docker rmi -f 310829530225.dkr.ecr.us-east-1.amazonaws.com/slogan-server:latest 2>/dev/null || true
    docker rmi -f 310829530225.dkr.ecr.us-east-1.amazonaws.com/error-generator:latest 2>/dev/null || true
    docker rmi -f 310829530225.dkr.ecr.us-east-1.amazonaws.com/location-tracker:latest 2>/dev/null || true
    docker rmi -f 310829530225.dkr.ecr.us-east-1.amazonaws.com/code-fix-generator:latest 2>/dev/null || true
    docker rmi -f 310829530225.dkr.ecr.us-east-1.amazonaws.com/nginx-proxy:latest 2>/dev/null || true

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
    echo "📥 Pulling code-fix-generator image..."
    docker pull 310829530225.dkr.ecr.us-east-1.amazonaws.com/code-fix-generator:latest

    echo ""
    echo "📥 Pulling nginx-proxy image..."
    docker pull 310829530225.dkr.ecr.us-east-1.amazonaws.com/nginx-proxy:latest

    echo ""
    echo "🛑 Stopping existing containers (if any)..."
    docker stop nginx-proxy 2>/dev/null || true
    docker rm nginx-proxy 2>/dev/null || true
    docker stop slogan-server 2>/dev/null || true
    docker rm slogan-server 2>/dev/null || true
    docker stop error-generator 2>/dev/null || true
    docker rm error-generator 2>/dev/null || true
    docker stop location-tracker 2>/dev/null || true
    docker rm location-tracker 2>/dev/null || true
    docker stop code-fix-generator 2>/dev/null || true
    docker rm code-fix-generator 2>/dev/null || true

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

    if [ ! -z "$PERPLEXITY_API_KEY" ]; then
        echo "🔍 Using Perplexity API for governing body searches"
        TRACKER_CMD="$TRACKER_CMD -e PERPLEXITY_API_KEY=${PERPLEXITY_API_KEY}"
    else
        echo "⚠️  No Perplexity API key provided, governing body searches will be skipped"
    fi

    if [ ! -z "$TRACKER_PASSWORD" ]; then
        TRACKER_CMD="$TRACKER_CMD -e TRACKER_PASSWORD=${TRACKER_PASSWORD}"
    fi

    if [ ! -z "$STRIPE_SECRET_KEY" ]; then
        echo "💳 Using Stripe API for donation feature"
        TRACKER_CMD="$TRACKER_CMD -e STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}"
        TRACKER_CMD="$TRACKER_CMD -e STRIPE_PUBLISHABLE_KEY=${STRIPE_PUBLISHABLE_KEY}"
    else
        echo "⚠️  No Stripe API keys provided, donation feature will be disabled"
    fi

    if [ ! -z "$OPENAI_API_KEY" ]; then
        echo "🧠 Using OpenAI API for Rorschach interpretations"
        TRACKER_CMD="$TRACKER_CMD -e OPENAI_API_KEY=${OPENAI_API_KEY}"
    else
        echo "⚠️  No OpenAI API key provided, Rorschach interpretations will be disabled"
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
    echo "🚀 Starting code-fix-generator..."

    # Build code-fix-generator docker run command with optional DeepSeek API key
    FIX_GEN_CMD="docker run -d \
        --name code-fix-generator \
        --restart unless-stopped \
        --network ec2-test-network \
        -p 7070:7070"

    if [ ! -z "$DEEPSEEK_API_KEY" ]; then
        echo "🤖 Using DeepSeek API for AI-generated satirical fixes"
        FIX_GEN_CMD="$FIX_GEN_CMD -e DEEPSEEK_API_KEY=${DEEPSEEK_API_KEY}"
    else
        echo "⚠️  No DeepSeek API key provided, using fallback satirical fixes"
    fi

    FIX_GEN_CMD="$FIX_GEN_CMD 310829530225.dkr.ecr.us-east-1.amazonaws.com/code-fix-generator:latest"

    eval $FIX_GEN_CMD

    echo "✅ Code fix generator started!"
    echo ""

    # Wait for code-fix-generator to be ready
    echo "⏳ Waiting for code-fix-generator to be ready..."
    sleep 2

    echo ""
    echo "🚀 Starting error-generator..."

    # Build docker run command with optional Giphy API key
    DOCKER_CMD="docker run -d \
        --name error-generator \
        --restart unless-stopped \
        --network ec2-test-network \
        -e SLOGAN_SERVER_URL=http://slogan-server:8080 \
        -e LOCATION_TRACKER_URL=https://location-tracker:8443 \
        -e FIX_GENERATOR_URL=http://code-fix-generator:7070 \
        -e ERROR_INTERVAL_SECONDS=${ERROR_INTERVAL_SECONDS}"

    if [ ! -z "$GIPHY_API_KEY" ]; then
        echo "🔑 Using Giphy API key for real GIFs"
        DOCKER_CMD="$DOCKER_CMD -e GIPHY_API_KEY=${GIPHY_API_KEY}"
    else
        echo "⚠️  No Giphy API key provided, using placeholder GIFs"
    fi

    if [ ! -z "$PEXELS_API_KEY" ]; then
        echo "🍽️  Using Pexels API key for food blog images"
        DOCKER_CMD="$DOCKER_CMD -e PEXELS_API_KEY=${PEXELS_API_KEY}"
    else
        echo "⚠️  No Pexels API key provided, using placeholder food images"
    fi

    if [ ! -z "$SPOTIFY_CLIENT_ID" ] && [ ! -z "$SPOTIFY_CLIENT_SECRET" ] && [ ! -z "$SPOTIFY_SEED_GENRES" ]; then
        echo "🎵 Using Spotify API for song recommendations (genres: ${SPOTIFY_SEED_GENRES})"
        DOCKER_CMD="$DOCKER_CMD -e SPOTIFY_CLIENT_ID=${SPOTIFY_CLIENT_ID}"
        DOCKER_CMD="$DOCKER_CMD -e SPOTIFY_CLIENT_SECRET=${SPOTIFY_CLIENT_SECRET}"
        DOCKER_CMD="$DOCKER_CMD -e SPOTIFY_SEED_GENRES=${SPOTIFY_SEED_GENRES}"
    else
        echo "⚠️  No Spotify credentials provided, using placeholder songs"
    fi

    if [ ! -z "$ANTHROPIC_API_KEY" ]; then
        echo "📚 Using Anthropic API for children's story generation"
        DOCKER_CMD="$DOCKER_CMD -e ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}"
    else
        echo "⚠️  No Anthropic API key provided, children's stories will not be generated"
    fi

    if [ ! -z "$GEMINI_API_KEY" ]; then
        echo "🎨 Using Vertex AI Imagen for absurdist meme generation"
        DOCKER_CMD="$DOCKER_CMD -e GEMINI_API_KEY=${GEMINI_API_KEY}"
        DOCKER_CMD="$DOCKER_CMD -e S3_MEME_BUCKET=${S3_MEME_BUCKET:-error-generator-memes}"
        DOCKER_CMD="$DOCKER_CMD -e AWS_REGION=${AWS_REGION:-us-east-1}"
        DOCKER_CMD="$DOCKER_CMD -e GCP_PROJECT_ID=${GCP_PROJECT_ID:-notspies}"
        DOCKER_CMD="$DOCKER_CMD -e GCP_LOCATION=${GCP_LOCATION:-us-central1}"
    else
        echo "⚠️  No Gemini API key provided, memes will not be generated"
    fi

    DOCKER_CMD="$DOCKER_CMD 310829530225.dkr.ecr.us-east-1.amazonaws.com/error-generator:latest"

    eval $DOCKER_CMD

    echo "✅ Error generator started!"
    echo ""

    echo "🚀 Starting nginx reverse proxy..."
    docker run -d \
        --name nginx-proxy \
        --restart unless-stopped \
        --network ec2-test-network \
        -p 80:80 \
        -p 443:443 \
        -v /var/www/static:/var/www/static:ro \
        310829530225.dkr.ecr.us-east-1.amazonaws.com/nginx-proxy:latest

    echo "✅ Nginx proxy started!"
    echo ""

    echo "📊 Container status:"
    docker ps --filter name=nginx-proxy --filter name=slogan-server --filter name=error-generator --filter name=location-tracker --filter name=code-fix-generator

    echo ""
    echo "📝 Recent logs from slogan-server:"
    docker logs --tail 10 slogan-server

    echo ""
    echo "📝 Recent logs from location-tracker:"
    docker logs --tail 10 location-tracker

    echo ""
    echo "📝 Recent logs from code-fix-generator:"
    docker logs --tail 10 code-fix-generator

    echo ""
    echo "📝 Recent logs from error-generator:"
    docker logs --tail 10 error-generator

    echo ""
    echo "📝 Recent logs from nginx-proxy:"
    docker logs --tail 10 nginx-proxy

    echo ""
    echo "🔍 Validating deployment..."

    # Validate all containers are running (not restarting)
    NGINX_STATUS=$(docker inspect --format='{{.State.Status}}' nginx-proxy 2>/dev/null)
    LOCATION_STATUS=$(docker inspect --format='{{.State.Status}}' location-tracker 2>/dev/null)
    ERROR_GEN_STATUS=$(docker inspect --format='{{.State.Status}}' error-generator 2>/dev/null)
    SLOGAN_STATUS=$(docker inspect --format='{{.State.Status}}' slogan-server 2>/dev/null)
    FIX_GEN_STATUS=$(docker inspect --format='{{.State.Status}}' code-fix-generator 2>/dev/null)

    if [ "$NGINX_STATUS" != "running" ]; then
        echo "⚠️  WARNING: nginx-proxy is not running (status: $NGINX_STATUS)"
        echo "   Check logs with: docker logs nginx-proxy"
    else
        echo "✅ nginx-proxy is running"
    fi

    if [ "$LOCATION_STATUS" != "running" ]; then
        echo "⚠️  WARNING: location-tracker is not running (status: $LOCATION_STATUS)"
        echo "   Check logs with: docker logs location-tracker"
    else
        echo "✅ location-tracker is running"
    fi

    if [ "$FIX_GEN_STATUS" != "running" ]; then
        echo "⚠️  WARNING: code-fix-generator is not running (status: $FIX_GEN_STATUS)"
        echo "   Check logs with: docker logs code-fix-generator"
    else
        echo "✅ code-fix-generator is running"
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
INNER_EOF
EOF
)

# Execute deployment via SSM Send-Command
echo "🚀 Executing deployment via Systems Manager..."

# Base64 encode the script to avoid JSON escaping issues
ENCODED_SCRIPT=$(echo "${DEPLOY_SCRIPT}" | base64)

# Step 1: Write the deployment script to the EC2 instance
echo "📝 Uploading deployment script to EC2..."
UPLOAD_CMD_ID=$(aws ssm send-command \
    --instance-ids ${INSTANCE_ID} \
    --region ${AWS_REGION} \
    --document-name "AWS-RunShellScript" \
    --comment "Upload deployment script" \
    --parameters "commands=[\"echo '${ENCODED_SCRIPT}' | base64 -d > /tmp/deploy-apps.sh\",\"chmod +x /tmp/deploy-apps.sh\"]" \
    --query 'Command.CommandId' \
    --output text)

# Wait for upload to complete
aws ssm wait command-executed \
    --command-id ${UPLOAD_CMD_ID} \
    --instance-id ${INSTANCE_ID} \
    --region ${AWS_REGION} 2>/dev/null || true

# Step 2: Execute the deployment script
echo "🚀 Executing deployment script..."
COMMAND_ID=$(aws ssm send-command \
    --instance-ids ${INSTANCE_ID} \
    --region ${AWS_REGION} \
    --document-name "AWS-RunShellScript" \
    --comment "EC2 Test Apps Deployment" \
    --parameters 'commands=["bash /tmp/deploy-apps.sh"]' \
    --output-s3-bucket-name "aws-ssm-session-logs-${AWS_ACCOUNT_ID}-${AWS_REGION}" 2>/dev/null \
    --query 'Command.CommandId' \
    --output text || aws ssm send-command \
    --instance-ids ${INSTANCE_ID} \
    --region ${AWS_REGION} \
    --document-name "AWS-RunShellScript" \
    --comment "EC2 Test Apps Deployment" \
    --parameters 'commands=["bash /tmp/deploy-apps.sh"]' \
    --query 'Command.CommandId' \
    --output text)

echo "   Command ID: ${COMMAND_ID}"
echo ""

# Wait for command to complete
echo "⏳ Waiting for deployment to complete..."
aws ssm wait command-executed \
    --command-id ${COMMAND_ID} \
    --instance-id ${INSTANCE_ID} \
    --region ${AWS_REGION} 2>/dev/null || true

# Get command output
echo ""
echo "📋 Deployment output:"
aws ssm get-command-invocation \
    --command-id ${COMMAND_ID} \
    --instance-id ${INSTANCE_ID} \
    --region ${AWS_REGION} \
    --query 'StandardOutputContent' \
    --output text

# Check for errors
STATUS=$(aws ssm get-command-invocation \
    --command-id ${COMMAND_ID} \
    --instance-id ${INSTANCE_ID} \
    --region ${AWS_REGION} \
    --query 'Status' \
    --output text)

if [ "$STATUS" != "Success" ]; then
    echo ""
    echo "❌ Deployment failed with status: ${STATUS}"
    echo ""
    echo "Error output:"
    aws ssm get-command-invocation \
        --command-id ${COMMAND_ID} \
        --instance-id ${INSTANCE_ID} \
        --region ${AWS_REGION} \
        --query 'StandardErrorContent' \
        --output text
    exit 1
fi

echo ""
echo "✅ Deployment complete!"
echo ""
echo "🌐 Service URLs:"
echo "   🌍 Main site (via Nginx): https://notspies.org"
echo "   🌍 Main site (via Nginx): http://notspies.org (redirects to HTTPS)"
echo ""
echo "   Direct service access (for debugging):"
echo "   - Slogan server:      http://${PUBLIC_DNS}:8080"
echo "   - Location tracker:   https://${PUBLIC_DNS}:8082 (HTTPS with self-signed cert)"
echo "   - Location tracker:   http://${PUBLIC_DNS}:8081 (HTTP for Twilio webhooks)"
echo "   - Fix generator:      http://${PUBLIC_DNS}:7070 (Satirical code fixes)"
echo ""
echo "📱 Twilio SMS Webhook URL:"
echo "   http://${PUBLIC_DNS}:8081/api/twilio/sms"
echo ""
echo "📊 To view logs using Session Manager:"
echo "  ./ssm-connect.sh  # Then run: docker logs -f <container-name>"
echo ""
echo "  Or use SSM send-command:"
echo "  aws ssm send-command --instance-ids ${INSTANCE_ID} --document-name 'AWS-RunShellScript' \\"
echo "    --parameters 'commands=[\"docker logs --tail 50 nginx-proxy\"]' \\"
echo "    --query 'Command.CommandId' --output text"
echo ""
echo "🔐 Security Improvements:"
echo "   ✅ No SSH keys required (using AWS Systems Manager)"
echo "   ✅ Port 22 not exposed to internet"
echo "   ✅ IAM-based access control"
echo "   ✅ Session logging via CloudTrail"
echo ""
echo "🤖 Satirical Fix Generator:"
echo "   All errors are now automatically enhanced with AI-generated satirical code fixes"
echo "   Fixes are stored in DynamoDB and displayed in the location tracker UI"
echo ""
echo "🌐 Nginx Reverse Proxy:"
echo "   All traffic to notspies.org is now routed through Nginx"
echo "   HTTP requests are automatically redirected to HTTPS"
echo "   SSL/TLS termination handled by Nginx with self-signed certificates"
echo ""
echo "📖 Troubleshooting: See DEPLOYMENT_TROUBLESHOOTING.md"
