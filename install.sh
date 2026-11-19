#!/bin/bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Repository information
OWNER="Nurysso"
REPO="vanish"
BINARY_NAME="vx"
FALLBACK_TAG="v0.9.6"   # used only if the latest release can't be resolved

# Fetch a URL to stdout using curl or wget
fetch() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$1"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$1"
    else
        echo -e "${RED}❌ Neither curl nor wget found. Please install one of them.${NC}" >&2
        exit 1
    fi
}

# Resolve the latest release tag (without needing jq)
get_latest_version() {
    local tag=""

    # 1) GitHub API
    tag=$(fetch "https://api.github.com/repos/$OWNER/$REPO/releases/latest" 2>/dev/null \
        | grep -m1 '"tag_name"' \
        | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' || true)

    # 2) Fallback: follow the /releases/latest redirect (not subject to API rate limits)
    if [ -z "$tag" ] && command -v curl >/dev/null 2>&1; then
        tag=$(curl -fsSLI -o /dev/null -w '%{url_effective}' \
            "https://github.com/$OWNER/$REPO/releases/latest" 2>/dev/null \
            | sed -E 's|.*/tag/||' || true)
    fi

    echo "$tag"
}

echo -e "${BLUE}🗑️  Vanish Installer${NC}"
echo -e "${BLUE}================${NC}"

# Use provided version, otherwise look up the latest release
if [ -n "${1:-}" ]; then
    VERSION="$1"
else
    echo -e "${YELLOW}🔍 Checking for the latest release...${NC}"
    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
        echo -e "${YELLOW}⚠️  Could not determine latest release, falling back to $FALLBACK_TAG${NC}"
        VERSION="$FALLBACK_TAG"
    fi
fi

# Remove 'v' prefix if present
VERSION="${VERSION#v}"

# Detect operating system
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux*) OS="linux" ;;
    darwin*) OS="macos" ;;
    mingw*|msys*|cygwin*)
        echo -e "${RED}❌ Windows is not supported for now${NC}"
        exit 1
        ;;
    *)
        echo -e "${RED}❌ Unsupported operating system: $OS${NC}"
        exit 1
        ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="x86_64" ;;
    aarch64|arm64) ARCH="aarch64" ;;
    armv7l) ARCH="arm" ;;
    *)
        echo -e "${RED}❌ Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# Construct download URL
BINARY_FILE="${BINARY_NAME}-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/$OWNER/$REPO/releases/download/v${VERSION}/${BINARY_FILE}"

echo -e "${YELLOW}📋 System Information:${NC}"
echo "   OS: $OS"
echo "   Architecture: $ARCH"
echo "   Version: v$VERSION"
echo "   Binary: $BINARY_FILE"
echo

# Determine installation directory (in order of preference)
if [ -w "/usr/local/bin" ] 2>/dev/null; then
    INSTALL_DIR="/usr/local/bin"
elif [ -d "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
else
    INSTALL_DIR="$HOME/bin"
fi
mkdir -p "$INSTALL_DIR"
TARGET_PATH="$INSTALL_DIR/$BINARY_NAME"

echo -e "${YELLOW}⬇ Downloading from: $DOWNLOAD_URL${NC}"

# Download to a temp file first so a failed download doesn't clobber an existing install
TMP_FILE=$(mktemp)
trap 'rm -f "$TMP_FILE"' EXIT

if command -v curl >/dev/null 2>&1; then
    DL_OK=0; curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE" || DL_OK=1
elif command -v wget >/dev/null 2>&1; then
    DL_OK=0; wget -q "$DOWNLOAD_URL" -O "$TMP_FILE" || DL_OK=1
else
    echo -e "${RED}❌ Neither curl nor wget found. Please install one of them.${NC}"
    exit 1
fi

if [ "$DL_OK" -ne 0 ]; then
    echo -e "${RED}❌ Failed to download $BINARY_NAME${NC}"
    echo -e "${RED}   URL: $DOWNLOAD_URL${NC}"
    exit 1
fi

chmod +x "$TMP_FILE"
mv "$TMP_FILE" "$TARGET_PATH"

# Verify installation
if [ -f "$TARGET_PATH" ]; then
    echo -e "${GREEN}✅ Successfully installed $BINARY_NAME v$VERSION${NC}"
    echo -e "${GREEN}   Location: $TARGET_PATH${NC}"

    # Check if directory is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo
        echo -e "${YELLOW}⚠️  Installation directory not in PATH${NC}"
        echo -e "${YELLOW}   Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):${NC}"
        echo -e "${BLUE}   export PATH=\"$INSTALL_DIR:\$PATH\"${NC}"
        echo
        echo -e "${YELLOW}   Or run: echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.bashrc${NC}"
    fi

    echo
    echo -e "${GREEN}🚀 Quick Start:${NC}"
    echo "   $BINARY_NAME --help    # Show help"
    echo "   $BINARY_NAME -t        # Choose theme"
    echo "   $BINARY_NAME file.txt  # Delete safely"
    echo "   $BINARY_NAME -l        # List cached files"
    echo
    echo -e "${BLUE}📚 Documentation: https://github.com/$OWNER/$REPO${NC}"
else
    echo -e "${RED}❌ Installation failed - binary not found at $TARGET_PATH${NC}"
    exit 1
fi
