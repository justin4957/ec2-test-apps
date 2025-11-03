#!/usr/bin/env python3
"""
Example Usage of Rhythm Service

Demonstrates how to use the rhythm service components
for educational purposes.
"""

import numpy as np
from beat_detector import BeatDetector, create_simple_beat_detector
from song_structure import SongStructureAnalyzer, visualize_structure
from hls4ml_interface import HLS4MLInference


def example_beat_detection():
    """Example: Detect beats from audio features"""
    print("=" * 60)
    print("EXAMPLE 1: Beat Detection from Spotify Features")
    print("=" * 60)

    detector = create_simple_beat_detector()

    # Simulate Spotify audio features
    audio_features = {
        'tempo': 125.0,
        'time_signature': 4,
        'energy': 0.8,
        'danceability': 0.7
    }

    beat_info = detector.detect_beats_from_features(audio_features)

    print(f"\nDetected tempo: {beat_info['tempo']} BPM")
    print(f"Beat interval: {beat_info['beat_interval']:.3f} seconds")
    print(f"Time signature: {beat_info['time_signature']}/4")
    print(f"Energy: {beat_info['energy']}")
    print(f"Danceability: {beat_info['danceability']}")

    # Calculate when beats occur
    print(f"\nFirst 10 beats occur at:")
    for i in range(10):
        beat_time = i * beat_info['beat_interval']
        print(f"  Beat {i+1}: {beat_time:.2f}s")


def example_song_structure():
    """Example: Analyze song structure"""
    print("\n" + "=" * 60)
    print("EXAMPLE 2: Song Structure Analysis")
    print("=" * 60)

    analyzer = SongStructureAnalyzer()

    # Simulate song data
    song_data = {
        'name': 'Where Is My Mind?',
        'artist': 'Pixies',
        'duration_ms': 234000,  # 3:54
        'audio_features': {
            'tempo': 115,
            'energy': 0.8,
            'danceability': 0.6
        }
    }

    # Analyze structure
    structure = analyzer.analyze(song_data)

    print(f"\nAnalyzing: {song_data['name']} by {song_data['artist']}")
    print(f"Duration: {song_data['duration_ms']/1000:.1f}s")

    print(f"\nDetected {len(structure)} sections:")
    for i, section in enumerate(structure):
        print(f"  {i+1}. {section['type']:8} | {section['start']:6.1f}s - {section['start']+section['duration']:6.1f}s | {section['duration']:5.1f}s")

    # Visualize
    print("\n" + visualize_structure(structure, song_data['duration_ms']/1000))

    # Test section lookup
    test_times = [10, 45, 90, 150]
    print(f"\nSection at specific times:")
    for t in test_times:
        section = analyzer.get_section_at_time(structure, t)
        print(f"  {t:3d}s: {section}")


def example_error_pattern_mapping():
    """Example: Map song sections to error patterns"""
    print("\n" + "=" * 60)
    print("EXAMPLE 3: Error Pattern Mapping")
    print("=" * 60)

    section_to_error = {
        "intro": "minimal",
        "verse": "basic",
        "chorus": "business",
        "bridge": "chaotic",
        "outro": "philosophical"
    }

    print("\nSong Section → Error Pattern Mapping:")
    print("-" * 60)

    for section, pattern in section_to_error.items():
        print(f"  {section:12} → {pattern:15}")

        # Show example errors for each pattern
        examples = {
            "minimal": "Simple errors, sparse",
            "basic": "NullPointerException, IndexOutOfBounds",
            "business": "PaymentGatewayTimeout, InventoryMismatch",
            "chaotic": "Multiple rapid-fire errors",
            "philosophical": "Errors with governing body references"
        }

        print(f"    Example: {examples[pattern]}")
        print()


def example_rhythm_controller():
    """Example: Rhythm-driven error triggering"""
    print("=" * 60)
    print("EXAMPLE 4: Rhythm-Driven Error Generation")
    print("=" * 60)

    # Simulate a song playing
    song = {
        'name': 'Smells Like Teen Spirit',
        'artist': 'Nirvana',
        'tempo': 117,
        'duration': 301  # 5:01
    }

    beat_interval = 60.0 / song['tempo']

    print(f"\nNow playing: {song['name']} by {song['artist']}")
    print(f"Tempo: {song['tempo']} BPM")
    print(f"Beat interval: {beat_interval:.3f}s\n")

    # Song structure
    structure = [
        {'start': 0, 'duration': 8, 'type': 'intro'},
        {'start': 8, 'duration': 48, 'type': 'verse'},
        {'start': 56, 'duration': 28, 'type': 'chorus'},
        {'start': 84, 'duration': 48, 'type': 'verse'},
        {'start': 132, 'duration': 28, 'type': 'chorus'},
        {'start': 160, 'duration': 20, 'type': 'bridge'},
        {'start': 180, 'duration': 28, 'type': 'chorus'},
        {'start': 208, 'duration': 93, 'type': 'outro'},
    ]

    # Simulate first 20 beats
    print("Simulating first 20 beats with error triggers:\n")

    for beat_num in range(20):
        beat_time = beat_num * beat_interval

        # Find current section
        current_section = "verse"
        for section in structure:
            if section['start'] <= beat_time < section['start'] + section['duration']:
                current_section = section['type']
                break

        # Map to error type
        error_type = {
            'intro': 'minimal',
            'verse': 'basic',
            'chorus': 'business',
            'bridge': 'chaotic',
            'outro': 'philosophical'
        }.get(current_section, 'basic')

        print(f"Beat {beat_num+1:2d} @ {beat_time:6.2f}s | Section: {current_section:12} | Error: {error_type}")


def example_fpga_workflow():
    """Example: FPGA deployment workflow"""
    print("\n" + "=" * 60)
    print("EXAMPLE 5: FPGA Deployment Workflow (Educational)")
    print("=" * 60)

    print("\nThis example shows the workflow for deploying a beat")
    print("detection model to FPGA using hls4ml:\n")

    steps = [
        ("1. Create Model", "Build quantized RNN/LSTM with QKeras"),
        ("2. Train Model", "Train on beat-annotated audio dataset"),
        ("3. Convert to HLS", "Use hls4ml to generate C++ code"),
        ("4. Synthesize", "Generate FPGA bitstream (5-15 min)"),
        ("5. Deploy", "Load onto FPGA and run inference"),
        ("6. Benchmark", "Measure latency: ~1-10μs!")
    ]

    for step, description in steps:
        print(f"  {step:20} {description}")

    print("\nExpected Performance:")
    print(f"  CPU Inference:  10-50 ms")
    print(f"  FPGA Inference: 1-10 μs")
    print(f"  Speedup:        1000-5000x faster!")

    print("\nThis enables real-time beat detection with microsecond latency,")
    print("perfect for synchronizing error generation to music!")


def main():
    """Run all examples"""
    print("\n")
    print("╔" + "=" * 58 + "╗")
    print("║" + " " * 58 + "║")
    print("║" + "  RHYTHM-DRIVEN ERROR GENERATOR - EXAMPLE USAGE  ".center(58) + "║")
    print("║" + " " * 58 + "║")
    print("╚" + "=" * 58 + "╝")
    print("\n")

    try:
        example_beat_detection()
        example_song_structure()
        example_error_pattern_mapping()
        example_rhythm_controller()
        example_fpga_workflow()

        print("\n" + "=" * 60)
        print("All examples completed successfully!")
        print("=" * 60 + "\n")

    except Exception as e:
        print(f"\n❌ Error running examples: {e}")
        import traceback
        traceback.print_exc()


if __name__ == '__main__':
    main()
