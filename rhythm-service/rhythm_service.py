#!/usr/bin/env python3
"""
Rhythm-Driven Error Generator Service

This service analyzes music from Spotify in real-time and sends rhythmic
triggers to the error-generator based on beat detection and song structure.

Educational ML project demonstrating:
- Audio feature extraction with Librosa
- RNN/LSTM sequence modeling for beat detection
- Song structure analysis (verse/chorus/bridge)
- FPGA deployment via hls4ml for ultra-low-latency inference
- Real-time synchronization with music
"""

import os
import time
import logging
from flask import Flask, jsonify, request
import requests
from dotenv import load_dotenv

from beat_detector import BeatDetector
from song_structure import SongStructureAnalyzer
from hls4ml_interface import HLS4MLInference

# Load environment variables
load_dotenv()

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Initialize Flask app
app = Flask(__name__)

# Configuration
SPOTIFY_CLIENT_ID = os.getenv('SPOTIFY_CLIENT_ID')
SPOTIFY_CLIENT_SECRET = os.getenv('SPOTIFY_CLIENT_SECRET')
ERROR_GENERATOR_URL = os.getenv('ERROR_GENERATOR_URL', 'http://localhost:9090')
USE_FPGA = os.getenv('USE_FPGA', 'false').lower() == 'true'

# Initialize components
beat_detector = BeatDetector(use_fpga=USE_FPGA)
structure_analyzer = SongStructureAnalyzer(use_fpga=USE_FPGA)
hls4ml_inference = HLS4MLInference() if USE_FPGA else None


class RhythmController:
    """Controls error generation timing based on musical rhythm"""

    def __init__(self):
        self.current_song = None
        self.current_tempo = 120  # Default BPM
        self.current_section = "verse"
        self.beat_count = 0

    def analyze_song(self, song_data):
        """
        Analyze song features and structure

        Args:
            song_data: Dictionary with song metadata and audio features
        """
        logger.info(f"Analyzing song: {song_data.get('name')} by {song_data.get('artist')}")

        # Extract audio features
        audio_features = song_data.get('audio_features', {})
        self.current_tempo = audio_features.get('tempo', 120)

        logger.info(f"Detected tempo: {self.current_tempo} BPM")

        # Analyze song structure
        try:
            structure = structure_analyzer.analyze(song_data)
            logger.info(f"Song structure: {structure}")
            return structure
        except Exception as e:
            logger.error(f"Structure analysis failed: {e}")
            return None

    def get_error_type_for_section(self, section):
        """
        Map song sections to error types

        Args:
            section: Song section (verse, chorus, bridge, outro)

        Returns:
            Error pattern type
        """
        section_map = {
            "intro": "minimal",
            "verse": "basic",
            "chorus": "business",
            "bridge": "chaotic",
            "outro": "philosophical"
        }
        return section_map.get(section, "basic")

    def calculate_beat_interval(self):
        """Calculate time between beats in seconds"""
        return 60.0 / self.current_tempo

    def trigger_error(self, error_type, beat_num):
        """
        Send trigger to error-generator

        Args:
            error_type: Type of error pattern to generate
            beat_num: Current beat number in the song
        """
        try:
            payload = {
                "trigger": "rhythm",
                "error_type": error_type,
                "beat": beat_num,
                "section": self.current_section,
                "tempo": self.current_tempo
            }

            response = requests.post(
                f"{ERROR_GENERATOR_URL}/api/rhythm-trigger",
                json=payload,
                timeout=2
            )

            if response.status_code == 200:
                result = response.json()
                logger.info(f"✓ Triggered {error_type} error on beat {beat_num}: {result.get('message')}")
            else:
                logger.warning(f"Error generator responded with {response.status_code}")

        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to trigger error: {e}")


# Global rhythm controller
rhythm_controller = RhythmController()


@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint"""
    return jsonify({
        "status": "healthy",
        "service": "rhythm-service",
        "fpga_enabled": USE_FPGA
    })


@app.route('/api/analyze-song', methods=['POST'])
def analyze_song():
    """
    Analyze a song and return rhythm information

    Expected payload:
    {
        "name": "Song Title",
        "artist": "Artist Name",
        "spotify_id": "track_id",
        "audio_features": {...}
    }
    """
    try:
        song_data = request.get_json()

        if not song_data:
            return jsonify({"error": "No song data provided"}), 400

        # Analyze the song
        structure = rhythm_controller.analyze_song(song_data)

        return jsonify({
            "success": True,
            "tempo": rhythm_controller.current_tempo,
            "beat_interval": rhythm_controller.calculate_beat_interval(),
            "structure": structure
        })

    except Exception as e:
        logger.error(f"Error analyzing song: {e}")
        return jsonify({"error": str(e)}), 500


@app.route('/api/start-rhythm-mode', methods=['POST'])
def start_rhythm_mode():
    """
    Start rhythm-driven error generation

    This will analyze the current song and begin sending
    beat-synchronized triggers to the error generator
    """
    try:
        data = request.get_json()
        song_data = data.get('song_data', {})
        duration = data.get('duration', 180)  # Default 3 minutes

        logger.info("🎵 Starting rhythm-driven error generation...")

        # Analyze song structure
        structure = rhythm_controller.analyze_song(song_data)
        beat_interval = rhythm_controller.calculate_beat_interval()

        # Simulate real-time beat detection
        # In production, this would stream audio and detect beats in real-time
        response_data = {
            "success": True,
            "message": "Rhythm mode started",
            "tempo": rhythm_controller.current_tempo,
            "beat_interval": beat_interval,
            "duration": duration
        }

        # Note: In a real implementation, this would spawn a background task
        # to continuously monitor the music and send triggers

        return jsonify(response_data)

    except Exception as e:
        logger.error(f"Error starting rhythm mode: {e}")
        return jsonify({"error": str(e)}), 500


@app.route('/api/beat-trigger', methods=['POST'])
def beat_trigger():
    """
    Manual beat trigger endpoint for testing

    Payload:
    {
        "section": "verse|chorus|bridge|outro",
        "beat_num": 1
    }
    """
    try:
        data = request.get_json()
        section = data.get('section', 'verse')
        beat_num = data.get('beat_num', 1)

        rhythm_controller.current_section = section
        error_type = rhythm_controller.get_error_type_for_section(section)
        rhythm_controller.trigger_error(error_type, beat_num)

        return jsonify({
            "success": True,
            "triggered": True,
            "error_type": error_type,
            "section": section,
            "beat": beat_num
        })

    except Exception as e:
        logger.error(f"Error processing beat trigger: {e}")
        return jsonify({"error": str(e)}), 500


@app.route('/api/status', methods=['GET'])
def status():
    """Get current rhythm service status"""
    return jsonify({
        "service": "rhythm-service",
        "current_song": rhythm_controller.current_song,
        "tempo": rhythm_controller.current_tempo,
        "current_section": rhythm_controller.current_section,
        "beat_count": rhythm_controller.beat_count,
        "fpga_enabled": USE_FPGA,
        "beat_interval_ms": rhythm_controller.calculate_beat_interval() * 1000
    })


def main():
    """Main entry point"""
    logger.info("🎼 Rhythm Service Starting...")
    logger.info(f"FPGA Mode: {'ENABLED' if USE_FPGA else 'DISABLED (CPU Simulation)'}")
    logger.info(f"Error Generator URL: {ERROR_GENERATOR_URL}")

    if not SPOTIFY_CLIENT_ID or not SPOTIFY_CLIENT_SECRET:
        logger.warning("⚠️  Spotify credentials not set. Some features may be limited.")

    # Start Flask server
    port = int(os.getenv('PORT', 5000))
    app.run(host='0.0.0.0', port=port, debug=False)


if __name__ == '__main__':
    main()
