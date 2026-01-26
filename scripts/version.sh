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
                # New prerelease tag - don't bump version numbers, only change prerelease
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

# Check for uncommitted changes (excluding distribution manifests which we'll update)
if ! git diff --quiet -- ':!distributions/' 2>/dev/null; then
    echo "⚠️  Warning: You have uncommitted changes in your working directory"
    echo "   (excluding distribution manifests)"
    read -p "   Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "❌ Aborted"
        exit 1
    fi
fi

# Determine the new version
if [ -n "$EXPLICIT_VERSION" ]; then
    NEW_VERSION="$EXPLICIT_VERSION"
else
    CURRENT_VERSION=$(get_latest_version)
    echo "📋 Current version: v${CURRENT_VERSION}"
    NEW_VERSION=$(bump_version "$CURRENT_VERSION" "$BUMP_TYPE" "$PRERELEASE_TAG")
fi

# Check if version actually changed
CURRENT_MANIFEST_VERSION=""
if [ -f "distributions/npm/package.json" ]; then
    if command -v jq > /dev/null 2>&1; then
        CURRENT_MANIFEST_VERSION=$(jq -r '.version' distributions/npm/package.json 2>/dev/null || echo "")
    else
        CURRENT_MANIFEST_VERSION=$(grep -o '"version": "[^"]*"' distributions/npm/package.json | cut -d'"' -f4 || echo "")
    fi
fi

if [ "$NEW_VERSION" = "$CURRENT_MANIFEST_VERSION" ] && [ -n "$CURRENT_MANIFEST_VERSION" ]; then
    echo "ℹ️  Version $NEW_VERSION is already set in manifests"
    echo "   Skipping manifest updates"
    SKIP_MANIFEST_UPDATE=true
else
    SKIP_MANIFEST_UPDATE=false
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

# Function to update Homebrew formula
update_homebrew_formula() {
    local version=$1
    local formula_file="distributions/homebrew/hyperterse.rb"

    if [ ! -f "$formula_file" ]; then
        echo "⚠️  Warning: Homebrew formula not found: $formula_file"
        return
    fi

    echo "📝 Updating Homebrew formula..."

    # Update version line (URLs use #{version} interpolation, so they update automatically)
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS uses BSD sed
        sed -i '' "s/^  version \".*\"/  version \"${version}\"/" "$formula_file"
    else
        # Linux uses GNU sed
        sed -i "s/^  version \".*\"/  version \"${version}\"/" "$formula_file"
    fi

    echo "   ✓ Updated $formula_file"
}

# Function to update NPM package.json
update_npm_package() {
    local version=$1
    local package_file="distributions/npm/package.json"

    if [ ! -f "$package_file" ]; then
        echo "⚠️  Warning: package.json not found: $package_file"
        return
    fi

    echo "📝 Updating NPM package.json..."

    # Use a temporary file for JSON editing (more reliable than sed for JSON)
    if command -v jq > /dev/null 2>&1; then
        # Use jq if available (most reliable)
        jq ".version = \"${version}\"" "$package_file" > "${package_file}.tmp" && mv "${package_file}.tmp" "$package_file"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS sed fallback
        sed -i '' "s/\"version\": \".*\"/\"version\": \"${version}\"/" "$package_file"
    else
        # Linux sed fallback
        sed -i "s/\"version\": \".*\"/\"version\": \"${version}\"/" "$package_file"
    fi

    echo "   ✓ Updated $package_file"
}

# Update all distribution manifests
if [ "$SKIP_MANIFEST_UPDATE" = false ]; then
    echo ""
    echo "📦 Updating distribution manifests..."
    update_homebrew_formula "$NEW_VERSION"
    update_npm_package "$NEW_VERSION"

    # Check if there are any changes to commit
    if git diff --quiet distributions/; then
        echo "   ℹ️  No manifest changes detected"
    else
        echo ""
        echo "💾 Committing manifest changes..."
        git add distributions/homebrew/hyperterse.rb distributions/npm/package.json
        git commit -m "Update distribution manifests to v${NEW_VERSION}"
        echo "   ✓ Committed manifest changes"
    fi
fi

# Get timestamp
TIMESTAMP=$(date -u +"%Y-%m-%d %H:%M:%S UTC")

# Create annotated tag with timestamp
echo ""
echo "🏷️  Creating tag: $TAG_NAME"
echo "    Timestamp: $TIMESTAMP"

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

