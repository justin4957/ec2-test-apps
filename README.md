# Automated Nonsense Error Application (remote server)

Three minimal Go applications + one Python ML service for automated repetitive unencrypted absurd error logging on remote servers, now with **rhythm-driven error generation** powered by machine learning!

![Errors](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExcHJ0bXp4bjBvNzBsZGJscGRyeWJlbnRqZG5ydzZ3YjZxdGw3eDJ2YyZlcD12MV9naWZzX3NlYXJjaCZjdD1n/3o7btPCcdNniyf0ArS/giphy.gif)

<img width="1102" height="556" alt="Screenshot 2025-10-28 at 10 02 17 AM" src="https://github.com/user-attachments/assets/868b4222-e365-4cfd-b1e5-939202afcf91" />


example
```http://slogan-server:8080
error-generator-1  | 2025/10/28 13:52:14 Sending errors every 60 seconds
error-generator-1  | 2025/10/28 13:52:14 GIPHY_API_KEY not set, using placeholder GIFs
error-generator-1  | 2025/10/28 13:52:14 Sending error: NullPointerException in
UserService.java:42
error-generator-1  | 2025/10/28 13:52:14 With GIF: https://giphy.com/gifs/error-placeholder-1
slogan-server-1    | 2025/10/28 13:52:14 Received error log: NullPointerException in
UserService.java:42 (GIF: https://giphy.com/gifs/error-placeholder-1)
slogan-server-1    | 2025/10/28 13:52:14 Responded with slogan: Off by one: Close enough is
good enough
error-generator-1  | 2025/10/28 13:52:14 Received response: 🚬 Off by one: Close enough is good
 enough
error-generator-1  |
error-generator-1  | === ERROR LOG ===
error-generator-1  | Error: `[NullPointerException in UserService.JAVA:42](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExNWllcmdiZ2p2amNtemM0MXkyem9oYjc0MnBndzc0Yzh1NzB6cXR2YyZlcD12MV9naWZzX3NlYXJjaCZjdD1n/olAik8MhYOB9K/giphy.gif)`
error-generator-1  | GIF: https://giphy.com/gifs/error-placeholder-1
error-generator-1  | Response: 🚬 Off by one: Close enough is good enough
error-generator-1  | ================
error-generator-1  |
```

<img width="1087" height="583" alt="Screenshot 2025-10-28 at 12 36 15 PM" src="https://github.com/user-attachments/assets/8b76a1ff-4bd5-413c-9e8e-fe8c2109ec0f" />


## The Point

This application exists to ensure there is a **paid-for absurd error log**, complete with appropriately comical GIFs and music from the "Silver Screen Static" playlist, running on remote computers somewhere in the cloud. It's performatively ridiculous for repetitive advertising purposes - because nothing says "professional engineering" quite like spending actual money to have a server respond to fake errors with cigarette emojis and nonsensical slogans every 60 seconds.

The error logs are intentionally meaningless. The GIFs are deliberately absurd. The music is perfectly curated alternative nostalgia. The whole thing runs on actual EC2 instances that cost real money. This is art. This is advertising. This is what happens when you have Docker, FPGA-based ML inference, and no adult supervision.

![Music](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExbWI0aWF3bnVzamd3YnI4MGh3ZWx6Z2RlcnJ2bHFkbTVxZXljdDBpZSZlcD12MV9naWZzX3NlYXJjaCZjdD1n/l0HlNQ03J5JxX6lva/giphy.gif)

## Overview

### Slogan Server
HTTP server that receives error log messages and responds with:
- A cigarette emoji (🚬)
- An **AI-generated sardonic slogan** using OpenAI GPT-4o-mini
- Extracts context from error messages and GIF URLs
- Falls back to 115 pre-generated slogans if OpenAI is unavailable

### Error Generator
Client application that:
- Batch loads GIF URLs from Giphy API (to avoid rate limiting)
- Every minute (configurable) generates a random error log message
- Retrieves a random GIF URL from the cached batch
- Plays songs from the curated **"Silver Screen Static"** playlist (100 tracks of alternative nostalgia)
- Sends the error message unencrypted to the slogan server
- Displays the response

### Location Tracker
Personal security / educational tool that:
- Tracks location of **YOUR OWN** devices (phone, laptop, etc.)
- Password-protected web interface
- Real-time location sharing with trusted people
- Auto-refreshing location display (10s intervals)
- Direct Google Maps integration
- Locations auto-expire after 24 hours
- **ONLY for devices you own with explicit consent**

  ![](https://media2.giphy.com/media/v1.Y2lkPTc5MGI3NjExa2oyZTQ4aTg1dzF3Ymc1aDhwaHBhZmttdXh0NTkzbW92bTBwYnp0biZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/ECwTCTrHPVqKI/giphy.gif)

### 🎵 Rhythm-Driven Error Generator (NEW!)
**Educational ML project** that uses FPGA-accelerated neural networks to sync error generation with music:
- **Beat detection ML model** deployed on FPGA via hls4ml
- Real-time audio analysis of Spotify tracks
- **Song structure detection** (verse/chorus/bridge) triggers different error patterns
- Ultra-low-latency inference (~microseconds) for musical synchronization
- Transforms error logging into a rhythmic performance
- See [rhythm-service/README.md](rhythm-service/README.md) for details

![FPGA](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExcnFxN2duaDJ2dmF4bjJ3NG5ocHFucHZja2hscTB5cGRhY3p4aGtybyZlcD12MV9naWZzX3NlYXJjaCZjdD1n/3oKIPnAiaMCws8nOsE/giphy.gif)

## Architecture

```
┌─────────────────────┐         HTTP POST          ┌─────────────────┐
│  Error Generator    │ ──────────────────────────> │  Slogan Server  │
│                     │                              │                 │
│ - Giphy cache       │   {"message": "...",        │ - 115 slogans   │
│ - Timer (dynamic)   │    "gif_url": "...",        │ - OpenAI GPT-4  │
│ - Spotify songs     │    "song": {...}}           │ - Random picker │
│ - Businesses        │                              │                 │
│                     │ <────────────────────────── │                 │
│                     │   {"emoji": "🚬",           │                 │
│                     │    "slogan": "..."}         │                 │
└─────────────────────┘                              └─────────────────┘
         │ ▲                                                │
         │ │ rhythm signals                                 │
         │ │                                                │
         v │                                                v
┌─────────────────────────────────────────────────────────────────┐
│                  Rhythm Service (Python + hls4ml)               │
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐    │
│  │  Beat        │    │  Song        │    │  FPGA        │    │
│  │  Detection   │───>│  Structure   │───>│  Inference   │    │
│  │  (RNN/LSTM)  │    │  Analysis    │    │  (hls4ml)    │    │
│  └──────────────┘    └──────────────┘    └──────────────┘    │
│                                                                 │
│  - Spotify API       - Verse/Chorus       - Ultra-low latency  │
│  - Audio features    - Bridge/Outro       - Real-time sync     │
│  - Tempo analysis    - Pattern matching   - Error triggers     │
└─────────────────────────────────────────────────────────────────┘
         │                                                │
         │ POST /api/errorlogs                           │
         v                                                v
┌──────────────────────────────────────────────────────────────┐
│                     Location Tracker                         │
│                                                              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    │
│  │   Auth &    │    │   Error     │    │   Twilio    │    │
│  │   Location  │    │   Logging   │    │   SMS       │    │
│  │   Tracking  │    │   Storage   │    │   Webhook   │    │
│  └─────────────┘    └─────────────┘    └─────────────┘    │
│                                                              │
│  - Password auth      - Error logs         - SMS → Notes   │
│  - Real-time map      - DynamoDB           - Auto-attach   │
│  - 24h retention      - Google Maps        - User feedback │
│  - Auto-refresh       - Businesses         - Context       │
└──────────────────────────────────────────────────────────────┘
         ▲                                           ▲
         │                                           │
    Browser/Device                          Twilio SMS Webhook
```

## Features

### 🤖 AI-Powered Slogan Generation
- **OpenAI GPT-4o-mini** generates unique, sardonic slogans for each error
- Analyzes error messages and GIF context for contextual humor
- Temperature: 0.9 for maximum creativity
- Example: `DeadlockDetected: Embrace the bliss of perpetual stagnation!`

![AI](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExNXE3YnFiaGd5ZGo5dHFsNHZ5ajA5dXB4eGN4Z3I2eWlzd3hpNm5zbiZlcD12MV9naWZzX3NlYXJjaCZjdD1n/3o7qE1YN7aBOFPRw8E/giphy.gif)

### 🎨 Real GIFs from Giphy
- Batch loads 25 GIFs per request to avoid rate limiting
- Random search terms: "error", "fail", "glitch", "broken", "oops"
- Extracts GIF context from URLs for better AI prompts

### 🎵 Silver Screen Static Playlist
The error generator features a carefully curated playlist of **100 tracks** spanning alternative classics, soundtrack gems, and nostalgic deep cuts:

**Act I: Suburban Dreams & Mall Rat Nights**
- The Killing Moon - Echo & the Bunnymen
- If You Leave - Orchestral Manoeuvres in the Dark
- Please, Please, Please Let Me Get What I Want - The Dream Academy
- Don't You (Forget About Me) - Simple Minds
- Under the Milky Way - The Church
- There Is a Light That Never Goes Out - The Smiths

**Act II: Late Night AM Radio & Static Transmissions**
- Into Dust - Mazzy Star
- Fade Into You - Mazzy Star
- Wicked Game - Chris Isaak
- Nightswimming - R.E.M.
- Song to the Siren - This Mortal Coil

**Act III: The Alternative Nation - 90s Soundtrack Gold**
- #1 Crush - Garbage
- 6 Underground - Sneaker Pimps
- How Soon Is Now? - Love Spit Love
- Where Is My Mind? - Pixies
- Smells Like Teen Spirit - Nirvana
- Blister In The Sun - Violent Femmes

**Act IV: Cinematic Weirdness & Cult Classics**
- I'm Deranged - David Bowie
- In Heaven - The Lady in the Radiator (Twin Peaks)

**Act V: Lost Highways & Open Roads**
- Walking After You - Foo Fighters
- The Passenger - Iggy Pop
- Roadrunner - The Modern Lovers
- Transatlanticism - Death Cab for Cutie

**Act VI: The Credits Roll - Bittersweet Endings**
- New Slang - The Shins
- Such Great Heights - Iron & Wine
- The End of the World - Skeeter Davis

...and 70+ more tracks spanning grunge, Britpop, post-punk, shoegaze, and modern indie!

![Vinyl](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExMnB6YmJqc3htdGZqZHU4eTVlOTlyZGpnN3lsMzR6M2t0ZDdwaDIzNyZlcD12MV9naWZzX3NlYXJjaCZjdD1n/l0IyjiXOXTlqfwls4/giphy.gif)

### 🎼 Rhythm-Driven Error Patterns (Educational ML Feature)
Learn machine learning and FPGA deployment by building a system that:
- **Trains an RNN/LSTM** to recognize song structure patterns
- **Deploys models to FPGA** using hls4ml for ultra-low-latency inference
- **Synchronizes error generation** with musical beats and sections:
  - **Verse**: Basic errors (`NullPointer`, `IndexOutOfBounds`)
  - **Chorus**: Business-related errors with nearby business names
  - **Bridge**: Chaotic multi-error bursts
  - **Outro**: Philosophical errors with governing body references
- **Real-time audio analysis** via Spotify API
- **Educational value**: Audio ML, sequence modeling, hardware acceleration, system integration

### 🛡️ Intelligent Fallback
- 115 pre-generated sardonic slogans as backup
- Automatic fallback if OpenAI API fails
- No interruption to service

### 🔐 Secure Configuration
- API keys stored in `.env.ec2` (git-ignored)
- Environment variable-based configuration
- No hardcoded secrets

### 📊 Observable
- Logs indicate slogan source: `(openai)` or `(fallback)`
- Real-time error/slogan streaming
- Container health checks

### 📍 Location Tracking (Educational / Personal Security)
- **Password-protected** web interface for viewing device locations
- Real-time GPS location sharing from any device
- Auto-refresh every 10 seconds
- Direct links to Google Maps
- Shows location accuracy (±20m) and timestamp
- In-memory storage (no database needed)
- Auto-cleanup after 24 hours
- **IMPORTANT**: ONLY for tracking YOUR OWN devices with explicit consent
- See [location-tracker/README.md](location-tracker/README.md) for full details

### 📱 Twilio SMS Integration
- **User Experience Notes**: Send SMS messages via Twilio that attach to error logs
- SMS messages are stored and automatically attached to the next error log
- Provides real-world context and user feedback for error tracking
- Simple webhook endpoint: `/api/twilio/sms`
- Notes displayed in UI with special highlighting
- See [TWILIO_INTEGRATION.md](TWILIO_INTEGRATION.md) for setup and configuration

  Error:
<img width="481" height="187" alt="Screenshot 2025-10-29 at 12 05 48 PM" src="https://github.com/user-attachments/assets/701e2516-41cb-4370-8f96-efda2758d448" />

![SMS](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExd2duYnJjdWE1amVhY3lpMmViNzQwZmJlMW5sNHJsMmV3dmhvNmU3NCZlcD12MV9naWZzX3NlYXJjaCZjdD1n/xT0xeJpnrWC4XWblEk/giphy.gif)

## Local Testing

### Prerequisites
- Docker
- Docker Compose
- (Optional) Giphy API key for real GIF URLs
- (Optional) OpenAI API key for AI-generated slogans
- (Optional) Spotify API credentials for music integration
- (Optional) Python 3.8+ for rhythm service

### Quick Start

1. Clone or navigate to the project directory:
```bash
cd ec2-test-apps
```

2. **(Recommended)** Set up API keys for full experience:

**For local testing:**
```bash
cp .env.example .env
# Edit .env and add your API keys:
# - GIPHY_API_KEY (for real GIFs)
# - OPENAI_API_KEY (for AI-generated slogans)
# - SPOTIFY_CLIENT_ID (for music)
# - SPOTIFY_CLIENT_SECRET (for music)
```

**For EC2 deployment:**
```bash
cp .env.ec2.example .env.ec2
# Edit .env.ec2 and add your API keys:
# - GIPHY_API_KEY (for real GIFs)
# - OPENAI_API_KEY (for AI-generated slogans)
# - SPOTIFY_CLIENT_ID (for music)
# - SPOTIFY_CLIENT_SECRET (for music)
```

See [GIPHY_API_SETUP.md](GIPHY_API_SETUP.md) for detailed configuration instructions.

3. Build and run with Docker Compose:
```bash
docker-compose up --build
```

4. Watch the logs to see error generation and slogan responses every minute.

5. **(Optional)** Enable rhythm-driven mode:
```bash
cd rhythm-service
pip install -r requirements.txt
python rhythm_service.py
```

6. **(Demo!)** Try the rhythm-driven error demo - generates errors every 16 beats:
```bash
# In one terminal, start error-generator in rhythm mode
cd error-generator
RHYTHM_SERVICE_URL=http://localhost:5001 go run main.go

# In another terminal, run the demo
cd rhythm-service
python3 demo_rhythm_errors.py
```

This simulates an entire song (intro → verses → chorus → bridge → outro) and triggers errors synchronized to the beat every 16 beats (4 bars). Perfect for demos and understanding the rhythm-to-error mapping! See [rhythm-service/README.md](rhythm-service/README.md#-demo-mode) for more options.

![Rocket](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExdnpoa3N5cjNrYnBiM2t5c3l3bGk4aGxjYWI5NXZwanljdmU0YmpwbiZlcD12MV9naWZzX3NlYXJjaCZjdD1n/3o6Zt6ML6BklcajjsA/giphy.gif)

### Configuration

Environment variables for `slogan-server`:

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENAI_API_KEY` | OpenAI API key for AI-generated slogans (optional) | Falls back to pre-generated slogans if not set |

Environment variables for `error-generator`:

| Variable | Description | Default |
|----------|-------------|---------|
| `SLOGAN_SERVER_URL` | URL of the slogan server | `http://localhost:8080` |
| `ERROR_INTERVAL_SECONDS` | Seconds between error logs | `60` |
| `GIPHY_API_KEY` | Giphy API key (optional) | Placeholder URLs if not set |
| `SPOTIFY_CLIENT_ID` | Spotify API client ID (optional) | Placeholder songs if not set |
| `SPOTIFY_CLIENT_SECRET` | Spotify API client secret (optional) | Placeholder songs if not set |
| `RHYTHM_SERVICE_URL` | URL of rhythm service for ML-driven timing (optional) | Standard interval if not set |

Environment variables for `rhythm-service`:

| Variable | Description | Default |
|----------|-------------|---------|
| `SPOTIFY_CLIENT_ID` | Spotify API client ID (required) | None |
| `SPOTIFY_CLIENT_SECRET` | Spotify API client secret (required) | None |
| `USE_FPGA` | Enable FPGA acceleration via hls4ml | `false` (CPU simulation) |

**Getting API Keys:**
- **Giphy**: Get a free API key at [https://developers.giphy.com/](https://developers.giphy.com/)
- **OpenAI**: Get an API key at [https://platform.openai.com/api-keys](https://platform.openai.com/api-keys)
- **Spotify**: Create an app at [https://developer.spotify.com/dashboard](https://developer.spotify.com/dashboard)

See [GIPHY_API_SETUP.md](GIPHY_API_SETUP.md) for detailed configuration instructions.

### Testing Individual Services

Build and run slogan-server:
```bash
cd slogan-server
docker build -t slogan-server .
docker run -p 8080:8080 -e OPENAI_API_KEY=your_key_here slogan-server
# Or without OpenAI (uses fallback slogans):
# docker run -p 8080:8080 slogan-server
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

## Educational Resources

### Learning FPGA-based Machine Learning
- [hls4ml Documentation](https://fastmachinelearning.org/hls4ml/)
- [Xilinx Vitis HLS Guide](https://www.xilinx.com/products/design-tools/vitis/vitis-hls.html)
- [Audio Feature Extraction for ML](https://librosa.org/)
- [RNN/LSTM for Sequence Modeling](https://www.tensorflow.org/guide/keras/rnn)

### Music Information Retrieval
- [Librosa for Audio Analysis](https://librosa.org/doc/latest/index.html)
- [Spotify Web API](https://developer.spotify.com/documentation/web-api)
- [Million Song Dataset](http://millionsongdataset.com/)

![Learning](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExNTdqOTlvaGJ6b2tveWkzYW90M2JsdTRxdmhsY2x5MWpmdGt5MzhiZCZlcD12MV9naWZzX3NlYXJjaCZjdD1n/3oKIPnf83YOnVZ7ptu/giphy.gif)

## License

For testing and educational purposes only.
