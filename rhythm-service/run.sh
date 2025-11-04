#!/bin/bash
# Quick start script for rhythm service

# Load environment variables
export $(grep -v '^#' .env | xargs)

# Suppress warnings
export PYTHONWARNINGS="ignore::UserWarning,ignore::DeprecationWarning"

echo "🎵 Starting Rhythm Service on port ${PORT:-5001}..."
echo "Error Generator URL: ${ERROR_GENERATOR_URL:-http://localhost:9090}"
echo ""

# Run the service
python3 rhythm_service.py
