#!/usr/bin/env python3
"""
Demo: Rhythm-Driven Error Generation

Simulates an entire song and triggers errors every 16 beats,
demonstrating the integration between rhythm analysis and error generation.

Usage:
    python3 demo_rhythm_errors.py

    # Or customize:
    ERROR_GENERATOR_URL=http://localhost:9090 TEMPO=128 python3 demo_rhythm_errors.py
"""

import os
import time
import requests
import logging
from typing import List, Dict, Tuple

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Configuration
ERROR_GENERATOR_URL = os.getenv('ERROR_GENERATOR_URL', 'http://localhost:9090')
TEMPO = int(os.getenv('TEMPO', '120'))  # BPM
BEATS_PER_TRIGGER = 16  # Trigger error every 16 beats (4 bars in 4/4 time)


class SongSimulator:
    """Simulates a song structure with beats and sections"""

    def __init__(self, tempo: int = 120):
        self.tempo = tempo
        self.beat_duration = 60.0 / tempo  # seconds per beat

    def create_song_structure(self) -> List[Dict]:
        """
        Create a typical pop song structure.
        Returns list of sections with their beat ranges.
        """
        structure = [
            # Intro: 8 beats (2 bars)
            {'section': 'intro', 'start_beat': 0, 'end_beat': 8, 'error_type': 'basic'},

            # Verse 1: 16 beats (4 bars)
            {'section': 'verse', 'start_beat': 8, 'end_beat': 24, 'error_type': 'basic'},

            # Pre-Chorus: 8 beats (2 bars)
            {'section': 'pre-chorus', 'start_beat': 24, 'end_beat': 32, 'error_type': 'business'},

            # Chorus: 16 beats (4 bars)
            {'section': 'chorus', 'start_beat': 32, 'end_beat': 48, 'error_type': 'business'},

            # Verse 2: 16 beats (4 bars)
            {'section': 'verse', 'start_beat': 48, 'end_beat': 64, 'error_type': 'basic'},

            # Pre-Chorus: 8 beats (2 bars)
            {'section': 'pre-chorus', 'start_beat': 64, 'end_beat': 72, 'error_type': 'business'},

            # Chorus: 16 beats (4 bars)
            {'section': 'chorus', 'start_beat': 72, 'end_beat': 88, 'error_type': 'business'},

            # Bridge: 16 beats (4 bars)
            {'section': 'bridge', 'start_beat': 88, 'end_beat': 104, 'error_type': 'chaotic'},

            # Final Chorus: 16 beats (4 bars)
            {'section': 'chorus', 'start_beat': 104, 'end_beat': 120, 'error_type': 'business'},

            # Outro: 8 beats (2 bars)
            {'section': 'outro', 'start_beat': 120, 'end_beat': 128, 'error_type': 'philosophical'},
        ]

        return structure

    def get_section_at_beat(self, structure: List[Dict], beat_num: int) -> Dict:
        """Get the section info for a given beat number"""
        for section in structure:
            if section['start_beat'] <= beat_num < section['end_beat']:
                return section
        return structure[-1]  # Default to last section

    def calculate_song_duration(self, structure: List[Dict]) -> Tuple[int, float]:
        """Calculate total beats and duration"""
        total_beats = structure[-1]['end_beat']
        total_duration = total_beats * self.beat_duration
        return total_beats, total_duration


class RhythmErrorTrigger:
    """Triggers errors based on rhythm analysis"""

    def __init__(self, error_generator_url: str):
        self.error_generator_url = error_generator_url
        self.trigger_count = 0

    def check_service_health(self) -> bool:
        """Check if error generator is running"""
        try:
            response = requests.get(f"{self.error_generator_url}/health", timeout=2)
            if response.status_code == 200:
                logger.info("✓ Error generator service is healthy")
                return True
            else:
                logger.warning(f"⚠️  Error generator returned status {response.status_code}")
                return False
        except requests.exceptions.RequestException as e:
            logger.error(f"❌ Cannot connect to error generator at {self.error_generator_url}")
            logger.error(f"   Error: {e}")
            logger.error(f"   Make sure error-generator is running on port 9090")
            return False

    def trigger_error(self, beat: int, section: str, error_type: str, tempo: float):
        """Send rhythm trigger to error generator"""
        try:
            payload = {
                "trigger": "rhythm",
                "error_type": error_type,
                "beat": beat,
                "section": section,
                "tempo": tempo
            }

            response = requests.post(
                f"{self.error_generator_url}/api/rhythm-trigger",
                json=payload,
                timeout=5
            )

            if response.status_code == 200:
                self.trigger_count += 1
                logger.info(
                    f"🎵 Beat {beat:3d} | {section:12s} | {error_type:14s} | "
                    f"Trigger #{self.trigger_count}"
                )
                return True
            else:
                logger.warning(f"⚠️  Trigger failed with status {response.status_code}")
                return False

        except requests.exceptions.RequestException as e:
            logger.error(f"❌ Failed to trigger error: {e}")
            return False


def print_song_structure(structure: List[Dict], tempo: int, beat_duration: float):
    """Print a visual representation of the song structure"""
    print("\n" + "="*70)
    print("🎵 SONG STRUCTURE")
    print("="*70)
    print(f"Tempo: {tempo} BPM | Beat Duration: {beat_duration:.3f}s")
    print(f"Error Trigger Interval: Every {BEATS_PER_TRIGGER} beats (4 bars)")
    print("-"*70)
    print(f"{'Section':<15} {'Beats':<15} {'Duration':<15} {'Error Type'}")
    print("-"*70)

    for section in structure:
        beat_count = section['end_beat'] - section['start_beat']
        duration = beat_count * beat_duration
        beats_range = f"{section['start_beat']}-{section['end_beat']}"

        print(
            f"{section['section']:<15} "
            f"{beats_range:<15} "
            f"{duration:>6.1f}s        "
            f"{section['error_type']}"
        )

    total_beats = structure[-1]['end_beat']
    total_duration = total_beats * beat_duration
    expected_triggers = total_beats // BEATS_PER_TRIGGER

    print("-"*70)
    print(f"{'TOTAL':<15} {total_beats:<15} {total_duration:>6.1f}s")
    print(f"Expected Error Triggers: {expected_triggers}")
    print("="*70 + "\n")


def run_demo(simulate_realtime: bool = False):
    """
    Run the rhythm-driven error generation demo.

    Args:
        simulate_realtime: If True, delays between beats to simulate real playback
    """
    print("\n" + "🎼"*35)
    print("  RHYTHM-DRIVEN ERROR GENERATOR DEMO")
    print("🎼"*35 + "\n")

    # Initialize components
    simulator = SongSimulator(tempo=TEMPO)
    trigger = RhythmErrorTrigger(error_generator_url=ERROR_GENERATOR_URL)

    # Check if error generator is running
    if not trigger.check_service_health():
        print("\n" + "⚠️ "*30)
        print("ERROR GENERATOR NOT RUNNING!")
        print("\nTo start it:")
        print("  cd ../error-generator")
        print("  RHYTHM_SERVICE_URL=http://localhost:5001 go run main.go")
        print("⚠️ "*30 + "\n")

        response = input("Continue anyway (errors won't be generated)? [y/N]: ")
        if response.lower() != 'y':
            print("Demo cancelled.")
            return

    # Create song structure
    structure = simulator.create_song_structure()
    total_beats, total_duration = simulator.calculate_song_duration(structure)

    # Print song structure
    print_song_structure(structure, TEMPO, simulator.beat_duration)

    # Ask user to confirm
    if simulate_realtime:
        print(f"⏱️  Real-time mode: Demo will take {total_duration:.1f} seconds")
    else:
        print("⚡ Fast mode: Demo will run as fast as possible")

    print("\nPress Enter to start, or Ctrl+C to cancel...")
    try:
        input()
    except KeyboardInterrupt:
        print("\nDemo cancelled.")
        return

    print("\n" + "▶️ "*35)
    print("  STARTING DEMO")
    print("▶️ "*35 + "\n")

    # Run through all beats
    start_time = time.time()
    triggers_sent = 0

    try:
        for beat in range(total_beats):
            # Get current section
            section_info = simulator.get_section_at_beat(structure, beat)

            # Trigger error every 16 beats
            if beat > 0 and beat % BEATS_PER_TRIGGER == 0:
                success = trigger.trigger_error(
                    beat=beat,
                    section=section_info['section'],
                    error_type=section_info['error_type'],
                    tempo=TEMPO
                )
                if success:
                    triggers_sent += 1

            # Simulate real-time playback if requested
            if simulate_realtime:
                time.sleep(simulator.beat_duration)

        # Trigger one final error at the end
        final_section = structure[-1]
        trigger.trigger_error(
            beat=total_beats,
            section=final_section['section'],
            error_type=final_section['error_type'],
            tempo=TEMPO
        )
        triggers_sent += 1

    except KeyboardInterrupt:
        print("\n\n⏸️  Demo interrupted by user")

    # Print summary
    elapsed_time = time.time() - start_time

    print("\n" + "="*70)
    print("📊 DEMO SUMMARY")
    print("="*70)
    print(f"Song Duration:       {total_duration:.1f}s ({total_beats} beats)")
    print(f"Execution Time:      {elapsed_time:.1f}s")
    print(f"Triggers Sent:       {triggers_sent}")
    print(f"Expected Triggers:   {(total_beats // BEATS_PER_TRIGGER) + 1}")
    print(f"Tempo:               {TEMPO} BPM")
    print(f"Trigger Interval:    Every {BEATS_PER_TRIGGER} beats")
    print("="*70 + "\n")

    print("✅ Demo complete! Check your error logs for the generated errors.\n")


def main():
    """Main entry point"""
    import sys

    # Check for command line arguments
    realtime = '--realtime' in sys.argv or '-r' in sys.argv

    if '--help' in sys.argv or '-h' in sys.argv:
        print(__doc__)
        print("\nOptions:")
        print("  --realtime, -r    Simulate real-time playback (delays between beats)")
        print("  --help, -h        Show this help message")
        print("\nEnvironment Variables:")
        print("  ERROR_GENERATOR_URL   URL of error generator (default: http://localhost:9090)")
        print("  TEMPO                 Song tempo in BPM (default: 120)")
        return

    run_demo(simulate_realtime=realtime)


if __name__ == '__main__':
    main()
