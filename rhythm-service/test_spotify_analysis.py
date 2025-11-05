#!/usr/bin/env python3
"""
Test script to find tracks with available audio analysis on Spotify.
"""

import os
import sys
import time
import base64
import requests

SPOTIFY_CLIENT_ID = os.getenv('SPOTIFY_CLIENT_ID', '')
SPOTIFY_CLIENT_SECRET = os.getenv('SPOTIFY_CLIENT_SECRET', '')

# Popular tracks from various genres and eras to test
TEST_TRACKS = [
    # Classic Rock
    ("Bohemian Rhapsody", "Queen"),
    ("Stairway to Heaven", "Led Zeppelin"),
    ("Hotel California", "Eagles"),
    ("Sweet Child O' Mine", "Guns N' Roses"),
    ("Back In Black", "AC/DC"),

    # Alternative/90s
    ("Smells Like Teen Spirit", "Nirvana"),
    ("Creep", "Radiohead"),
    ("Wonderwall", "Oasis"),
    ("Black Hole Sun", "Soundgarden"),
    ("Jeremy", "Pearl Jam"),

    # Indie/Modern
    ("Mr. Brightside", "The Killers"),
    ("Seven Nation Army", "The White Stripes"),
    ("Take Me Out", "Franz Ferdinand"),
    ("Float On", "Modest Mouse"),
    ("Electric Feel", "MGMT"),

    # Electronic/Dance
    ("Around the World", "Daft Punk"),
    ("One More Time", "Daft Punk"),
    ("Levels", "Avicii"),
    ("Animals", "Martin Garrix"),
    ("Titanium", "David Guetta"),

    # Pop
    ("Shape of You", "Ed Sheeran"),
    ("Blinding Lights", "The Weeknd"),
    ("Rolling in the Deep", "Adele"),
    ("Uptown Funk", "Mark Ronson"),
    ("Happy", "Pharrell Williams"),

    # Hip Hop
    ("Lose Yourself", "Eminem"),
    ("Stronger", "Kanye West"),
    ("HUMBLE.", "Kendrick Lamar"),
    ("God's Plan", "Drake"),
    ("Sicko Mode", "Travis Scott"),

    # Classic
    ("Billie Jean", "Michael Jackson"),
    ("Thriller", "Michael Jackson"),
    ("Bohemian Rhapsody", "Queen"),
    ("Imagine", "John Lennon"),
    ("Hey Jude", "The Beatles"),
]


def authenticate(client_id: str, client_secret: str) -> str:
    """Authenticate with Spotify and return access token"""
    auth_str = f"{client_id}:{client_secret}"
    auth_bytes = auth_str.encode('utf-8')
    auth_base64 = base64.b64encode(auth_bytes).decode('utf-8')

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
        return response.json()['access_token']
    else:
        print(f"❌ Authentication failed: {response.status_code}")
        return None


def search_track(access_token: str, track_name: str, artist_name: str):
    """Search for a track"""
    query = f"track:{track_name} artist:{artist_name}"

    response = requests.get(
        'https://api.spotify.com/v1/search',
        headers={'Authorization': f'Bearer {access_token}'},
        params={'q': query, 'type': 'track', 'limit': 1},
        timeout=10
    )

    if response.status_code == 200:
        data = response.json()
        if data['tracks']['items']:
            return data['tracks']['items'][0]
    return None


def test_audio_analysis(access_token: str, track_id: str) -> bool:
    """Test if audio analysis is available for a track"""
    response = requests.get(
        f'https://api.spotify.com/v1/audio-analysis/{track_id}',
        headers={'Authorization': f'Bearer {access_token}'},
        timeout=30
    )

    return response.status_code == 200


def main():
    if not SPOTIFY_CLIENT_ID or not SPOTIFY_CLIENT_SECRET:
        print("❌ Spotify credentials not set")
        print("Set SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET environment variables")
        return

    print("🎵 Testing Spotify Audio Analysis Availability")
    print("=" * 70)
    print()

    # Authenticate
    print("Authenticating with Spotify...")
    access_token = authenticate(SPOTIFY_CLIENT_ID, SPOTIFY_CLIENT_SECRET)

    if not access_token:
        return

    print("✓ Authenticated\n")

    # Test tracks
    available_tracks = []
    unavailable_tracks = []

    for i, (track_name, artist_name) in enumerate(TEST_TRACKS, 1):
        print(f"[{i:2d}/{len(TEST_TRACKS)}] Testing: {track_name} by {artist_name}...", end=" ")

        # Search for track
        track = search_track(access_token, track_name, artist_name)

        if not track:
            print("❌ Not found")
            unavailable_tracks.append((track_name, artist_name, "Not found"))
            continue

        track_id = track['id']
        track_uri = track['uri']
        actual_name = track['name']
        actual_artist = track['artists'][0]['name']

        # Test analysis
        has_analysis = test_audio_analysis(access_token, track_id)

        if has_analysis:
            print("✅ Available")
            available_tracks.append({
                'name': actual_name,
                'artist': actual_artist,
                'id': track_id,
                'uri': track_uri
            })
        else:
            print("❌ Restricted")
            unavailable_tracks.append((actual_name, actual_artist, "403 Restricted"))

        # Be nice to Spotify API
        time.sleep(0.5)

    # Results
    print("\n" + "=" * 70)
    print("RESULTS")
    print("=" * 70)
    print()

    print(f"✅ AVAILABLE ({len(available_tracks)}):")
    print("-" * 70)
    for i, track in enumerate(available_tracks, 1):
        print(f"{i:2d}. {track['name']} - {track['artist']}")
        print(f"    URI: {track['uri']}")
        print(f"    ID:  {track['id']}")
        print()

    if len(available_tracks) >= 10:
        print("🎉 Found 10+ tracks with audio analysis!")
        print("\nYou can use any of these with the demo:")
        print()
        for i, track in enumerate(available_tracks[:10], 1):
            print(f"  python3 demo_rhythm_errors.py --track \"{track['name']}\" --artist \"{track['artist']}\"")
        print()
    else:
        print(f"⚠️  Only found {len(available_tracks)} tracks with analysis available")

    print()
    print(f"❌ UNAVAILABLE ({len(unavailable_tracks)}):")
    print("-" * 70)
    for track_name, artist_name, reason in unavailable_tracks[:10]:
        print(f"  • {track_name} - {artist_name} ({reason})")
    if len(unavailable_tracks) > 10:
        print(f"  ... and {len(unavailable_tracks) - 10} more")


if __name__ == '__main__':
    main()
