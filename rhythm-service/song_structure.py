"""
Song Structure Analysis Module

Identifies song sections (verse, chorus, bridge, intro, outro) using:
- Spotify audio analysis API
- Self-similarity matrix analysis
- Sequence modeling with RNN/LSTM
- Optional FPGA acceleration
"""

import numpy as np
import logging

logger = logging.getLogger(__name__)

try:
    import librosa
    LIBROSA_AVAILABLE = True
except ImportError:
    logger.warning("Librosa not available, using simplified structure analysis")
    LIBROSA_AVAILABLE = False


class SongStructureAnalyzer:
    """
    Analyzes song structure to identify sections

    Sections mapped to error patterns:
    - Intro: Minimal errors
    - Verse: Basic technical errors
    - Chorus: Business-related errors
    - Bridge: Chaotic multi-error bursts
    - Outro: Philosophical errors with governing bodies
    """

    def __init__(self, use_fpga=False):
        """
        Initialize structure analyzer

        Args:
            use_fpga: If True, use FPGA acceleration
        """
        self.use_fpga = use_fpga
        self.model = None

        logger.info(f"Song structure analyzer initialized (FPGA: {use_fpga})")

    def analyze(self, song_data):
        """
        Analyze song structure from Spotify data

        Args:
            song_data: Dictionary containing Spotify track info and audio features

        Returns:
            Dictionary with section timings and classifications
        """
        # Try to use Spotify's built-in analysis first
        spotify_analysis = song_data.get('spotify_analysis')

        if spotify_analysis:
            return self._analyze_from_spotify(spotify_analysis)

        # Fall back to audio analysis
        audio_features = song_data.get('audio_features', {})
        duration = song_data.get('duration_ms', 180000) / 1000.0

        return self._estimate_structure(audio_features, duration)

    def _analyze_from_spotify(self, analysis):
        """
        Extract structure from Spotify's audio analysis

        Args:
            analysis: Spotify audio analysis object

        Returns:
            Structured section data
        """
        sections = analysis.get('sections', [])

        if not sections:
            logger.warning("No sections in Spotify analysis")
            return self._default_structure()

        # Map Spotify sections to our error patterns
        structure = []

        for i, section in enumerate(sections):
            start = section.get('start', 0)
            duration = section.get('duration', 0)
            loudness = section.get('loudness', -20)
            tempo = section.get('tempo', 120)

            # Classify section based on characteristics
            section_type = self._classify_section(
                section_num=i,
                loudness=loudness,
                tempo=tempo,
                total_sections=len(sections)
            )

            structure.append({
                'start': start,
                'duration': duration,
                'type': section_type,
                'loudness': loudness,
                'tempo': tempo
            })

        logger.info(f"Analyzed {len(structure)} sections from Spotify data")
        return structure

    def _classify_section(self, section_num, loudness, tempo, total_sections):
        """
        Classify a section as intro/verse/chorus/bridge/outro

        Args:
            section_num: Index of section
            loudness: Section loudness (dB)
            tempo: Section tempo (BPM)
            total_sections: Total number of sections

        Returns:
            Section type string
        """
        # Intro is usually the first section
        if section_num == 0:
            return "intro"

        # Outro is usually the last section
        if section_num == total_sections - 1:
            return "outro"

        # Chorus tends to be louder
        if loudness > -10:
            return "chorus"

        # Bridge is often in the later part but not the end
        if section_num > total_sections * 0.6 and section_num < total_sections - 1:
            return "bridge"

        # Default to verse
        return "verse"

    def _estimate_structure(self, audio_features, duration):
        """
        Estimate song structure from audio features when analysis is unavailable

        Args:
            audio_features: Spotify audio features
            duration: Song duration in seconds

        Returns:
            Estimated structure
        """
        logger.info("Estimating song structure from audio features")

        # Typical song structure: intro -> verse -> chorus -> verse -> chorus -> bridge -> chorus -> outro
        # This is a simplified heuristic

        energy = audio_features.get('energy', 0.5)
        danceability = audio_features.get('danceability', 0.5)

        # Estimate section durations (percentages of song)
        sections = [
            {'type': 'intro', 'start_pct': 0.0, 'duration_pct': 0.05},
            {'type': 'verse', 'start_pct': 0.05, 'duration_pct': 0.15},
            {'type': 'chorus', 'start_pct': 0.20, 'duration_pct': 0.10},
            {'type': 'verse', 'start_pct': 0.30, 'duration_pct': 0.15},
            {'type': 'chorus', 'start_pct': 0.45, 'duration_pct': 0.10},
            {'type': 'bridge', 'start_pct': 0.60, 'duration_pct': 0.15},
            {'type': 'chorus', 'start_pct': 0.75, 'duration_pct': 0.15},
            {'type': 'outro', 'start_pct': 0.90, 'duration_pct': 0.10},
        ]

        # Convert percentages to actual times
        structure = []
        for section in sections:
            structure.append({
                'start': section['start_pct'] * duration,
                'duration': section['duration_pct'] * duration,
                'type': section['type'],
                'energy': energy,
                'danceability': danceability
            })

        logger.info(f"Estimated {len(structure)} sections")
        return structure

    def _default_structure(self):
        """Return a default structure when analysis fails"""
        return [
            {'start': 0, 'duration': 30, 'type': 'verse'},
            {'start': 30, 'duration': 20, 'type': 'chorus'},
            {'start': 50, 'duration': 30, 'type': 'verse'},
            {'start': 80, 'duration': 20, 'type': 'chorus'},
            {'start': 100, 'duration': 20, 'type': 'bridge'},
            {'start': 120, 'duration': 30, 'type': 'chorus'},
        ]

    def analyze_self_similarity(self, audio_data, sr=22050):
        """
        Analyze song structure using self-similarity matrix

        This technique identifies repeating sections in music

        Args:
            audio_data: Audio samples
            sr: Sample rate

        Returns:
            Self-similarity matrix and estimated section boundaries
        """
        if not LIBROSA_AVAILABLE:
            logger.warning("Cannot compute self-similarity without Librosa")
            return None, None

        try:
            # Compute chroma features (12-dimensional pitch class representation)
            chroma = librosa.feature.chroma_cqt(y=audio_data, sr=sr)

            # Compute self-similarity matrix
            similarity_matrix = np.dot(chroma.T, chroma)

            # Normalize
            similarity_matrix = librosa.util.normalize(similarity_matrix, axis=1)

            # Detect boundaries using novelty function
            novelty = librosa.segment.recurrence_to_lag(similarity_matrix)
            boundaries = librosa.segment.agglomerative(chroma, k=8)

            logger.info(f"Detected {len(boundaries)} segment boundaries")

            return similarity_matrix, boundaries

        except Exception as e:
            logger.error(f"Self-similarity analysis failed: {e}")
            return None, None

    def get_section_at_time(self, structure, time_seconds):
        """
        Get the section type at a specific time

        Args:
            structure: Structure data from analyze()
            time_seconds: Time in seconds

        Returns:
            Section type string
        """
        for section in structure:
            start = section['start']
            end = start + section['duration']

            if start <= time_seconds < end:
                return section['type']

        # Default to last section if time is beyond song
        if structure:
            return structure[-1]['type']

        return "verse"


def visualize_structure(structure, duration):
    """
    Create ASCII visualization of song structure

    Args:
        structure: Structure data from analyze()
        duration: Total song duration

    Returns:
        ASCII string visualization
    """
    # Create timeline
    width = 80
    timeline = [' '] * width

    section_chars = {
        'intro': 'I',
        'verse': 'V',
        'chorus': 'C',
        'bridge': 'B',
        'outro': 'O'
    }

    for section in structure:
        start_pos = int((section['start'] / duration) * width)
        end_pos = int(((section['start'] + section['duration']) / duration) * width)

        char = section_chars.get(section['type'], '?')

        for i in range(start_pos, min(end_pos, width)):
            timeline[i] = char

    # Create visualization
    viz = []
    viz.append("Song Structure:")
    viz.append("=" * width)
    viz.append(''.join(timeline))
    viz.append("=" * width)
    viz.append("Legend: I=Intro V=Verse C=Chorus B=Bridge O=Outro")
    viz.append(f"Duration: {duration:.1f}s")

    return '\n'.join(viz)
