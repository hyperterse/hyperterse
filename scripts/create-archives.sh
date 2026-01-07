#!/bin/bash
set -e

# Ensure we're in the project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Default to dist directory
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
        platform="${dir#hyperterse-}"
        
        # Copy binary to dist root
        cp "$binary" "./$binary_name"
        
        # Create archive
        if [[ "$binary_name" == *.exe ]]; then
            # Windows - create zip
            archive_name="hyperterse-${platform}.zip"
            zip "${archive_name}" "${binary_name}"
            [ -f ../README.md ] && zip -u "${archive_name}" ../README.md || true
            [ -f ../LICENSE ] && zip -u "${archive_name}" ../LICENSE || true
            [ -f ../install.sh ] && zip -u "${archive_name}" ../install.sh || true
        else
            # Unix-like - create tar.gz
            archive_name="hyperterse-${platform}.tar.gz"
            # Build list of files to include
            files="${binary_name}"
            [ -f ../README.md ] && files="${files} ../README.md" || true
            [ -f ../LICENSE ] && files="${files} ../LICENSE" || true
            [ -f ../install.sh ] && files="${files} ../install.sh" || true
            # Create archive with all files
            tar -czf "${archive_name}" ${files}
        fi
    fi
done

echo "✓ Archives created successfully"

