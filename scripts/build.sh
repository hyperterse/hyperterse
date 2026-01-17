#!/bin/bash
set -e

# Ensure we're in the project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Parse arguments
GOOS="${1:-}"
GOARCH="${2:-}"
OUTPUT_DIR="${3:-dist}"
OUTPUT_NAME="${4:-hyperterse}"

# Get version from git tag
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
# Remove 'v' prefix if present
VERSION=${VERSION#v}
# Remove -dirty suffix if present
VERSION=${VERSION%-dirty}
# Remove commit hash suffix (e.g., 1.0.0-5-gabc1234 -> 1.0.0, 1.0.0-alpha.1-5-gabc1234 -> 1.0.0-alpha.1)
# But preserve prerelease tags (e.g., 1.0.0-alpha.1 should stay as 1.0.0-alpha.1)
# Git describe commit hash pattern: -<number>-g<hex> (e.g., -5-gabc1234)
if [[ $VERSION =~ -[0-9]+-g[0-9a-f]+$ ]]; then
    # Remove the commit hash suffix (everything from -<number>-g onwards)
    VERSION=$(echo "$VERSION" | sed -E 's/-[0-9]+-g[0-9a-f]+$//')
fi

mkdir -p "${OUTPUT_DIR}"

# Build binary name
if [ -n "$GOOS" ] && [ -n "$GOARCH" ]; then
    # Cross-compilation mode
    output_name="${OUTPUT_NAME}-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    echo "Building ${output_name} for ${GOOS}/${GOARCH}..."

    # Build the binary with version embedded
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -trimpath -ldflags="-s -w -X main.Version=${VERSION}" -o "${OUTPUT_DIR}/${output_name}" .

    echo "✓ Built ${output_name} (version: ${VERSION})"
else
    # Local build mode (current platform)
    echo "Building hyperterse..."
    go build -mod=mod -trimpath -ldflags="-s -w -X main.Version=${VERSION}" -o "${OUTPUT_DIR}/${OUTPUT_NAME}" .
    echo "✓ Build complete (version: ${VERSION})"
fi
