#!/usr/bin/env bash
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DIR"

echo "🚀 Starting Antigravity Remote server on port 8765..."
exec ./bin/server -port 8765 -static ./frontend/build
