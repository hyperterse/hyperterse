#!/bin/bash
set -e

# Ensure we're in the project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Check if protoc is available
if ! command -v protoc &> /dev/null; then
    echo "❌ Error: protoc not found. Run 'make setup' or './scripts/setup.sh' first"
    exit 1
fi

# Clean and generate protobuf code
echo "🔨 Generating protobuf files..."
rm -rf core/proto core/types
mkdir -p core/proto
export PATH="$(go env GOPATH)/bin:$PATH"
protoc \
    -I. \
    --go_out=core \
    --go_opt=paths=source_relative \
    proto/connectors/connectors.proto \
    proto/primitives/primitives.proto \
    proto/hyperterse/hyperterse.proto \
    proto/runtime/runtime.proto

# Generate types
echo "🔨 Generating types..."
mkdir -p core/types
go run scripts/generate_types/script.go proto/connectors/connectors.proto proto/primitives/primitives.proto

echo "✓ Protobuf generation complete"

