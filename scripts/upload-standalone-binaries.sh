#!/bin/bash

# Script to upload standalone binaries to GitHub releases after Goreleaser runs
# This ensures binaries are available with the naming convention expected by install.sh
# Format: hyperterse-{os}-{arch} (e.g., hyperterse-linux-amd64)

set -e

if [ -z "$GITHUB_TOKEN" ]; then
    echo "Error: GITHUB_TOKEN environment variable is required"
    exit 1
fi

if [ -z "$GITHUB_REPOSITORY" ]; then
    echo "Error: GITHUB_REPOSITORY environment variable is required (e.g., hyperterse/hyperterse)"
    exit 1
fi

if [ -z "$GITHUB_REF" ]; then
    echo "Error: GITHUB_REF environment variable is required (e.g., refs/tags/v1.0.0)"
    exit 1
fi

# Extract tag from GITHUB_REF
TAG="${GITHUB_REF#refs/tags/}"

# Extract repository owner and name
REPO_OWNER=$(echo "$GITHUB_REPOSITORY" | cut -d'/' -f1)
REPO_NAME=$(echo "$GITHUB_REPOSITORY" | cut -d'/' -f2)

echo "Uploading standalone binaries for tag: $TAG"
echo "Repository: $REPO_OWNER/$REPO_NAME"

# Function to upload a file to GitHub release
upload_to_release() {
    local file_path=$1
    local file_name=$(basename "$file_path")
    
    if [ ! -f "$file_path" ]; then
        echo "Warning: File not found: $file_path"
        return 1
    fi
    
    echo "Uploading $file_name..."
    
    # Use GitHub CLI if available, otherwise use API
    if command -v gh >/dev/null 2>&1; then
        gh release upload "$TAG" "$file_path" --repo "$REPO_OWNER/$REPO_NAME" --clobber
    else
        # Use curl to upload via GitHub API
        # First, get the release ID
        release_id=$(curl -sL \
            -H "Authorization: token $GITHUB_TOKEN" \
            -H "Accept: application/vnd.github.v3+json" \
            "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/tags/$TAG" | \
            grep '"id"' | head -1 | sed -E 's/.*"id": ([0-9]+).*/\1/')
        
        if [ -z "$release_id" ]; then
            echo "Error: Could not find release for tag $TAG"
            return 1
        fi
        
        # Upload the asset
        upload_url="https://uploads.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/$release_id/assets?name=$file_name"
        curl -X POST \
            -H "Authorization: token $GITHUB_TOKEN" \
            -H "Content-Type: application/octet-stream" \
            --data-binary "@$file_path" \
            "$upload_url"
    fi
}

# Process each dist directory
for dist_dir in dist/hyperterse_*; do
    if [ ! -d "$dist_dir" ]; then
        continue
    fi
    
    # Extract OS and architecture from directory name
    # Format: hyperterse_{os}_{arch} (e.g., hyperterse_linux_amd64, hyperterse_linux_armv7)
    os_arch=$(basename "$dist_dir" | sed 's/hyperterse_//')
    os=$(echo "$os_arch" | cut -d'_' -f1)
    arch=$(echo "$os_arch" | cut -d'_' -f2-)
    
    # Normalize architecture names to match install.sh expectations
    case "$arch" in
        amd64|x86_64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        armv7|armv7l)
            arch="armv7"
            ;;
        armv6|armv6l)
            arch="armv6"
            ;;
        386|i386|i686)
            arch="386"
            ;;
    esac
    
    binary_path="$dist_dir/hyperterse"
    
    if [ ! -f "$binary_path" ]; then
        echo "Warning: Binary not found in $dist_dir"
        continue
    fi
    
    # Create standalone binary with install.sh naming convention
    if [ "$os" = "windows" ]; then
        standalone_name="hyperterse-${os}-${arch}.exe"
    else
        standalone_name="hyperterse-${os}-${arch}"
    fi
    
    standalone_path="$dist_dir/$standalone_name"
    cp "$binary_path" "$standalone_path"
    chmod +x "$standalone_path"
    
    # Upload standalone binary
    upload_to_release "$standalone_path"
    
    echo "✓ Uploaded $standalone_name"
done

echo "✓ All standalone binaries uploaded successfully"

