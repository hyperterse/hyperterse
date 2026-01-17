#!/bin/bash
set -e

# Ensure we're in the project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Parse arguments
GOOS="${1}"
GOARCH="${2}"
OUTPUT_DIR="${3:-dist}"

if [ -z "$GOOS" ] || [ -z "$GOARCH" ]; then
    echo "Usage: $0 <GOOS> <GOARCH> [OUTPUT_DIR]"
    echo "Example: $0 linux amd64 dist"
    exit 1
fi

# Use build.sh script for building
./scripts/build.sh "$GOOS" "$GOARCH" "$OUTPUT_DIR" "hyperterse"

