# Rhythm-Driven Error Generator - Quick Start Guide

Congratulations! You've just set up an educational ML project that synchronizes error generation with music using FPGA-accelerated neural networks.

![Success](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExMXpqeGNlOWZxdGhvOXB0eDBhemFjYWhlYnl5Y3hvMmp6aGRncTZ5YiZlcD12MV9naWZzX3NlYXJjaCZjdD1n/3ohzdIuqJoo8QdKlnW/giphy.gif)

## What You've Built

### Enhanced Main README
- Fun GIFs throughout the documentation
- Complete "Silver Screen Static" playlist with 100+ tracks
- Song recommendations organized into themed "Acts"
- Documentation for the new rhythm-driven feature
- Educational resources and links

### New Rhythm Service (`rhythm-service/`)

A complete Python ML service with:

1. **`rhythm_service.py`** - Main Flask HTTP service
   - Analyzes songs from Spotify
   - Triggers errors synchronized to beats
   - Communicates with Go error-generator

2. **`beat_detector.py`** - Beat detection module
   - Librosa-based onset detection
   - RNN/LSTM for temporal patterns
   - FPGA acceleration support

3. **`song_structure.py`** - Song structure analysis
   - Identifies verse/chorus/bridge/outro
   - Maps sections to error patterns
   - Self-similarity matrix analysis

4. **`hls4ml_interface.py`** - FPGA deployment
   - QKeras model quantization
   - HLS C++ code generation
   - Performance benchmarking
   - 1000-5000x speedup potential!

5. **`example_usage.py`** - Educational examples
   - Demonstrates all features
   - Shows complete workflow
   - Perfect for learning

6. **Supporting Files**
   - `requirements.txt` - Python dependencies
   - `Dockerfile` - Container configuration
   - `.env.example` - Environment template
   - `README.md` - Comprehensive documentation

## Getting Started

### 1. Install Python Dependencies

```bash
cd rhythm-service
pip install -r requirements.txt
```

This will install:
- TensorFlow (ML framework)
- Librosa (audio analysis)
- hls4ml (FPGA deployment)
- Flask (HTTP server)
- Spotipy (Spotify API)

### 2. Set Up Environment

```bash
cp .env.example .env
```

Edit `.env` and add your Spotify credentials:
```bash
SPOTIFY_CLIENT_ID=your_client_id
SPOTIFY_CLIENT_SECRET=your_client_secret
ERROR_GENERATOR_URL=http://localhost:9090
USE_FPGA=false  # Set to true if you have FPGA hardware
```

Get Spotify credentials at: https://developer.spotify.com/dashboard

### 3. Run Examples

```bash
python example_usage.py
```

This will show you:
- Beat detection from Spotify features
- Song structure analysis
- Error pattern mapping
- Rhythm-driven triggering
- FPGA deployment workflow

### 4. Start the Service

```bash
python rhythm_service.py
```

The service starts on port 5000. Test it:

```bash
# Health check
curl http://localhost:5000/health

# Analyze a song
curl -X POST http://localhost:5000/api/analyze-song \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Where Is My Mind?",
    "artist": "Pixies",
    "audio_features": {
      "tempo": 115,
      "energy": 0.8
    }
  }'

# Trigger a beat
curl -X POST http://localhost:5000/api/beat-trigger \
  -H "Content-Type: application/json" \
  -d '{"section": "chorus", "beat_num": 1}'
```

### 5. (Optional) Build Docker Container

```bash
docker build -t rhythm-service .
docker run -p 5000:5000 --env-file .env rhythm-service
```

## How It Works

### Song Section → Error Pattern Flow

1. **Music Analysis**
   - Service receives song from Spotify API
   - Extracts tempo, energy, structure
   - Identifies sections (verse/chorus/bridge)

2. **Beat Detection**
   - RNN/LSTM model detects beats
   - Optional FPGA acceleration (1-10μs latency!)
   - Calculates precise timing

3. **Error Triggering**
   - Maps section to error type:
     - Verse → Basic errors
     - Chorus → Business errors
     - Bridge → Chaotic bursts
     - Outro → Philosophical errors
   - Sends HTTP trigger to error-generator

4. **Error Generation**
   - Go error-generator receives trigger
   - Generates appropriate error
   - Pairs with GIF and song from playlist
   - Logs to location-tracker

## Educational Value

This project teaches:

### Audio Machine Learning
- Feature extraction with Librosa
- Onset detection and beat tracking
- Spectral analysis (chroma, MFCC)
- Temporal sequence modeling

### Deep Learning
- RNN/LSTM architectures
- Sequence-to-sequence models
- Model training and evaluation
- Transfer learning

### FPGA Deployment
- Neural network quantization
- QKeras fixed-point models
- HLS C++ code generation
- Hardware synthesis
- Performance optimization

### System Integration
- REST API design
- Microservices architecture
- Real-time data processing
- Event-driven systems

## Next Steps

### Short Term (Learn the Basics)
1. Run all examples and understand the output
2. Analyze different songs from your playlist
3. Experiment with different error patterns
4. Visualize song structures

### Medium Term (Model Training)
1. Collect beat-annotated audio datasets
2. Train your own beat detection model
3. Evaluate model performance
4. Fine-tune hyperparameters

### Long Term (FPGA Deployment)
1. Acquire FPGA development board
2. Install Xilinx Vitis or Intel Quartus
3. Synthesize your trained model
4. Deploy and benchmark on hardware
5. Achieve microsecond latency!

## Resources

### Datasets
- [Ballroom Dataset](http://mtg.upf.edu/ismir2004/contest/tempoContest/node5.html) - Beat annotations
- [Million Song Dataset](http://millionsongdataset.com/) - Large-scale audio features
- [GTZAN](http://marsyas.info/downloads/datasets.html) - Music genre dataset

### Documentation
- [Librosa Tutorials](https://librosa.org/doc/latest/tutorial.html)
- [hls4ml Documentation](https://fastmachinelearning.org/hls4ml/)
- [Spotify Web API](https://developer.spotify.com/documentation/web-api)
- [TensorFlow Guide](https://www.tensorflow.org/guide)

### Research Papers
- [Beat Tracking with Deep Learning](https://archives.ismir.net/ismir2018/paper/000188.pdf)
- [Music Structure Analysis](https://transactions.ismir.net/articles/10.5334/tismir.31/)
- [Neural Network Quantization](https://arxiv.org/abs/1906.04721)
- [FPGA Acceleration](https://arxiv.org/abs/1904.08986)

## Troubleshooting

**"ModuleNotFoundError: No module named 'librosa'"**
```bash
pip install librosa
# On macOS: brew install libsndfile
# On Ubuntu: sudo apt-get install libsndfile1
```

**"FPGA synthesis takes forever"**
- Normal! Synthesis takes 5-15 minutes
- Use CPU mode during development: `USE_FPGA=false`

**"Spotify API rate limiting"**
- Cache audio features locally
- Use the built-in playlist data
- Wait before retrying

## Contributing Ideas

Want to extend this project? Try:
- Real-time audio streaming (not just Spotify features)
- Visualization dashboard with D3.js
- Multiple FPGA boards for parallel processing
- Cloud FPGA deployment (AWS F1)
- Mobile app integration
- Live performance mode

## Summary

You now have:
- ✅ Comprehensive README with GIFs and song recommendations
- ✅ Complete Python rhythm service
- ✅ Beat detection with RNN/LSTM
- ✅ Song structure analysis
- ✅ FPGA deployment framework
- ✅ HTTP API for integration
- ✅ Educational examples
- ✅ Docker support
- ✅ Full documentation

Have fun syncing errors to music! 🎵🚀

![Party](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExb3dzZnE2cXRhbWZqbWFudGRjMGlub3Zua2R0ZTRkYTY3ZHk3cTV0YSZlcD12MV9naWZzX3NlYXJjaCZjdD1n/l0MYt5jPR6QX5pnqM/giphy.gif)
