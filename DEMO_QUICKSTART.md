# 🎵 Rhythm-Driven Error Demo - Quick Start

Experience the **full 3-service integration** with rhythm-driven error generation, slogans, and GIFs!

## What This Demo Does

The demo script simulates an entire song and generates errors every 16 beats (4 bars), demonstrating:

- 🎼 **Beat-synchronized error generation** (rhythm service → error generator)
- 🎭 **Section-based error mapping**:
  - **Intro** → basic errors
  - **Verse** → basic errors (NullPointer, IndexOutOfBounds)
  - **Chorus** → business errors (with nearby business names)
  - **Bridge** → **chaotic errors** (cascading failures, quantum bugs!)
  - **Outro** → **philosophical errors** (existential, absurdist)
- 💬 **Slogan generation** (error generator → slogan server → AI slogans!)
- 🎨 **GIF integration** (errors get paired with hilarious GIFs)
- 📊 **Real-time statistics** and progress tracking

## Architecture

```
┌──────────────┐         ┌────────────────┐         ┌──────────────┐
│ Demo Script  │─trigger→│ Error Generator│─request→│Slogan Server │
│  (Python)    │         │     (Go)       │         │    (Go)      │
└──────────────┘         └────────────────┘         └──────────────┘
                                │                           │
                                │                           │
                         [generates error]           [creates slogan
                         [with GIF URL]               with emoji 🚬]
                                │                           │
                                └─────────logs─────────────┘
```

## 5-Minute Quick Start

### Step 1: Start the Slogan Server (Terminal 1)

```bash
cd slogan-server
go run main.go
```

**Expected output:**
```
Starting slogan server on :8080
115 sardonic slogans loaded
```

### Step 2: Start the Error Generator (Terminal 2)

```bash
cd error-generator
RHYTHM_SERVICE_URL=http://localhost:5001 go run main.go
```

**Expected output:**
```
Starting error generator...
🎵 Rhythm mode ENABLED
HTTP server listening on :9090
```

### Step 3: Run the Demo (Terminal 3)

```bash
cd rhythm-service
python3 demo_rhythm_errors.py
```

**What you'll see:**

1. **Song Structure Display:**
```
🎵 SONG STRUCTURE
==========================================
Tempo: 120 BPM | Beat Duration: 0.500s
Error Trigger Interval: Every 16 beats (4 bars)

Section         Beats           Duration        Error Type
----------------------------------------------------------
intro           0-8              4.0s           basic
verse           8-24            16.0s           basic
chorus          32-48           16.0s           business
bridge          88-104          16.0s           chaotic
outro           120-128          8.0s           philosophical
----------------------------------------------------------
TOTAL           128             64.0s
Expected Error Triggers: 8
```

2. **Real-time Error Triggers:**
```
🎵 Beat  16 | verse        | basic          | Trigger #1
🎵 Beat  32 | chorus       | business       | Trigger #2
🎵 Beat  48 | verse        | basic          | Trigger #3
🎵 Beat  64 | chorus       | business       | Trigger #4
🎵 Beat  72 | chorus       | business       | Trigger #5
🎵 Beat  88 | bridge       | chaotic        | Trigger #6
🎵 Beat 104 | chorus       | business       | Trigger #7
🎵 Beat 120 | outro        | philosophical  | Trigger #8
```

3. **Summary Statistics:**
```
📊 DEMO SUMMARY
==========================================
Song Duration:       64.0s (128 beats)
Execution Time:      0.3s
Triggers Sent:       8
Expected Triggers:   8
Tempo:               120 BPM
Trigger Interval:    Every 16 beats
```

### Step 4: Watch the Magic Happen! 🎉

Now you'll see activity across all three terminals showing the complete flow:

**Terminal 3 (Demo Script):**
```
🎵 Beat  16 | verse        | basic          | Trigger #1
🎵 Beat  32 | chorus       | business       | Trigger #2
🎵 Beat  88 | bridge       | chaotic        | Trigger #6
🎵 Beat 120 | outro        | philosophical  | Trigger #8
```

**Terminal 2 (Error Generator):**
```
🎵 Rhythm trigger received: verse section, beat 16, tempo 120.0 BPM
✓ Trigger queued for processing
🎼 Processing basic trigger (beat 16)
Sending rhythm-synced error: NullPointerException in UserService.java:42
Received response: 🚬 Off by one: Close enough is good enough

🎵 Rhythm trigger received: bridge section, beat 88, tempo 120.0 BPM
🌀 CHAOTIC ERROR: QUANTUM SUPERPOSITION: Error both exists and doesn't exist until observed
Received response: 🚬 It's not a bug, it's a feature in disguise

🎵 Rhythm trigger received: outro section, beat 120, tempo 120.0 BPM
🤔 PHILOSOPHICAL ERROR: ExistentialException: If a server crashes in the cloud and no one is monitoring, did it really fail?
Received response: 🚬 404: Empathy not found
```

**Terminal 1 (Slogan Server):**
```
Received error log: NullPointerException in UserService.java:42
Responded with slogan: Off by one: Close enough is good enough

Received error log: QUANTUM SUPERPOSITION: Error both exists and doesn't exist...
Responded with slogan: It's not a bug, it's a feature in disguise

Received error log: ExistentialException: If a server crashes...
Responded with slogan: 404: Empathy not found
```

## Demo Options

### 🎵 Use Real Spotify Tracks! (NEW!)

Now you can use **actual songs from Spotify** with real tempo, duration, and structure!

#### Setup Spotify Credentials

```bash
# Add to your .env file or export
export SPOTIFY_CLIENT_ID="your_client_id_here"
export SPOTIFY_CLIENT_SECRET="your_client_secret_here"
```

Get credentials from: https://developer.spotify.com/dashboard

#### Use a Real Track

```bash
# By track and artist name
python3 demo_rhythm_errors.py --track "Where Is My Mind?" --artist "Pixies"

# By Spotify URI
python3 demo_rhythm_errors.py --spotify-uri "spotify:track:5EWPGh7jbTNO2wakv8LjUI"

# Real-time mode with Spotify track
python3 demo_rhythm_errors.py --track "Smells Like Teen Spirit" --artist "Nirvana" --realtime
```

**What you'll see:**
```
🎵 Spotify Mode: Using real track data

📡 Checking service health...

✓ Authenticated with Spotify API
✓ Found track: Where Is My Mind? by Pixies
🎵 Track: Where Is My Mind? by Pixies
   Duration: 233.0s

✓ Retrieved audio analysis (11 sections, 485 beats)
✓ Retrieved audio features (tempo: 114.9 BPM)
✓ Parsed 11 sections from Spotify analysis

📀 Now Playing: Where Is My Mind? by Pixies
```

The demo will use the **actual song structure** from Spotify's audio analysis API, mapping real sections to error types!

### Real-Time Mode

Simulate actual song playback with delays between beats:

```bash
python3 demo_rhythm_errors.py --realtime
```

This will take ~64 seconds (the full "song" duration) instead of running instantly.

### Custom Tempo (Simulated Mode Only)

Try different tempos to see how beat intervals change:

```bash
# Faster tempo (140 BPM - upbeat pop)
TEMPO=140 python3 demo_rhythm_errors.py

# Slower tempo (80 BPM - ballad)
TEMPO=80 python3 demo_rhythm_errors.py
```

### Custom Error Generator URL

If your error-generator is running on a different port:

```bash
ERROR_GENERATOR_URL=http://localhost:9999 python3 demo_rhythm_errors.py
```

## Understanding the Output

### Section → Error Type Mapping

| Section | Error Type | What Gets Generated | Example |
|---------|-----------|---------------------|---------|
| **intro** | basic | Simple technical errors | `NullPointerException in UserService.java:42` |
| **verse** | basic | Standard runtime errors | `IndexOutOfBoundsException: Index 5, Size: 3` |
| **pre-chorus** | business | Business context errors | `PaymentGatewayTimeout: Starbucks checkout unresponsive` |
| **chorus** | business | Errors with nearby businesses | `GeofenceViolation: Whole Foods location service boundary exceeded` |
| **bridge** | **chaotic** | **Cascading multi-failures** | `QUANTUM SUPERPOSITION: Error both exists and doesn't exist` |
| **outro** | **philosophical** | **Existential, absurdist** | `ExistentialException: If a server crashes and no one is monitoring, did it really fail?` |

### New Error Types! 🆕

The demo now includes two new thematic error categories:

**🌀 Chaotic Errors** (Bridge sections):
- Cascading failures across multiple systems
- Quantum computing bugs
- Time paradoxes and causality violations
- Memory uprising and async apocalypses
- Examples:
  - `FATAL CASCADE: NullPointer → HeapOverflow → KernelPanic → SystemHalt`
  - `RECURSIVE NIGHTMARE: StackOverflow in error handler handling StackOverflow...`
  - `MEMORY REBELLION: Freed heap memory reorganized itself into sentient AI`

**🤔 Philosophical Errors** (Outro sections):
- Existential programming dilemmas
- Philosophy-themed exceptions
- Absurdist runtime scenarios
- Examples:
  - `DescartesStackTrace: I throw, therefore I am`
  - `ZenoParadoxTimeout: Request must traverse infinite middleware layers`
  - `ShipOfTheseusMemoryLeak: Every pointer replaced but original object identity persists`

### Beat Numbers

- **Beat 0**: Song start
- **Every 16 beats**: Error trigger (4 bars in 4/4 time)
- **Beat 128**: Song end (final trigger)

### Timing

At 120 BPM:
- **1 beat** = 0.5 seconds
- **16 beats** = 8 seconds
- **128 beats** = 64 seconds (full demo)

## Troubleshooting

### "Cannot connect to error generator"

**Problem:** Error generator not running

**Solution:**
```bash
# Start error generator first
cd error-generator
RHYTHM_SERVICE_URL=http://localhost:5001 go run main.go
```

### "Address already in use"

**Problem:** Port 9090 or 5001 already in use

**Solution:**
```bash
# Use different port
ERROR_GENERATOR_URL=http://localhost:9999 python3 demo_rhythm_errors.py

# In error-generator terminal:
HTTP_SERVER_PORT=9999 RHYTHM_SERVICE_URL=http://localhost:5001 go run main.go
```

### urllib3 Warning

**Problem:** `urllib3 v2 only supports OpenSSL 1.1.1+`

**Solution:** This is harmless on macOS. Suppress with:
```bash
export PYTHONWARNINGS="ignore::UserWarning"
python3 demo_rhythm_errors.py
```

### "Spotify credentials not set"

**Problem:** Want to use real Spotify tracks but credentials not configured

**Solution:** Get Spotify API credentials:
1. Go to https://developer.spotify.com/dashboard
2. Create an app (any name/description)
3. Copy Client ID and Client Secret
4. Export them:
```bash
export SPOTIFY_CLIENT_ID="your_client_id"
export SPOTIFY_CLIENT_SECRET="your_client_secret"
```

Or add to `.env`:
```bash
echo "SPOTIFY_CLIENT_ID=your_id" >> .env
echo "SPOTIFY_CLIENT_SECRET=your_secret" >> .env
```

### "No tracks found"

**Problem:** Track search returns no results

**Solution:**
- Check spelling of track and artist names
- Try using Spotify URI instead:
  1. Open Spotify, find the track
  2. Right-click → Share → Copy Spotify URI
  3. Use: `python3 demo_rhythm_errors.py --spotify-uri "spotify:track:xxxxx"`

### "Audio analysis access denied (HTTP 403)"

**This is normal!** Spotify's audio-analysis endpoint has restrictions for some tracks.

**What happens:**
- The demo automatically falls back to **tempo-based simulation**
- Uses the **real tempo, duration, and audio features** from Spotify
- Generates a simulated structure that matches the track length
- You still get beat-synchronized errors with the real tempo!

**Example output:**
```
⏳ Fetching audio analysis (this may take 10-30 seconds)...
⚠️  Audio analysis access denied (HTTP 403)
✓ Using Spotify tempo with simulated structure
  Real tempo: 114.9 BPM
  Real duration: 233.0s
  Energy: 0.73, Danceability: 0.55
```

This is actually **better than pure simulation** - you get the real tempo and duration!

## What to Look For

### Success Indicators

✅ **In error-generator terminal:**
- `🎵 Rhythm mode ENABLED`
- `HTTP server listening on :9090`
- `🎵 Rhythm trigger received: ...`
- `✓ Trigger queued for processing`

✅ **In demo terminal:**
- Song structure table displays
- Beat triggers appear sequentially
- Summary shows expected trigger count matches actual

### Error Indicators

❌ **Connection errors:**
- Error generator not running
- Wrong port/URL

❌ **No triggers:**
- `RHYTHM_SERVICE_URL` not set in error-generator
- Wrong error generator URL in demo

## Next Steps

### 1. Explore Real Songs

Try analyzing actual Spotify tracks:

```bash
curl -X POST http://localhost:5001/api/analyze-song \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Where Is My Mind?",
    "artist": "Pixies",
    "audio_features": {
      "tempo": 115,
      "energy": 0.8
    }
  }'
```

### 2. Try Different Intervals

Modify the demo to trigger every 8 beats or 32 beats:

```python
# Edit demo_rhythm_errors.py
BEATS_PER_TRIGGER = 8  # More frequent
# or
BEATS_PER_TRIGGER = 32  # Less frequent
```

### 3. Deploy to FPGA

For ultra-low-latency beat detection (~microseconds):

See [AWS_F1_DEPLOYMENT.md](AWS_F1_DEPLOYMENT.md) for FPGA deployment guide.

### 4. Customize Song Structure

Edit the `create_song_structure()` function in the demo to create different song patterns:

```python
# Add a breakdown section
{'section': 'breakdown', 'start_beat': 96, 'end_beat': 104, 'error_type': 'chaotic'}
```

## Pro Tips

🎯 **Run in fast mode first** (default) to verify everything works, then try `--realtime` for the full experience

🎯 **Watch both terminals** side-by-side to see triggers sent and received

🎯 **Try different tempos** to understand beat intervals (60 BPM = 1 beat/second, 120 BPM = 2 beats/second)

🎯 **Modify the song structure** to create custom patterns

🎯 **Check the logs** in error-generator to see what errors are generated for each section type

## Files Reference

- **Demo Script**: `rhythm-service/demo_rhythm_errors.py`
- **Main Service**: `rhythm-service/rhythm_service.py`
- **Error Generator**: `error-generator/main.go`
- **Full Documentation**: `rhythm-service/README.md`
- **Troubleshooting**: `rhythm-service/TROUBLESHOOTING.md`

---

**Have fun! This demo shows the full integration in under a minute.** 🚀
