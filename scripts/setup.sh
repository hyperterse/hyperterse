#!/bin/bash
set -e

echo "🚀 Setting up Hyperterse..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.25.1 or later from https://go.dev/dl/"
    exit 1
fi

echo "✓ Go found: $(go version)"

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "📦 Installing protoc..."

    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        if command -v brew &> /dev/null; then
            brew install protobuf
        else
            echo "❌ Homebrew not found. Please install protoc manually:"
            echo "   brew install protobuf"
            exit 1
        fi
    else
        # Linux
        echo "❌ protoc not found. Please install protoc manually:"
        echo "   sudo apt-get install protobuf-compiler  # Debian/Ubuntu"
        echo "   sudo yum install protobuf-compiler       # RHEL/CentOS"
        exit 1
    fi
else
    echo "✓ protoc found: $(protoc --version)"
fi

# Install protoc-gen-go if not present
if ! command -v protoc-gen-go &> /dev/null && [ ! -f "$(go env GOPATH)/bin/protoc-gen-go" ]; then
    echo "📦 Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
else
    echo "✓ protoc-gen-go found"
fi

# Note: Buf CLI is optional and only needed for linting/formatting proto files
# Install it manually if desired: https://buf.build/docs/installation

# Download Go dependencies
echo "📥 Downloading Go dependencies..."
go mod download

# Generate protobuf code
echo "🔨 Generating protobuf files..."
mkdir -p core/pb
export PATH="$(go env GOPATH)/bin:$PATH"
protoc \
    -I. \
    --go_out=core/pb \
    --go_opt=paths=source_relative \
    proto/connectors.proto \
    proto/primitives.proto \
    proto/hyperterse.proto \
    proto/runtime.proto

# Move generated files to correct location
echo "📦 Organizing generated files..."
if [ -f "core/pb/proto/hyperterse.pb.go" ]; then
    mv core/pb/proto/hyperterse.pb.go core/pb/ 2>/dev/null || true
    mv core/pb/proto/connectors.pb.go core/pb/ 2>/dev/null || true
    mv core/pb/proto/primitives.pb.go core/pb/ 2>/dev/null || true
    mv core/pb/proto/runtime.pb.go core/pb/ 2>/dev/null || true
    rmdir core/pb/proto 2>/dev/null || true
fi
if [ -d "core/pb/hyperterse" ]; then
    mv core/pb/hyperterse/hyperterse.pb.go core/pb/ 2>/dev/null || true
    mv core/pb/hyperterse/connectors.pb.go core/pb/ 2>/dev/null || true
    mv core/pb/hyperterse/primitives.pb.go core/pb/ 2>/dev/null || true
    rmdir core/pb/hyperterse 2>/dev/null || true
fi
if [ -f "core/pb/runtime.pb.go" ]; then
    mkdir -p core/pb/runtime
    mv core/pb/runtime.pb.go core/pb/runtime/ 2>/dev/null || true
fi

# Generate types
echo "🔨 Generating types..."
mkdir -p core/types
go run scripts/generate_types/script.go proto/connectors.proto proto/primitives.proto

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

