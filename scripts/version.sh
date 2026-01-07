#!/bin/bash
set -e

# Ensure we're in the project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Function to display usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --major              Bump major version (e.g., 1.0.0 -> 2.0.0)"
    echo "  --minor              Bump minor version (e.g., 1.0.0 -> 1.1.0)"
    echo "  --patch              Bump patch version (e.g., 1.0.0 -> 1.0.1)"
    echo "  --prerelease <tag>   Create a prerelease version (e.g., --prerelease alpha)"
    echo "  --version <version>  Specify exact version (e.g., --version 1.2.3)"
    echo "  --push               Push tag to remote using 'git push --follow-tags'"
    echo ""
    echo "Examples:"
    echo "  $0 --major"
    echo "  $0 --minor"
    echo "  $0 --patch"
    echo "  $0 --prerelease beta"
    echo "  $0 --version 2.0.0"
    exit 1
}

# Function to get the latest version tag
get_latest_version() {
    # Get all tags matching v* pattern, sort by version, and get the latest
    local latest_tag=$(git tag -l "v*" | sort -V | tail -n 1)

    if [ -z "$latest_tag" ]; then
        echo "0.0.0"
    else
        # Remove the 'v' prefix
        echo "${latest_tag#v}"
    fi
}

# Function to parse version string into components
parse_version() {
    local version=$1
    IFS='.' read -ra parts <<< "$version"
    MAJOR=${parts[0]:-0}
    MINOR=${parts[1]:-0}
    PATCH=${parts[2]:-0}
    PRERELEASE=""

    # Check if there's a prerelease part (e.g., 1.0.0-alpha.1)
    if [[ $version =~ - ]]; then
        PRERELEASE="${version#*-}"
        # Remove prerelease from patch if it was included
        PATCH="${parts[2]%%-*}"
    fi
}

# Function to bump version
bump_version() {
    local current_version=$1
    local bump_type=$2
    local prerelease_tag=$3

    parse_version "$current_version"

    case "$bump_type" in
        major)
            MAJOR=$((MAJOR + 1))
            MINOR=0
            PATCH=0
            PRERELEASE=""
            ;;
        minor)
            MINOR=$((MINOR + 1))
            PATCH=0
            PRERELEASE=""
            ;;
        patch)
            PATCH=$((PATCH + 1))
            PRERELEASE=""
            ;;
        prerelease)
            if [ -z "$prerelease_tag" ]; then
                echo "❌ Error: --prerelease requires a tag (e.g., alpha, beta, rc)"
                exit 1
            fi
            # If there's already a prerelease, increment the number
            if [[ "$PRERELEASE" =~ ^${prerelease_tag}\.([0-9]+)$ ]]; then
                local prerelease_num=${BASH_REMATCH[1]}
                prerelease_num=$((prerelease_num + 1))
                PRERELEASE="${prerelease_tag}.${prerelease_num}"
            elif [[ "$PRERELEASE" =~ ^${prerelease_tag}$ ]]; then
                PRERELEASE="${prerelease_tag}.1"
            else
                # New prerelease tag
                PATCH=$((PATCH + 1))
                PRERELEASE="${prerelease_tag}.1"
            fi
            ;;
        *)
            echo "❌ Error: Unknown bump type: $bump_type"
            exit 1
            ;;
    esac

    if [ -n "$PRERELEASE" ]; then
        echo "${MAJOR}.${MINOR}.${PATCH}-${PRERELEASE}"
    else
        echo "${MAJOR}.${MINOR}.${PATCH}"
    fi
}

# Parse command line arguments
BUMP_TYPE=""
PRERELEASE_TAG=""
EXPLICIT_VERSION=""
PUSH=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --major)
            BUMP_TYPE="major"
            shift
            ;;
        --minor)
            BUMP_TYPE="minor"
            shift
            ;;
        --patch)
            BUMP_TYPE="patch"
            shift
            ;;
        --prerelease)
            BUMP_TYPE="prerelease"
            PRERELEASE_TAG="$2"
            if [ -z "$PRERELEASE_TAG" ]; then
                echo "❌ Error: --prerelease requires a tag"
                usage
            fi
            shift 2
            ;;
        --version)
            EXPLICIT_VERSION="$2"
            if [ -z "$EXPLICIT_VERSION" ]; then
                echo "❌ Error: --version requires a version number"
                usage
            fi
            shift 2
            ;;
        --push)
            PUSH=true
            shift
            ;;
        --help|-h)
            usage
            ;;
        *)
            echo "❌ Error: Unknown option: $1"
            usage
            ;;
    esac
done

# Validate that exactly one option was provided
if [ -z "$BUMP_TYPE" ] && [ -z "$EXPLICIT_VERSION" ]; then
    echo "❌ Error: Must specify one of --major, --minor, --patch, --prerelease, or --version"
    usage
fi

if [ -n "$BUMP_TYPE" ] && [ -n "$EXPLICIT_VERSION" ]; then
    echo "❌ Error: Cannot specify both bump type and explicit version"
    usage
fi

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "❌ Error: Not in a git repository"
    exit 1
fi

# Determine the new version
if [ -n "$EXPLICIT_VERSION" ]; then
    NEW_VERSION="$EXPLICIT_VERSION"
else
    CURRENT_VERSION=$(get_latest_version)
    echo "📋 Current version: v${CURRENT_VERSION}"
    NEW_VERSION=$(bump_version "$CURRENT_VERSION" "$BUMP_TYPE" "$PRERELEASE_TAG")
fi

# Validate version format (basic check)
if ! [[ "$NEW_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
    echo "❌ Error: Invalid version format: $NEW_VERSION"
    echo "   Expected format: MAJOR.MINOR.PATCH[-PRERELEASE]"
    exit 1
fi

# Check if tag already exists
TAG_NAME="v${NEW_VERSION}"
if git rev-parse "$TAG_NAME" > /dev/null 2>&1; then
    echo "❌ Error: Tag $TAG_NAME already exists"
    exit 1
fi

# Get timestamp
TIMESTAMP=$(date -u +"%Y-%m-%d %H:%M:%S UTC")

# Create annotated tag with timestamp
echo "🏷️  Creating tag: $TAG_NAME"
echo "   Timestamp: $TIMESTAMP"

git tag -a "$TAG_NAME" -m "Release $TAG_NAME

Timestamp: $TIMESTAMP"

echo ""
echo "✅ Successfully created tag: $TAG_NAME"

# Push if --push flag was provided
if [ "$PUSH" = true ]; then
    echo ""
    echo "🚀 Pushing tag to remote..."
    git push --follow-tags
    echo "✅ Successfully pushed tag: $TAG_NAME"
else
    echo ""
    echo "Next steps:"
    echo "  git push --follow-tags    # Push the tag and commits to remote"
    echo "  Or use: $0 --version $NEW_VERSION --push"
fi

