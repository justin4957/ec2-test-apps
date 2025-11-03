# Rhythm-Driven Error Generator Service

Educational machine learning project that synchronizes error generation with music using FPGA-accelerated neural networks.

![Beat Detection](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExcW50ZGp6Y2pmZjB0dTBkYnJnMzBidHhlZ2Q1eDd6OGdtNDdqYmJodyZlcD12MV9naWZzX3NlYXJjaCZjdD1n/l0HlRnAWXxn28/giphy.gif)

## Overview

This service transforms error logging into a rhythmic performance by:
- **Detecting beats** in real-time using RNN/LSTM models
- **Analyzing song structure** (verse/chorus/bridge/outro)
- **Triggering error patterns** synchronized to musical sections
- **Deploying models to FPGA** via hls4ml for ultra-low-latency inference (~microseconds)

## Educational Value

Learn about:
- **Audio ML**: Feature extraction, onset detection, beat tracking
- **Sequence Modeling**: RNN/LSTM for temporal pattern recognition
- **FPGA Deployment**: Model quantization and hardware synthesis with hls4ml
- **Real-time Systems**: Low-latency inference for musical synchronization
- **System Integration**: Coordinating multiple services with HTTP APIs

![ML](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExMmM0cTNrbDR0OXo3NGRoMWhoeDE1N3oyeDB3dnR6NXdzeWhweHlteCZlcD12MV9naWZzX3NlYXJjaCZjdD1n/3oKIPEqDGUULpEU0aQ/giphy.gif)

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│              Rhythm Service (Python)                    │
│                                                         │
│  ┌──────────────┐      ┌──────────────┐               │
│  │ Spotify API  │──────>│ Beat         │               │
│  │ Audio        │      │ Detector     │               │
│  │ Features     │      │ (RNN/LSTM)   │               │
│  └──────────────┘      └──────┬───────┘               │
│                               │                         │
│  ┌──────────────┐      ┌─────▼────────┐               │
│  │ Song         │      │ HLS4ML       │               │
│  │ Structure    │──────>│ FPGA         │               │
│  │ Analyzer     │      │ Inference    │               │
│  └──────────────┘      └──────┬───────┘               │
│                               │                         │
│                        ┌──────▼───────┐                │
│                        │ HTTP Trigger │                │
│                        │ Controller   │                │
│                        └──────┬───────┘                │
└───────────────────────────────┼─────────────────────────┘
                                │
                         HTTP POST
                                │
                      ┌─────────▼──────────┐
                      │ Error Generator    │
                      │ (Go)               │
                      │                    │
                      │ /api/rhythm-trigger│
                      └────────────────────┘
```

## Song Section → Error Pattern Mapping

The service maps musical sections to specific error patterns:

| Song Section | Error Pattern | Description |
|-------------|---------------|-------------|
| **Intro** | Minimal | Sparse, simple errors to set the mood |
| **Verse** | Basic | Standard technical errors (NullPointer, IndexOutOfBounds) |
| **Chorus** | Business | Business-related errors with nearby location data |
| **Bridge** | Chaotic | Multi-error bursts, rapid-fire absurdity |
| **Outro** | Philosophical | Errors with governing body references |

![Sections](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExZnY1N2o3ZGI3dTNmYnNuOGJqM2oxYWgzeXRhdnI1ZHN4cXd3Z2Y5aiZlcD12MV9naWZzX3NlYXJjaCZjdD1n/xT5LMGupUKCHb7DnFu/giphy.gif)

## Quick Start

### Prerequisites

- Python 3.8+
- Spotify Developer Account (for API access)
- (Optional) FPGA board for hardware acceleration

### Installation

1. Install dependencies:
```bash
cd rhythm-service
pip install -r requirements.txt
```

2. Set up environment variables:
```bash
cp .env.example .env
# Edit .env and add:
# - SPOTIFY_CLIENT_ID
# - SPOTIFY_CLIENT_SECRET
# - ERROR_GENERATOR_URL
```

3. Run the service:
```bash
python rhythm_service.py
```

The service will start on port 5000 by default.

### Testing

Test the health endpoint:
```bash
curl http://localhost:5000/health
```

Trigger a manual beat:
```bash
curl -X POST http://localhost:5000/api/beat-trigger \
  -H "Content-Type: application/json" \
  -d '{"section": "chorus", "beat_num": 1}'
```

Analyze a song:
```bash
curl -X POST http://localhost:5000/api/analyze-song \
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

## Components

### Beat Detector (`beat_detector.py`)

Detects beats using:
- **Librosa** for onset detection and tempo estimation
- **RNN/LSTM model** for temporal pattern recognition
- **FPGA acceleration** via hls4ml (optional)

```python
from beat_detector import BeatDetector

detector = BeatDetector(use_fpga=True)
beats, tempo = detector.detect_beats_from_audio(audio_data, sr=22050)
```

### Song Structure Analyzer (`song_structure.py`)

Identifies song sections using:
- **Spotify's audio analysis API** (when available)
- **Self-similarity matrix** analysis
- **Heuristic estimation** based on audio features

```python
from song_structure import SongStructureAnalyzer

analyzer = SongStructureAnalyzer(use_fpga=True)
structure = analyzer.analyze(song_data)

# Get section at specific time
section = analyzer.get_section_at_time(structure, 45.0)  # "chorus"
```

### HLS4ML Interface (`hls4ml_interface.py`)

Handles FPGA deployment:
- **QKeras** for model quantization
- **hls4ml** for HLS C++ code generation
- **Synthesis** for FPGA bitstream generation
- **Benchmarking** CPU vs FPGA performance

```python
from hls4ml_interface import HLS4MLInference

interface = HLS4MLInference()

# Create quantized model
model = interface.create_quantized_model(input_shape=(10, 3))

# Convert to HLS
hls_model = interface.convert_to_hls(model, output_dir='beat_hls')

# Synthesize for FPGA
report = interface.synthesize_fpga_design(clock_period=5)

# Benchmark
results = interface.benchmark(test_data, num_iterations=100)
# Expected: 1000-5000x speedup!
```

## FPGA Deployment Workflow

![FPGA](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExZ2RrYnY3NWMyeWExMTc2aWF5cnRocnRhbW5renRvY2VsN2g2MzNjMSZlcD12MV9naWZzX3NlYXJjaCZjdD1n/l378BbHA2eQCQ5jDa/giphy.gif)

### Step 1: Train Model

```python
from beat_detector import BeatDetector, prepare_training_data

# Prepare training data
audio_files = [
    ('audio1.wav', 'beats1.txt'),
    ('audio2.wav', 'beats2.txt'),
]

X_train, y_train = prepare_training_data(audio_files)

# Create and train model
detector = BeatDetector()
model = detector.create_quantized_model(input_shape=X_train.shape[1:])
model.fit(X_train, y_train, epochs=50, batch_size=32)
```

### Step 2: Convert to HLS

```python
from hls4ml_interface import HLS4MLInference

interface = HLS4MLInference()
hls_model = interface.convert_to_hls(model, output_dir='beat_hls')
```

This generates C++ code that can run on FPGA!

### Step 3: Synthesize

```python
# Synthesis (takes 5-15 minutes)
report = interface.synthesize_fpga_design(clock_period=5)

print(f"Latency: {report['LatencyMin']} cycles")
print(f"Resources: {report['BRAM']}, {report['DSP']}, {report['FF']}, {report['LUT']}")
```

### Step 4: Deploy

Upload the synthesized design to your FPGA board and run inference at microsecond latency!

## API Endpoints

### `GET /health`
Health check endpoint

**Response:**
```json
{
  "status": "healthy",
  "service": "rhythm-service",
  "fpga_enabled": true
}
```

### `POST /api/analyze-song`
Analyze song structure and rhythm

**Request:**
```json
{
  "name": "Song Title",
  "artist": "Artist Name",
  "audio_features": {
    "tempo": 120,
    "energy": 0.7,
    "danceability": 0.6
  }
}
```

**Response:**
```json
{
  "success": true,
  "tempo": 120.0,
  "beat_interval": 0.5,
  "structure": [
    {"start": 0, "duration": 30, "type": "verse"},
    {"start": 30, "duration": 20, "type": "chorus"}
  ]
}
```

### `POST /api/start-rhythm-mode`
Start rhythm-driven error generation

**Request:**
```json
{
  "song_data": {
    "name": "Song",
    "audio_features": {...}
  },
  "duration": 180
}
```

### `POST /api/beat-trigger`
Manual beat trigger (for testing)

**Request:**
```json
{
  "section": "chorus",
  "beat_num": 1
}
```

## Integration with Error Generator

The rhythm service communicates with the error-generator via HTTP:

```
POST http://error-generator:9090/api/rhythm-trigger
{
  "trigger": "rhythm",
  "error_type": "business",
  "beat": 16,
  "section": "chorus",
  "tempo": 125
}
```

The error-generator will then immediately generate an error synchronized to the beat!

## Performance Benchmarks

### CPU Inference (TensorFlow)
- **Latency**: 10-50ms
- **Throughput**: 20-100 inferences/second
- **Good for**: Non-real-time analysis

### FPGA Inference (hls4ml)
- **Latency**: 1-10μs (microseconds!)
- **Throughput**: 100,000+ inferences/second
- **Good for**: Real-time beat detection

**Speedup**: 1000-5000x faster than CPU!

![Fast](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExOWd0ZWx3cGoxbWJsMWxwZ2JvYmczM3Y1Y3RqbGxlOWpvNW9rZWFwZyZlcD12MV9naWZzX3NlYXJjaCZjdD1n/3oriO5t2QB4IPKgxHi/giphy.gif)

## Educational Resources

### Machine Learning for Audio
- [Librosa Documentation](https://librosa.org/doc/latest/index.html)
- [Audio Feature Extraction](https://haythamfayek.com/2016/04/21/speech-processing-for-machine-learning.html)
- [Music Information Retrieval](https://www.audiolabs-erlangen.de/resources/MIR)

### FPGA and Hardware Acceleration
- [hls4ml Tutorial](https://fastmachinelearning.org/hls4ml/api/configuration.html)
- [QKeras Documentation](https://github.com/google/qkeras)
- [Xilinx Vitis HLS](https://www.xilinx.com/products/design-tools/vitis/vitis-hls.html)

### Research Papers
- [Beat Tracking with RNNs](https://archives.ismir.net/ismir2018/paper/000188.pdf)
- [Music Structure Analysis](https://transactions.ismir.net/articles/10.5334/tismir.31/)
- [Neural Network Quantization](https://arxiv.org/abs/1906.04721)

## Troubleshooting

### Librosa installation issues
```bash
# On macOS
brew install libsndfile
pip install librosa

# On Ubuntu
sudo apt-get install libsndfile1
pip install librosa
```

### FPGA synthesis taking too long
Synthesis can take 5-15 minutes. For development, use CPU mode:
```bash
export USE_FPGA=false
python rhythm_service.py
```

### Spotify API rate limiting
The Spotify API has rate limits. Cache audio features locally:
```python
import json

# Save features
with open('song_features.json', 'w') as f:
    json.dump(audio_features, f)
```

## Future Enhancements

- **Real-time audio streaming** from Spotify
- **Multiple FPGA boards** for parallel processing
- **Adaptive beat detection** that learns from user feedback
- **Visualization dashboard** showing beats and structure
- **Cloud FPGA** deployment (AWS F1 instances)

![Future](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExaHRpazJ6cHc5eGx3aGFvbW5uc3BvZWRjYnliZTJjYnI1NXNwZzRvNyZlcD12MV9naWZzX3NlYXJjaCZjdD1n/26AHONQ79FdWZhAI0/giphy.gif)

## License

Educational and testing purposes only.
