# Troubleshooting Guide

## Common Issues and Solutions

### Port 5000 Already in Use

**Error:**
```
Address already in use
Port 5000 is in use by another program.
```

**Cause:** macOS AirPlay Receiver uses port 5000

**Solution 1 (Recommended):** Use port 5001 (now the default)
- The service now defaults to port 5001
- Make sure you're running the latest version
- Check that `.env` file has `PORT=5001`

**Solution 2:** Set port explicitly
```bash
PORT=5001 python3 rhythm_service.py
```

**Solution 3:** Disable AirPlay
1. Open System Settings
2. Go to General → AirDrop & Handoff
3. Turn off AirPlay Receiver

### urllib3 OpenSSL Warning

**Warning:**
```
urllib3 v2 only supports OpenSSL 1.1.1+, currently the 'ssl' module is compiled with 'LibreSSL 2.8.3'
```

**Cause:** macOS uses LibreSSL instead of OpenSSL

**Impact:** Non-critical, service works fine

**Solution:** Suppress the warning
```bash
export PYTHONWARNINGS="ignore::UserWarning"
python3 rhythm_service.py
```

Or use the provided run script:
```bash
./run.sh
```

### hls4ml Not Available

**Warning:**
```
hls4ml not available, FPGA features disabled
QKeras not available, quantization features limited
```

**Cause:** Optional dependencies not installed

**Impact:** FPGA acceleration disabled (CPU mode works fine)

**Solution (if you want FPGA features):**
```bash
pip install hls4ml qkeras
```

**Note:** For most users, CPU mode is sufficient for testing.

### Spotify Credentials Warning

**Warning:**
```
⚠️  Spotify credentials not set. Some features may be limited.
```

**Solution:** Add Spotify credentials to `.env`
```bash
# Get credentials from https://developer.spotify.com/dashboard
SPOTIFY_CLIENT_ID=your_client_id_here
SPOTIFY_CLIENT_SECRET=your_client_secret_here
```

### ModuleNotFoundError: librosa

**Error:**
```
ModuleNotFoundError: No module named 'librosa'
```

**Solution:**
```bash
# macOS
brew install libsndfile
pip install librosa

# Ubuntu
sudo apt-get install libsndfile1
pip install librosa
```

## Quick Start Commands

### Option 1: Use Run Script (Recommended)
```bash
cd rhythm-service
./run.sh
```

### Option 2: Manual Start with Port
```bash
cd rhythm-service
PORT=5001 python3 rhythm_service.py
```

### Option 3: Suppress All Warnings
```bash
cd rhythm-service
export PYTHONWARNINGS="ignore"
PORT=5001 python3 rhythm_service.py
```

## Verify Service is Running

```bash
# Check health
curl http://localhost:5001/health

# Expected response:
# {"status":"healthy","service":"rhythm-service","fpga_enabled":false}
```

## Check What's Using Port 5000

```bash
lsof -i :5000
# Usually shows: ControlCenter (macOS AirPlay)
```

## Full Clean Start

```bash
cd rhythm-service

# 1. Install dependencies
pip install -r requirements.txt

# 2. Create/update .env file
cp .env.example .env
# Edit .env if needed

# 3. Run with explicit port
PORT=5001 python3 rhythm_service.py
```

## Still Having Issues?

1. **Check Python version:**
   ```bash
   python3 --version  # Should be 3.8+
   ```

2. **Check pip packages:**
   ```bash
   pip list | grep -E "flask|librosa|tensorflow"
   ```

3. **Check environment variables:**
   ```bash
   cat .env
   ```

4. **Check logs carefully:**
   - Look for actual errors vs warnings
   - Warnings about hls4ml/QKeras/urllib3 are normal
   - Only port conflicts need fixing

5. **Try the run script:**
   ```bash
   chmod +x run.sh
   ./run.sh
   ```

## Success Indicators

When starting successfully, you should see:
```
2025-11-04 12:19:14,204 - __main__ - INFO - 🎼 Rhythm Service Starting...
2025-11-04 12:19:14,204 - __main__ - INFO - FPGA Mode: DISABLED (CPU Simulation)
2025-11-04 12:19:14,204 - __main__ - INFO - Error Generator URL: http://localhost:9090
 * Serving Flask app 'rhythm_service'
 * Debug mode: off
 * Running on http://0.0.0.0:5001
```

The key line is: `* Running on http://0.0.0.0:5001`

If you see that, the service is running correctly! 🎉
