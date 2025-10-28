# EC2 Test Applications

Two minimal Go applications for testing EC2 deployment functionality with Docker.

## Overview

### Slogan Server
HTTP server that receives error log messages and responds with:
- A cigarette emoji (🚬)
- A random slogan from 114 nonsensical advertising slogans

### Error Generator
Client application that:
- Batch loads GIF URLs from Giphy API (to avoid rate limiting)
- Every minute (configurable) generates a random error log message
- Retrieves a random GIF URL from the cached batch
- Sends the error message unencrypted to the slogan server
- Displays the response

## Architecture

```
┌─────────────────┐         HTTP POST          ┌─────────────────┐
│ Error Generator │ ──────────────────────────> │  Slogan Server  │
│                 │                              │                 │
│ - Giphy cache   │   {"message": "...",        │ - 114 slogans   │
│ - Timer (60s)   │    "gif_url": "..."}        │ - Random picker │
│                 │                              │                 │
│                 │ <────────────────────────── │                 │
│                 │   {"emoji": "🚬",           │                 │
│                 │    "slogan": "..."}         │                 │
└─────────────────┘                              └─────────────────┘
```

## Local Testing

### Prerequisites
- Docker
- Docker Compose
- (Optional) Giphy API key for real GIF URLs

### Quick Start

1. Clone or navigate to the project directory:
```bash
cd ec2-test-apps
```

2. (Optional) Set up Giphy API key:
```bash
cp .env.example .env
# Edit .env and add your GIPHY_API_KEY
```

3. Build and run with Docker Compose:
```bash
docker-compose up --build
```

4. Watch the logs to see error generation and slogan responses every minute.

### Configuration

Environment variables for `error-generator`:

| Variable | Description | Default |
|----------|-------------|---------|
| `SLOGAN_SERVER_URL` | URL of the slogan server | `http://localhost:8080` |
| `ERROR_INTERVAL_SECONDS` | Seconds between error logs | `60` |
| `GIPHY_API_KEY` | Giphy API key (optional) | Placeholder URLs if not set |

### Testing Individual Services

Build and run slogan-server:
```bash
cd slogan-server
docker build -t slogan-server .
docker run -p 8080:8080 slogan-server
```

Build and run error-generator:
```bash
cd error-generator
docker build -t error-generator .
docker run -e SLOGAN_SERVER_URL=http://host.docker.internal:8080 error-generator
```

Test slogan-server manually:
```bash
curl -X POST http://localhost:8080/error-log \
  -H "Content-Type: application/json" \
  -d '{"message": "NullPointerException", "gif_url": "https://giphy.com/gifs/test"}'
```

## EC2 Deployment with aws-docker-tools

### Prerequisites
- AWS CLI configured with appropriate credentials
- ECR repositories created
- EC2 instances provisioned
- aws-docker-tools scripts available

### Step 1: Create ECR Repositories

```bash
cd ../aws-docker-tools
./ecr-create-repo.sh slogan-server
./ecr-create-repo.sh error-generator
```

### Step 2: Build and Push Docker Images

For slogan-server:
```bash
cd ../ec2-test-apps/slogan-server

# Build for AMD64 (EC2 standard instances)
docker buildx build --platform linux/amd64 -t slogan-server .

# Get ECR login
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com

# Tag and push
docker tag slogan-server:latest <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/slogan-server:latest
docker push <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/slogan-server:latest
```

For error-generator:
```bash
cd ../error-generator

docker buildx build --platform linux/amd64 -t error-generator .
docker tag error-generator:latest <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/error-generator:latest
docker push <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/error-generator:latest
```

### Step 3: Deploy to EC2

#### Deploy Slogan Server (Server EC2 Instance)

SSH into your server EC2 instance:
```bash
ssh -i your-key.pem ec2-user@<SERVER_EC2_PUBLIC_IP>
```

Pull and run the container:
```bash
# Login to ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com

# Pull image
docker pull <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/slogan-server:latest

# Run container
docker run -d \
  --name slogan-server \
  -p 8080:8080 \
  --restart unless-stopped \
  <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/slogan-server:latest

# Check logs
docker logs -f slogan-server
```

#### Deploy Error Generator (Client EC2 Instance)

SSH into your client EC2 instance:
```bash
ssh -i your-key.pem ec2-user@<CLIENT_EC2_PUBLIC_IP>
```

Pull and run the container:
```bash
# Login to ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com

# Pull image
docker pull <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/error-generator:latest

# Run container (replace SERVER_PRIVATE_IP with the slogan-server EC2 private IP)
docker run -d \
  --name error-generator \
  -e SLOGAN_SERVER_URL=http://<SERVER_PRIVATE_IP>:8080 \
  -e ERROR_INTERVAL_SECONDS=60 \
  -e GIPHY_API_KEY=<YOUR_GIPHY_KEY> \
  --restart unless-stopped \
  <AWS_ACCOUNT_ID>.dkr.ecr.us-east-1.amazonaws.com/error-generator:latest

# Check logs
docker logs -f error-generator
```

### Step 4: Verify Deployment

Check slogan-server health:
```bash
curl http://<SERVER_EC2_IP>:8080/health
```

Monitor error-generator logs:
```bash
ssh ec2-user@<CLIENT_EC2_IP> "docker logs -f error-generator"
```

Monitor slogan-server logs:
```bash
ssh ec2-user@<SERVER_EC2_IP> "docker logs -f slogan-server"
```

### Using ec2-status.sh

If you have the `ec2-status.sh` script from aws-docker-tools:
```bash
cd ../aws-docker-tools
./ec2-status.sh
```

This will show you the status of your EC2 instances and help identify the server IPs.

## Security Notes

- **Communication is unencrypted (HTTP)** - This is intentional for testing purposes
- In production, use HTTPS and proper authentication
- Ensure EC2 security groups allow:
  - Port 8080 from error-generator to slogan-server
  - SSH access (port 22) for deployment
- Consider using VPC private subnets for internal communication

## Monitoring

View live logs in docker-compose:
```bash
docker-compose logs -f
```

View individual service logs:
```bash
docker-compose logs -f slogan-server
docker-compose logs -f error-generator
```

## Troubleshooting

### Error generator can't reach slogan server
- Check network connectivity: `docker exec error-generator ping slogan-server`
- Verify slogan server is running: `docker-compose ps`
- Check slogan server logs: `docker-compose logs slogan-server`

### Giphy rate limiting
- The application batch loads 25 GIFs at a time
- With 60-second intervals, this provides 25 minutes of runtime per batch
- Set `GIPHY_API_KEY` for real GIFs, or it will use placeholders

### Docker build issues
- Ensure Go 1.21+ is specified in Dockerfile
- Check for network issues during `go mod download`
- Try cleaning Docker cache: `docker-compose build --no-cache`

## License

For testing purposes only.
