# Rhythm Mode Setup Guide

Complete guide for setting up rhythm-driven error generation!

![Rhythm](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExcnFxN2duaDJ2dmF4bjJ3NG5ocHFucHZja2hscTB5cGRhY3p4aGtybyZlcD12MV9naWZzX3NlYXJjaCZjdD1n/3oKIPnAiaMCws8nOsE/giphy.gif)

## Overview

Rhythm mode synchronizes error generation with music beats. Instead of generating errors every 60 seconds, errors are triggered in response to musical events (beats, sections, tempo changes).

## Architecture

```
┌─────────────────────┐         HTTP POST          ┌──────────────────────┐
│  Rhythm Service     │─────────────────────────────>│  Error Generator     │
│  (Python)           │  /api/rhythm-trigger       │  (Go)                │
│  Port: 5001         │                             │  Port: 9090          │
│                     │  {"trigger":"rhythm"...}    │                      │
│  - Beat Detection   │<────────────────────────────│  - HTTP Server       │
│  - Song Analysis    │  {"success":true}           │  - Rhythm Handler    │
│  - FPGA Inference   │                             │  - Error Generator   │
└─────────────────────┘                             └──────────────────────┘
```

## Port Configuration

We use **port 5001** for the rhythm service to avoid conflicts with macOS AirPlay Receiver which uses port 5000.

## Setup Instructions

### Step 1: Configure Environment Variables

Create `.env` files for both services:

**rhythm-service/.env**
```bash
# Spotify API Credentials
SPOTIFY_CLIENT_ID=your_client_id
SPOTIFY_CLIENT_SECRET=your_client_secret

# Error Generator URL
ERROR_GENERATOR_URL=http://localhost:9090

# Service Port (avoiding macOS AirPlay on 5000)
PORT=5001

# FPGA Mode
USE_FPGA=false
```

**error-generator environment** (or add to docker-compose.yml)
```bash
# Enable rhythm mode
RHYTHM_SERVICE_URL=http://localhost:5001

# HTTP server port for receiving triggers
ERROR_GENERATOR_PORT=9090

# Other configs
SLOGAN_SERVER_URL=http://localhost:8080
GIPHY_API_KEY=your_giphy_key
SPOTIFY_CLIENT_ID=your_spotify_client_id
SPOTIFY_CLIENT_SECRET=your_spotify_client_secret
```

### Step 2: Start the Services

**Terminal 1 - Slogan Server**
```bash
cd slogan-server
docker run -p 8080:8080 slogan-server
# Or run locally:
# go run main.go
```

**Terminal 2 - Error Generator (Rhythm Mode)**
```bash
cd error-generator

# Set environment variables
export RHYTHM_SERVICE_URL=http://localhost:5001
export ERROR_GENERATOR_PORT=9090
export SLOGAN_SERVER_URL=http://localhost:8080

# Run error generator
go run main.go
```

You should see:
```
Error Generator starting...
🎵 Rhythm mode ENABLED - listening on port 9090
Rhythm service URL: http://localhost:5001
🎵 Rhythm mode active - waiting for triggers from rhythm service...
Send triggers to: http://localhost:9090/api/rhythm-trigger
```

**Terminal 3 - Rhythm Service**
```bash
cd rhythm-service

# Install dependencies (if not already done)
pip install -r requirements.txt

# Start service
python rhythm_service.py
```

You should see:
```
🎼 Rhythm Service Starting...
FPGA Mode: DISABLED (CPU Simulation)
Error Generator URL: http://localhost:9090
 * Running on http://0.0.0.0:5001
```

### Step 3: Test the Integration

**Health Checks**
```bash
# Check rhythm service
curl http://localhost:5001/health

# Check error generator
curl http://localhost:9090/health
```

**Manual Trigger Test**
```bash
curl -X POST http://localhost:5001/api/beat-trigger \
  -H "Content-Type: application/json" \
  -d '{
    "section": "chorus",
    "beat_num": 1
  }'
```

Watch the error-generator terminal - you should see:
```
🎵 Rhythm trigger received: chorus section, beat 1, tempo 120.0 BPM
✓ Trigger queued for processing
🎼 Processing business trigger (beat 1)
Sending rhythm-synced error: PaymentGatewayTimeout...
```

**Analyze a Song**
```bash
curl -X POST http://localhost:5001/api/analyze-song \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Where Is My Mind?",
    "artist": "Pixies",
    "audio_features": {
      "tempo": 115,
      "energy": 0.8,
      "danceability": 0.6
    }
  }'
```

Response:
```json
{
  "success": true,
  "tempo": 115.0,
  "beat_interval": 0.521,
  "structure": [
    {"start": 0, "duration": 11.7, "type": "intro"},
    {"start": 11.7, "duration": 35.1, "type": "verse"},
    {"start": 46.8, "duration": 23.4, "type": "chorus"},
    ...
  ]
}
```

## How It Works

### 1. Song Analysis

When you POST to `/api/analyze-song`, the rhythm service:
- Extracts tempo from audio features
- Estimates or retrieves song structure
- Maps sections (verse/chorus/bridge) to error types

### 2. Beat Detection

The rhythm service:
- Calculates beat intervals from tempo
- Uses RNN/LSTM models (or FPGA) for precise detection
- Identifies current song section

### 3. Trigger Sending

For each beat or section change, the rhythm service sends:
```json
{
  "trigger": "rhythm",
  "error_type": "business",  // verse|chorus|bridge|outro
  "beat": 16,
  "section": "chorus",
  "tempo": 125.0
}
```

### 4. Error Generation

Error-generator receives trigger and:
- Selects appropriate error type based on section
- Fetches GIF and song from caches
- Generates error synchronized to beat
- Sends to slogan server
- Logs to location tracker

## Section → Error Type Mapping

| Section | Error Type | Examples |
|---------|------------|----------|
| Intro | Minimal | Simple, sparse errors |
| Verse | Basic | `NullPointerException`, `IndexOutOfBounds` |
| Chorus | Business | `PaymentGatewayTimeout`, `InventoryMismatch` |
| Bridge | Chaotic | Rapid multi-error bursts |
| Outro | Philosophical | Errors with governing body references |

## Troubleshooting

### Port 5000 Already in Use
**Problem**: macOS AirPlay Receiver uses port 5000

**Solution**: Rhythm service now defaults to port 5001

### Error Generator Not Receiving Triggers
**Check**:
1. Is `RHYTHM_SERVICE_URL` set?
2. Is error generator listening on correct port?
3. Are there firewall issues?

**Test**:
```bash
# Check error generator is listening
lsof -i :9090

# Test trigger directly
curl -X POST http://localhost:9090/api/rhythm-trigger \
  -H "Content-Type: application/json" \
  -d '{"trigger":"rhythm","error_type":"basic","beat":1,"section":"verse","tempo":120}'
```

### Rhythm Service Connection Errors
**Check**:
```bash
# Verify error generator URL in rhythm service
curl http://localhost:5001/api/status
```

**Fix**: Update `ERROR_GENERATOR_URL` in rhythm-service/.env

### Dependencies Missing
```bash
cd rhythm-service
pip install -r requirements.txt
```

Common issues:
- **Librosa**: Requires `libsndfile` (macOS: `brew install libsndfile`)
- **TensorFlow**: May need specific version for your system
- **hls4ml**: Optional, only needed for FPGA mode

## Performance Tips

### CPU Mode (Default)
- Beat detection: ~10-50ms latency
- Good for testing and development
- No special hardware needed

### FPGA Mode (Advanced)
- Beat detection: ~1-10μs latency
- Requires FPGA board and synthesis
- 1000-5000x faster than CPU
- See hls4ml documentation for setup

## Example Workflow

```bash
# 1. Start all services
# Terminal 1: Slogan server on 8080
# Terminal 2: Error generator on 9090 (rhythm mode)
# Terminal 3: Rhythm service on 5001

# 2. Analyze a song
curl -X POST http://localhost:5001/api/analyze-song \
  -H "Content-Type: application/json" \
  -d '{"name":"Smells Like Teen Spirit","artist":"Nirvana","audio_features":{"tempo":117}}'

# 3. Trigger beats manually (or let rhythm service do it automatically)
for i in {1..10}; do
  curl -X POST http://localhost:5001/api/beat-trigger \
    -H "Content-Type: application/json" \
    -d "{\"section\":\"chorus\",\"beat_num\":$i}"
  sleep 0.5
done

# 4. Watch errors being generated in sync!
```

## Next Steps

- **Real-time Audio**: Integrate live audio streaming
- **Automatic Mode**: Have rhythm service continuously analyze and trigger
- **Visualization**: Build a web dashboard showing beats and errors
- **FPGA Deployment**: Synthesize models for microsecond latency
- **Multiple Songs**: Queue up an entire playlist

![Party](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExb3dzZnE2cXRhbWZqbWFudGRjMGlub3Zua2R0ZTRkYTY3ZHk3cTV0YSZlcD12MV9naWZzX3NlYXJjaCZjdD1n/l0MYt5jPR6QX5pnqM/giphy.gif)

## Configuration Summary

| Service | Port | Environment Variable | Default |
|---------|------|---------------------|---------|
| Rhythm Service | 5001 | `PORT` | 5001 |
| Error Generator | 9090 | `ERROR_GENERATOR_PORT` | 9090 |
| Slogan Server | 8080 | N/A | 8080 |

Enjoy your rhythm-synced errors! 🎵🚀
