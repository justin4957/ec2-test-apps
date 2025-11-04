#!/usr/bin/env python3
"""
Demo: Rhythm-Driven Error Generation

Simulates an entire song and triggers errors every 16 beats,
demonstrating the integration between rhythm analysis and error generation.

Usage:
    # Simulated song
    python3 demo_rhythm_errors.py

    # Use a real Spotify track
    python3 demo_rhythm_errors.py --track "Where Is My Mind?" --artist "Pixies"

    # Or by Spotify URI
    python3 demo_rhythm_errors.py --spotify-uri "spotify:track:5EWPGh7jbTNO2wakv8LjUI"

    # Customize:
    ERROR_GENERATOR_URL=http://localhost:9090 TEMPO=128 python3 demo_rhythm_errors.py
"""

import os
import sys
import time
import base64
import requests
import logging
from typing import List, Dict, Tuple, Optional

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Configuration
ERROR_GENERATOR_URL = os.getenv('ERROR_GENERATOR_URL', 'http://localhost:9090')
SLOGAN_SERVER_URL = os.getenv('SLOGAN_SERVER_URL', 'http://localhost:8080')
SPOTIFY_CLIENT_ID = os.getenv('SPOTIFY_CLIENT_ID', '')
SPOTIFY_CLIENT_SECRET = os.getenv('SPOTIFY_CLIENT_SECRET', '')
TEMPO = int(os.getenv('TEMPO', '120'))  # BPM
BEATS_PER_TRIGGER = 16  # Trigger error every 16 beats (4 bars in 4/4 time)


class SpotifyClient:
    """Client for Spotify API to fetch real track data"""

    def __init__(self, client_id: str, client_secret: str):
        self.client_id = client_id
        self.client_secret = client_secret
        self.access_token = None
        self.token_expires = 0

    def authenticate(self) -> bool:
        """Authenticate with Spotify API"""
        if not self.client_id or not self.client_secret:
            logger.error("Spotify credentials not set")
            logger.error("Set SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET environment variables")
            return False

        try:
            # Encode credentials
            auth_str = f"{self.client_id}:{self.client_secret}"
            auth_bytes = auth_str.encode('utf-8')
            auth_base64 = base64.b64encode(auth_bytes).decode('utf-8')

            # Request token
            response = requests.post(
                'https://accounts.spotify.com/api/token',
                headers={
                    'Authorization': f'Basic {auth_base64}',
                    'Content-Type': 'application/x-www-form-urlencoded'
                },
                data={'grant_type': 'client_credentials'},
                timeout=10
            )

            if response.status_code == 200:
                data = response.json()
                self.access_token = data['access_token']
                self.token_expires = time.time() + data['expires_in']
                logger.info("✓ Authenticated with Spotify API")
                return True
            else:
                logger.error(f"Spotify auth failed: {response.status_code}")
                return False

        except Exception as e:
            logger.error(f"Spotify authentication error: {e}")
            return False

    def search_track(self, track_name: str, artist_name: str = '') -> Optional[Dict]:
        """Search for a track on Spotify"""
        if not self.access_token or time.time() >= self.token_expires:
            if not self.authenticate():
                return None

        try:
            query = f"track:{track_name}"
            if artist_name:
                query += f" artist:{artist_name}"

            response = requests.get(
                'https://api.spotify.com/v1/search',
                headers={'Authorization': f'Bearer {self.access_token}'},
                params={'q': query, 'type': 'track', 'limit': 1},
                timeout=10
            )

            if response.status_code == 200:
                data = response.json()
                if data['tracks']['items']:
                    track = data['tracks']['items'][0]
                    logger.info(f"✓ Found track: {track['name']} by {track['artists'][0]['name']}")
                    return track
                else:
                    logger.error(f"No tracks found for '{track_name}' by '{artist_name}'")
                    return None
            else:
                logger.error(f"Spotify search failed: {response.status_code}")
                return None

        except Exception as e:
            logger.error(f"Track search error: {e}")
            return None

    def get_track_by_uri(self, uri: str) -> Optional[Dict]:
        """Get track by Spotify URI"""
        if not self.access_token or time.time() >= self.token_expires:
            if not self.authenticate():
                return None

        try:
            track_id = uri.split(':')[-1]
            response = requests.get(
                f'https://api.spotify.com/v1/tracks/{track_id}',
                headers={'Authorization': f'Bearer {self.access_token}'},
                timeout=10
            )

            if response.status_code == 200:
                track = response.json()
                logger.info(f"✓ Found track: {track['name']} by {track['artists'][0]['name']}")
                return track
            else:
                logger.error(f"Spotify track fetch failed: {response.status_code}")
                return None

        except Exception as e:
            logger.error(f"Track fetch error: {e}")
            return None

    def get_audio_analysis(self, track_id: str) -> Optional[Dict]:
        """Get detailed audio analysis for a track"""
        if not self.access_token or time.time() >= self.token_expires:
            if not self.authenticate():
                return None

        try:
            response = requests.get(
                f'https://api.spotify.com/v1/audio-analysis/{track_id}',
                headers={'Authorization': f'Bearer {self.access_token}'},
                timeout=30  # Increased timeout - analysis can be slow
            )

            if response.status_code == 200:
                analysis = response.json()
                logger.info(f"✓ Retrieved audio analysis ({len(analysis.get('sections', []))} sections, {len(analysis.get('beats', []))} beats)")
                return analysis
            elif response.status_code == 403:
                logger.warning(f"Audio analysis access denied (HTTP 403)")
                logger.warning("This is a Spotify API limitation - some tracks have restricted analysis")
                logger.info("Will use tempo-based simulation instead")
                return None
            elif response.status_code == 404:
                logger.warning(f"Audio analysis not available for this track (HTTP 404)")
                logger.info("Will use tempo-based simulation instead")
                return None
            else:
                logger.error(f"Audio analysis fetch failed: HTTP {response.status_code}")
                try:
                    error_data = response.json()
                    logger.error(f"Error details: {error_data}")
                except:
                    logger.error(f"Response text: {response.text[:200]}")
                return None

        except requests.exceptions.Timeout:
            logger.error(f"Audio analysis request timed out (track_id: {track_id})")
            logger.warning("Spotify's analysis endpoint can be slow - this is normal")
            return None
        except Exception as e:
            logger.error(f"Audio analysis error: {e}")
            return None

    def get_audio_features(self, track_id: str) -> Optional[Dict]:
        """Get audio features for a track"""
        if not self.access_token or time.time() >= self.token_expires:
            if not self.authenticate():
                return None

        try:
            response = requests.get(
                f'https://api.spotify.com/v1/audio-features/{track_id}',
                headers={'Authorization': f'Bearer {self.access_token}'},
                timeout=10
            )

            if response.status_code == 200:
                features = response.json()
                logger.info(f"✓ Retrieved audio features (tempo: {features.get('tempo', 0):.1f} BPM)")
                return features
            else:
                logger.error(f"Audio features fetch failed: {response.status_code}")
                return None

        except Exception as e:
            logger.error(f"Audio features error: {e}")
            return None


class SongSimulator:
    """Simulates a song structure with beats and sections"""

    def __init__(self, tempo: int = 120, use_spotify_data: bool = False):
        self.tempo = tempo
        self.beat_duration = 60.0 / tempo  # seconds per beat
        self.use_spotify_data = use_spotify_data

    def create_song_structure_from_spotify(
        self,
        analysis: Dict,
        features: Dict,
        track_duration_ms: int
    ) -> List[Dict]:
        """
        Create song structure from Spotify audio analysis data.
        Maps Spotify sections to our error types.
        """
        self.tempo = features.get('tempo', 120)
        self.beat_duration = 60.0 / self.tempo

        track_duration_sec = track_duration_ms / 1000.0
        sections = analysis.get('sections', [])
        beats_data = analysis.get('beats', [])

        if not sections:
            logger.warning("No sections in Spotify analysis, using simulated structure")
            return self.create_song_structure()

        # Convert sections to our format with error type mapping
        song_structure = []

        for i, section in enumerate(sections):
            start_time = section['start']
            duration = section['duration']
            confidence = section.get('confidence', 0)
            loudness = section.get('loudness', 0)
            tempo = section.get('tempo', self.tempo)

            # Calculate beat numbers from time
            start_beat = int(start_time / self.beat_duration)
            num_beats = int(duration / self.beat_duration)
            end_beat = start_beat + num_beats

            # Map section characteristics to error types
            error_type = self._map_section_to_error_type(
                i, len(sections), loudness, tempo, confidence
            )

            # Determine section name
            section_name = self._determine_section_name(i, len(sections), loudness)

            song_structure.append({
                'section': section_name,
                'start_beat': start_beat,
                'end_beat': end_beat,
                'error_type': error_type,
                'start_time': start_time,
                'duration': duration,
                'loudness': loudness,
                'tempo': tempo,
                'confidence': confidence
            })

        logger.info(f"✓ Parsed {len(song_structure)} sections from Spotify analysis")
        return song_structure

    def _map_section_to_error_type(
        self,
        section_idx: int,
        total_sections: int,
        loudness: float,
        tempo: float,
        confidence: float
    ) -> str:
        """
        Map Spotify section characteristics to our error types.

        Strategy:
        - Intro (first 1-2 sections): basic
        - Quiet sections: basic
        - Loud sections with high tempo: business or chaotic
        - Middle dramatic sections: chaotic (bridge-like)
        - Last 1-2 sections: philosophical (outro-like)
        """
        # First section is always intro/basic
        if section_idx == 0:
            return 'basic'

        # Last 1-2 sections are outro/philosophical
        if section_idx >= total_sections - 2:
            return 'philosophical'

        # Middle dramatic section could be bridge/chaotic
        middle_range = range(int(total_sections * 0.5), int(total_sections * 0.75))
        if section_idx in middle_range and loudness > -8:
            return 'chaotic'

        # Loud sections are chorus/business
        if loudness > -10:
            return 'business'

        # Everything else is verse/basic
        return 'basic'

    def _determine_section_name(
        self,
        section_idx: int,
        total_sections: int,
        loudness: float
    ) -> str:
        """Determine human-readable section name"""
        if section_idx == 0:
            return 'intro'
        elif section_idx >= total_sections - 2:
            return 'outro'
        elif loudness > -10:
            return 'chorus'
        elif int(total_sections * 0.5) <= section_idx < int(total_sections * 0.75):
            return 'bridge'
        else:
            return 'verse'

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

    def __init__(self, error_generator_url: str, slogan_server_url: str):
        self.error_generator_url = error_generator_url
        self.slogan_server_url = slogan_server_url
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

    def check_slogan_server_health(self) -> bool:
        """Check if slogan server is running"""
        try:
            response = requests.get(f"{self.slogan_server_url}/health", timeout=2)
            if response.status_code == 200:
                logger.info("✓ Slogan server is healthy")
                return True
            else:
                logger.warning(f"⚠️  Slogan server returned status {response.status_code}")
                return False
        except requests.exceptions.RequestException as e:
            logger.warning(f"⚠️  Cannot connect to slogan server at {self.slogan_server_url}")
            logger.warning(f"   Errors will be generated but slogans won't be created")
            logger.warning(f"   To enable slogans: cd ../slogan-server && go run main.go")
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


def run_demo(
    simulate_realtime: bool = False,
    spotify_track_name: Optional[str] = None,
    spotify_artist_name: Optional[str] = None,
    spotify_uri: Optional[str] = None
):
    """
    Run the rhythm-driven error generation demo.

    Args:
        simulate_realtime: If True, delays between beats to simulate real playback
        spotify_track_name: Name of Spotify track to use
        spotify_artist_name: Artist name for Spotify search
        spotify_uri: Spotify URI (e.g., spotify:track:xxx)
    """
    print("\n" + "🎼"*35)
    print("  RHYTHM-DRIVEN ERROR GENERATOR DEMO")
    print("  Full 3-Service Integration")
    print("🎼"*35 + "\n")

    # Check if using Spotify
    use_spotify = bool(spotify_track_name or spotify_uri)

    if use_spotify:
        print("🎵 Spotify Mode: Using real track data\n")
    else:
        print("🎼 Simulation Mode: Using generated song structure\n")

    # Initialize components
    simulator = SongSimulator(tempo=TEMPO)
    trigger = RhythmErrorTrigger(
        error_generator_url=ERROR_GENERATOR_URL,
        slogan_server_url=SLOGAN_SERVER_URL
    )

    print("📡 Checking service health...\n")

    # Check if error generator is running
    error_gen_healthy = trigger.check_service_health()
    slogan_healthy = trigger.check_slogan_server_health()

    print()  # Blank line after health checks

    if not error_gen_healthy:
        print("⚠️ " + "="*68)
        print("ERROR GENERATOR NOT RUNNING!")
        print("="*70)
        print("\nThe demo requires error-generator to be running.\n")
        print("To start it:")
        print("  cd ../error-generator")
        print("  RHYTHM_SERVICE_URL=http://localhost:5001 go run main.go")
        print("\n" + "="*70 + "\n")

        response = input("Continue anyway (no errors will be generated)? [y/N]: ")
        if response.lower() != 'y':
            print("Demo cancelled.")
            return

    if not slogan_healthy:
        print("📝 INFO: Slogan server not running - errors will generate without slogans")
        print("   For full experience with slogans & GIFs:")
        print("   cd ../slogan-server && go run main.go\n")

    # Create song structure (Spotify or simulated)
    structure = None
    track_name = "Simulated Song"
    track_artist = "Demo Generator"

    if use_spotify:
        # Fetch from Spotify
        print("🎵 Fetching track data from Spotify...\n")

        spotify_client = SpotifyClient(SPOTIFY_CLIENT_ID, SPOTIFY_CLIENT_SECRET)

        if not spotify_client.authenticate():
            print("❌ Failed to authenticate with Spotify")
            print("Using simulated structure instead\n")
            use_spotify = False
        else:
            # Get track
            track = None
            if spotify_uri:
                track = spotify_client.get_track_by_uri(spotify_uri)
            elif spotify_track_name:
                track = spotify_client.search_track(spotify_track_name, spotify_artist_name or '')

            if not track:
                print("❌ Failed to fetch track from Spotify")
                print("Using simulated structure instead\n")
                use_spotify = False
            else:
                track_name = track['name']
                track_artist = track['artists'][0]['name']
                track_id = track['id']
                track_duration_ms = track['duration_ms']

                print(f"🎵 Track: {track_name} by {track_artist}")
                print(f"   Duration: {track_duration_ms / 1000:.1f}s\n")

                # Get audio features first (more reliable)
                features = spotify_client.get_audio_features(track_id)

                if not features:
                    print("❌ Failed to get audio features")
                    print("Using simulated structure instead\n")
                    use_spotify = False
                else:
                    # Try to get detailed analysis
                    print("⏳ Fetching audio analysis (this may take 10-30 seconds)...")
                    analysis = spotify_client.get_audio_analysis(track_id)

                    if analysis:
                        # Use full analysis
                        structure = simulator.create_song_structure_from_spotify(
                            analysis, features, track_duration_ms
                        )
                        print()
                    else:
                        # Fallback: use just tempo from features with simulated structure
                        print("✓ Using Spotify tempo with simulated structure")
                        print(f"  Real tempo: {features.get('tempo', 120):.1f} BPM")
                        print(f"  Real duration: {track_duration_ms / 1000:.1f}s")
                        print(f"  Energy: {features.get('energy', 0):.2f}, Danceability: {features.get('danceability', 0):.2f}\n")

                        simulator.tempo = features.get('tempo', 120)
                        simulator.beat_duration = 60.0 / simulator.tempo
                        structure = simulator.create_song_structure()

                        # Adjust structure duration to match track
                        total_beats = int((track_duration_ms / 1000.0) / simulator.beat_duration)
                        if structure:
                            # Scale last section to match actual duration
                            structure[-1]['end_beat'] = total_beats

    if not use_spotify or structure is None:
        structure = simulator.create_song_structure()

    total_beats, total_duration = simulator.calculate_song_duration(structure)

    # Print song structure
    if use_spotify:
        print(f"📀 Now Playing: {track_name} by {track_artist}")
    print_song_structure(structure, int(simulator.tempo), simulator.beat_duration)

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
        print("  --realtime, -r                  Simulate real-time playback (delays between beats)")
        print("  --track <name>                  Spotify track name to use")
        print("  --artist <name>                 Artist name for Spotify search")
        print("  --spotify-uri <uri>             Spotify URI (e.g., spotify:track:5EWPGh7jbTNO2wakv8LjUI)")
        print("  --help, -h                      Show this help message")
        print("\nEnvironment Variables:")
        print("  ERROR_GENERATOR_URL             URL of error generator (default: http://localhost:9090)")
        print("  SLOGAN_SERVER_URL               URL of slogan server (default: http://localhost:8080)")
        print("  SPOTIFY_CLIENT_ID               Spotify API client ID")
        print("  SPOTIFY_CLIENT_SECRET           Spotify API client secret")
        print("  TEMPO                           Song tempo in BPM (default: 120)")
        print("\nExamples:")
        print("  # Simulated song")
        print("  python3 demo_rhythm_errors.py")
        print()
        print("  # Real Spotify track")
        print("  python3 demo_rhythm_errors.py --track \"Where Is My Mind?\" --artist \"Pixies\"")
        print()
        print("  # By Spotify URI")
        print("  python3 demo_rhythm_errors.py --spotify-uri \"spotify:track:5EWPGh7jbTNO2wakv8LjUI\"")
        print()
        print("  # Real-time mode with Spotify")
        print("  python3 demo_rhythm_errors.py --track \"Smells Like Teen Spirit\" --artist \"Nirvana\" --realtime")
        return

    # Parse Spotify arguments
    spotify_track_name = None
    spotify_artist_name = None
    spotify_uri = None

    if '--track' in sys.argv:
        idx = sys.argv.index('--track')
        if idx + 1 < len(sys.argv):
            spotify_track_name = sys.argv[idx + 1]

    if '--artist' in sys.argv:
        idx = sys.argv.index('--artist')
        if idx + 1 < len(sys.argv):
            spotify_artist_name = sys.argv[idx + 1]

    if '--spotify-uri' in sys.argv:
        idx = sys.argv.index('--spotify-uri')
        if idx + 1 < len(sys.argv):
            spotify_uri = sys.argv[idx + 1]

    run_demo(
        simulate_realtime=realtime,
        spotify_track_name=spotify_track_name,
        spotify_artist_name=spotify_artist_name,
        spotify_uri=spotify_uri
    )


if __name__ == '__main__':
    main()
