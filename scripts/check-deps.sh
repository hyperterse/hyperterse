#!/bin/bash
set -e

# Ensure we're in the project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

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

# Download Go dependencies
echo "📥 Downloading Go dependencies..."
go mod download

