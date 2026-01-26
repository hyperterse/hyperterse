#!/bin/bash
set -e

# Script to update SHA256 checksums in Homebrew formula
# Usage: ./scripts/update-homebrew-sha256.sh <formula_file> <darwin_amd64> <darwin_arm64> <linux_amd64> <linux_arm64> <linux_arm>

FORMULA_FILE="$1"
DARWIN_AMD64="$2"
DARWIN_ARM64="$3"
LINUX_AMD64="$4"
LINUX_ARM64="$5"
LINUX_ARM="$6"

if [ -z "$FORMULA_FILE" ] || [ ! -f "$FORMULA_FILE" ]; then
    echo "Error: Formula file not found: $FORMULA_FILE"
    exit 1
fi

# Use Python for more reliable text processing
python3 << EOF
import re
import sys

darwin_amd64 = "$DARWIN_AMD64"
darwin_arm64 = "$DARWIN_ARM64"
linux_amd64 = "$LINUX_AMD64"
linux_arm64 = "$LINUX_ARM64"
linux_arm = "$LINUX_ARM"

with open("$FORMULA_FILE", 'r') as f:
    content = f.read()

# Update SHA256 for darwin-amd64 (line after darwin-amd64)
if darwin_amd64:
    content = re.sub(
        r'(darwin-amd64"\s*\n\s*)sha256 "[^"]*"',
        r'\1sha256 "' + darwin_amd64 + '"',
        content
    )

# Update SHA256 for darwin-arm64 (line after darwin-arm64)
if darwin_arm64:
    content = re.sub(
        r'(darwin-arm64"\s*\n\s*)sha256 "[^"]*"',
        r'\1sha256 "' + darwin_arm64 + '"',
        content
    )

# Update SHA256 for linux-amd64 (line after linux-amd64)
if linux_amd64:
    content = re.sub(
        r'(linux-amd64"\s*\n\s*)sha256 "[^"]*"',
        r'\1sha256 "' + linux_amd64 + '"',
        content
    )

# Update SHA256 for linux-arm64 (line after linux-arm64)
if linux_arm64:
    content = re.sub(
        r'(linux-arm64"\s*\n\s*)sha256 "[^"]*"',
        r'\1sha256 "' + linux_arm64 + '"',
        content
    )

# Update SHA256 for linux-arm (line after linux-arm, but not arm64)
if linux_arm:
    content = re.sub(
        r'(linux-arm"\s*\n\s*)sha256 "[^"]*"',
        r'\1sha256 "' + linux_arm + '"',
        content
    )

with open("$FORMULA_FILE", 'w') as f:
    f.write(content)
EOF

echo "✓ Updated SHA256 checksums in $FORMULA_FILE"
