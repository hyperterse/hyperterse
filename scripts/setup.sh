#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "🚀 Setting up Hyperterse..."

# Check and install dependencies
"$SCRIPT_DIR/check-deps.sh"

# Generate protobuf files
"$SCRIPT_DIR/generate-proto.sh"

echo ""
echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "  1. Build the project:  make build"
echo "  2. Run the server:     ./hyperterse -file config.yaml"
echo ""
echo "Available Make commands:"
echo "  make build   - Build the project"
echo "  make generate - Regenerate protobuf files"
echo "  make lint    - Lint proto files"
echo "  make format  - Format proto files"

