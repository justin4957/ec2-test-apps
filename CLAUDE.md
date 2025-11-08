# Claude Code Instructions for EC2 Test Apps

## Deployment Process

**IMPORTANT: Always use the deployment script first!**

When deploying any changes to EC2:

1. **First, always try the deploy script:**
   ```bash
   cd /Users/coolbeans/Development/dev/ec2-test-apps
   ./deploy-to-ec2.sh
   ```

2. **Only if the deploy script fails**, then try alternative methods like:
   - Direct SSH access
   - AWS SSM commands
   - Manual docker commands

The `deploy-to-ec2.sh` script handles:
- ECR login
- Pulling latest images
- Stopping and removing old containers
- Starting new containers with proper configuration
- Network setup
- Environment variable management
- Health checks and validation

## Building Containers

When building containers before deployment:

1. **Build location-tracker:**
   ```bash
   cd location-tracker
   docker buildx build --platform linux/amd64 -t 310829530225.dkr.ecr.us-east-1.amazonaws.com/location-tracker:latest --push .
   ```

2. **Build error-generator:**
   ```bash
   cd error-generator
   docker buildx build --platform linux/amd64 -t 310829530225.dkr.ecr.us-east-1.amazonaws.com/error-generator:latest --push .
   ```

3. **Build slogan-server:**
   ```bash
   cd slogan-server
   docker buildx build --platform linux/amd64 -t 310829530225.dkr.ecr.us-east-1.amazonaws.com/slogan-server:latest --push .
   ```

After building, always run `./deploy-to-ec2.sh` to deploy the changes.

## Project Structure

- `location-tracker/` - Go application for location tracking with Google Maps integration
- `error-generator/` - Go application that generates error logs with GIFs, songs, and stories
- `slogan-server/` - Go application for generating dynamic slogans
- `code-fix-generator/` - Python application for satirical code fixes using DeepSeek
- `nginx/` - Nginx reverse proxy configuration
- `deploy-to-ec2.sh` - Main deployment script (USE THIS FIRST!)
- `.env.ec2` - Environment variables for EC2 deployment

## Key Features

- Location tracking with simulated location support for authenticated users
- Google Maps autocomplete for location simulation
- Commercial real estate search with Perplexity API
- Governing bodies display in UI (city council, planning, zoning, civic orgs)
- Interactive fiction stories with Anthropic Claude
- Daily cryptogram puzzles
- SMS integration via Twilio
- Real-time error logs with GIFs and music
