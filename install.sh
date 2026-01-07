#!/bin/bash

# Hyperterse Installation Script
# This script downloads and installs the Hyperterse binary for your system
#
# Usage:
#   curl -fsSL hyperterse.ai/install.sh | sh
#   # or
#   curl -fsSL hyperterse.ai/install.sh | bash
#
# Environment variables:
#   VERSION      - Version to install (default: latest)
#   INSTALL_DIR  - Installation directory (default: ~/.local/bin)
#   BASE_URL     - Base URL for downloads (default: GitHub releases)
#
# Expected binary naming convention:
#   hyperterse-{os}-{arch} (e.g., hyperterse-linux-amd64, hyperterse-darwin-arm64)
#   For Windows: hyperterse-windows-amd64.exe

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="hyperterse"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-latest}"
BASE_URL="${BASE_URL:-https://github.com/hyperterse/hyperterse/releases}"

# Function to print colored output
info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1" >&2
}

# Function to detect OS
detect_os() {
    local os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        linux*)
            echo "linux"
            ;;
        darwin*)
            echo "darwin"
            ;;
        msys*|cygwin*|mingw*)
            echo "windows"
            ;;
        *)
            error "Unsupported operating system: $os"
            exit 1
            ;;
    esac
}

# Function to detect architecture
detect_arch() {
    local arch="$(uname -m | tr '[:upper:]' '[:lower:]')"
    case "$arch" in
        x86_64|amd64)
            echo "amd64"
            ;;
        arm64|aarch64)
            echo "arm64"
            ;;
        armv7l)
            echo "armv7"
            ;;
        armv6l)
            echo "armv6"
            ;;
        i386|i686)
            echo "386"
            ;;
        *)
            error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# Function to get download URL
get_download_url() {
    local os=$1
    local arch=$2
    local binary_suffix=""

    # Add .exe suffix for Windows
    if [ "$os" = "windows" ]; then
        binary_suffix=".exe"
    fi

    if [ "$VERSION" = "latest" ]; then
        # Try GitHub releases API first to get the latest version
        if [[ "$BASE_URL" == *"github.com"* ]]; then
            local api_url="${BASE_URL%/releases}/releases/latest"
            local latest_tag=""

            if command_exists curl; then
                latest_tag=$(curl -sL "$api_url" | grep '"tag_name":' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/' | head -1)
            elif command_exists wget; then
                latest_tag=$(wget -qO- "$api_url" | grep '"tag_name":' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/' | head -1)
            fi

            if [ -n "$latest_tag" ]; then
                # Remove 'v' prefix if present
                latest_tag="${latest_tag#v}"
                local version_url="${BASE_URL}/download/v${latest_tag}/hyperterse-${os}-${arch}${binary_suffix}"
                echo "$version_url"
                return
            fi
        fi

        # Fallback to latest/download pattern
        local latest_url="${BASE_URL}/latest/download/hyperterse-${os}-${arch}${binary_suffix}"
        echo "$latest_url"
    else
        # For specific versions
        local version_url="${BASE_URL}/download/v${VERSION}/hyperterse-${os}-${arch}${binary_suffix}"
        echo "$version_url"
    fi
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to download file
download_file() {
    local url=$1
    local output=$2

    info "Downloading from $url..."

    if command_exists curl; then
        if curl -fL --progress-bar -o "$output" "$url"; then
            return 0
        fi
    elif command_exists wget; then
        if wget -q --show-progress -O "$output" "$url"; then
            return 0
        fi
    else
        error "Neither curl nor wget is installed. Please install one of them."
        exit 1
    fi

    return 1
}

# Function to install binary
install_binary() {
    local binary_path=$1
    local install_path="$INSTALL_DIR/$BINARY_NAME"

    # Create install directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"

    # Copy binary to install directory
    cp "$binary_path" "$install_path"
    chmod +x "$install_path"

    success "Binary installed to $install_path"

    # Check if install directory is in PATH
    if echo "$PATH" | grep -q "$INSTALL_DIR"; then
        success "Installation complete! You can now run 'hyperterse' from anywhere."
    else
        warning "Installation directory ($INSTALL_DIR) is not in your PATH."
        echo ""
        info "To use hyperterse, add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo -e "  ${GREEN}export PATH=\"\$PATH:$INSTALL_DIR\"${NC}"
        echo ""
        info "Or run hyperterse directly:"
        echo -e "  ${GREEN}$install_path${NC}"
    fi
}

# Main installation function
main() {
    echo ""
    info "Hyperterse Installation Script"
    echo ""

    # Detect OS and architecture
    OS=$(detect_os)
    ARCH=$(detect_arch)

    info "Detected OS: $OS"
    info "Detected Architecture: $ARCH"
    echo ""

    # Handle Windows differently
    if [ "$OS" = "windows" ]; then
        BINARY_NAME="${BINARY_NAME}.exe"
        INSTALL_DIR="${INSTALL_DIR:-$HOME/AppData/Local/hyperterse}"
    fi

    # Create temporary directory for download
    TEMP_DIR=$(mktemp -d)
    trap "rm -rf $TEMP_DIR" EXIT

    DOWNLOAD_FILE="$TEMP_DIR/$BINARY_NAME"

    # Get download URL
    DOWNLOAD_URL=$(get_download_url "$OS" "$ARCH")

    # Download the binary
    if ! download_file "$DOWNLOAD_URL" "$DOWNLOAD_FILE"; then
        error "Failed to download binary from $DOWNLOAD_URL"
        echo ""
        info "This might mean:"
        echo "  1. The binary for your platform ($OS/$ARCH) is not available"
        echo "  2. The version ($VERSION) does not exist"
        echo "  3. There's a network connectivity issue"
        echo ""
        info "You can try building from source:"
        echo "  git clone https://github.com/hyperterse/hyperterse.git"
        echo "  cd hyperterse && make setup && make build"
        exit 1
    fi

    # Verify the downloaded file
    if [ ! -f "$DOWNLOAD_FILE" ] || [ ! -s "$DOWNLOAD_FILE" ]; then
        error "Downloaded file is empty or missing"
        exit 1
    fi

    success "Download complete!"
    echo ""

    # Check if running in non-interactive mode (piped from curl)
    # If stdin is not a terminal, skip the prompt and install automatically
    if [ -t 0 ] && [ -z "$SKIP_INSTALL_PROMPT" ]; then
        echo -n "Install to $INSTALL_DIR? [Y/n]: "
        read -r response
        if [[ "$response" =~ ^[Nn]$ ]]; then
            info "Binary downloaded to: $DOWNLOAD_FILE"
            info "You can manually move it to your desired location."
            exit 0
        fi
    else
        info "Installing to $INSTALL_DIR..."
    fi

    # Install the binary
    install_binary "$DOWNLOAD_FILE"

    echo ""
    success "Hyperterse installed successfully!"
    echo ""
    info "Get started by running:"
    echo -e "  ${GREEN}hyperterse -file config.yaml${NC}"
    echo ""
}

# Run main function
main "$@"

