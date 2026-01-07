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

mkdir -p "${OUTPUT_DIR}"

# Build binary name
output_name="hyperterse-${GOOS}-${GOARCH}"
if [ "$GOOS" = "windows" ]; then
    output_name="${output_name}.exe"
fi

echo "Building ${output_name} for ${GOOS}/${GOARCH}..."

# Build the binary
CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -trimpath -ldflags="-s -w" -o "${OUTPUT_DIR}/${output_name}" .

echo "✓ Built ${output_name}"

