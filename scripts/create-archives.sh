#!/bin/bash
set -e

# Flatten binaries from artifact subdirectories to dist root
# After download-artifact, structure is: dist/hyperterse-linux-amd64/hyperterse-linux-amd64
# This script flattens to: dist/hyperterse-linux-amd64

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

DIST_DIR="${1:-dist}"

if [ ! -d "$DIST_DIR" ]; then
    echo "Error: Directory $DIST_DIR does not exist"
    exit 1
fi

cd "$DIST_DIR"

for dir in hyperterse-*; do
    if [ -d "$dir" ]; then
        binary=$(find "$dir" -name "hyperterse-*" -type f | head -1)
        if [ -z "$binary" ]; then
            continue
        fi

        binary_name=$(basename "$binary")

        # Move binary to temp location first to avoid conflict with same-named directory
        # (mv would move INTO the directory instead of replacing it)
        mv "$binary" "./${binary_name}.tmp"

        # Clean up artifact directory
        rm -rf "$dir"

        # Rename to final name
        mv "./${binary_name}.tmp" "./$binary_name"
    fi
done

echo "✓ Binaries flattened successfully"

