# 🎵 Rhythm-Driven Error Demo - Quick Start

Experience the full rhythm-driven error generation system in action!

## What This Demo Does

The demo script simulates an entire song and generates errors every 16 beats (4 bars), demonstrating:

- 🎼 **Beat-synchronized error generation**
- 🎭 **Section-based error mapping** (verse → basic, chorus → business, bridge → chaotic, outro → philosophical)
- 🔗 **Full integration** between rhythm analysis and error logging
- 📊 **Real-time statistics** and progress tracking

## 5-Minute Quick Start

### Step 1: Start the Error Generator (Terminal 1)

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

### Step 2: Run the Demo (Terminal 2)

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

### Step 3: Check Error Logs (Terminal 1)

Back in Terminal 1 (error-generator), you'll see the received triggers:

```
🎵 Rhythm trigger received: verse section, beat 16, tempo 120.0 BPM
✓ Trigger queued for processing
Generating error type: basic

🎵 Rhythm trigger received: chorus section, beat 32, tempo 120.0 BPM
✓ Trigger queued for processing
Generating error type: business
...
```

## Demo Options

### Real-Time Mode

Simulate actual song playback with delays between beats:

```bash
python3 demo_rhythm_errors.py --realtime
```

This will take ~64 seconds (the full "song" duration) instead of running instantly.

### Custom Tempo

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

| Section | Error Type | What Gets Generated |
|---------|-----------|-------------------|
| **intro** | basic | Simple NullPointer, IndexOutOfBounds errors |
| **verse** | basic | Standard technical errors |
| **pre-chorus** | business | Errors with business context |
| **chorus** | business | Errors with nearby location data |
| **bridge** | chaotic | Multi-error bursts, rapid-fire |
| **outro** | philosophical | Errors with governing body references |

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
