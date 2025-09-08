#!/bin/bash
# Install buf for protobuf generation
# AIDEV-NOTE: Installs the buf CLI tool required for building services with protobuf

set -euo pipefail

# Configuration
BUF_VERSION="${BUF_VERSION:-v1.57.0}"
ARCH="${ARCH:-$(uname -m)}"
OS="${OS:-$(uname -s)}"
INSTALL_DIR="/usr/local/bin"

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Map architecture names
case "$ARCH" in
    x86_64|amd64)
        ARCH="x86_64"
        ;;
    aarch64|arm64)
        ARCH="aarch64"
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# Map OS names
case "$OS" in
    Linux|linux)
        OS="Linux"
        ;;
    Darwin|darwin)
        OS="Darwin"
        ;;
    *)
        echo -e "${RED}Error: Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

# Check for uninstall flag
if [ "${1:-}" = "--uninstall" ]; then
    echo "Uninstalling buf..."
    if [ -f "$INSTALL_DIR/buf" ]; then
        sudo rm -f "$INSTALL_DIR/buf"
        echo -e "${GREEN}✓${NC} Removed buf"
    else
        echo "buf is not installed"
    fi
    exit 0
fi

echo "Installing buf ${BUF_VERSION} for ${OS}-${ARCH}..."

# Download URL
DOWNLOAD_URL="https://github.com/bufbuild/buf/releases/download/${BUF_VERSION}/buf-${OS}-${ARCH}"

# Create temporary directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Download buf
echo "Downloading buf..."
if ! curl -sL "$DOWNLOAD_URL" -o "$TEMP_DIR/buf"; then
    echo -e "${RED}Error: Failed to download buf from $DOWNLOAD_URL${NC}"
    exit 1
fi

# Make executable
chmod +x "$TEMP_DIR/buf"

# Verify download
if ! "$TEMP_DIR/buf" --version >/dev/null 2>&1; then
    echo -e "${RED}Error: Downloaded binary is not valid${NC}"
    exit 1
fi

# Install
echo "Installing buf to $INSTALL_DIR..."
if [ "$EUID" -ne 0 ] && ! sudo -n true 2>/dev/null; then
    echo -e "${RED}Error: Installation requires root privileges${NC}"
    echo "Please run with sudo: sudo $0"
    exit 1
fi

sudo install -m 755 "$TEMP_DIR/buf" "$INSTALL_DIR/buf"

# Verify installation
if buf --version >/dev/null 2>&1; then
    echo -e "${GREEN}✓ buf installed successfully!${NC}"
    echo "Version: $(buf --version)"
else
    echo -e "${RED}Error: buf installation verification failed${NC}"
    exit 1
fi
