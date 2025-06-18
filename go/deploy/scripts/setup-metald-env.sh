#!/bin/bash
# Setup environment for metald service
# AIDEV-NOTE: Creates necessary directories and permissions for metald to manage network namespaces

set -euo pipefail

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "Setting up metald environment..."

# Create network namespace directory
if [ ! -d /run/netns ]; then
    echo -n "Creating /run/netns directory... "
    sudo mkdir -p /run/netns
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${GREEN}✓${NC} /run/netns already exists"
fi

# Ensure the directory persists across reboots by creating a tmpfiles.d entry
echo -n "Creating tmpfiles.d configuration... "
echo "d /run/netns 0755 root root -" | sudo tee /etc/tmpfiles.d/metald-netns.conf > /dev/null
echo -e "${GREEN}✓${NC}"

# Create VM asset directories if they don't exist
if [ ! -d /opt/vm-assets ]; then
    echo -n "Creating /opt/vm-assets directory... "
    sudo mkdir -p /opt/vm-assets
    sudo chown metald:metald /opt/vm-assets
    echo -e "${GREEN}✓${NC}"
fi

# Create metald data directory
if [ ! -d /opt/metald ]; then
    echo -n "Creating /opt/metald directory... "
    sudo mkdir -p /opt/metald
    sudo chown metald:metald /opt/metald
    echo -e "${GREEN}✓${NC}"
fi

# Check if metald user exists
if ! id metald >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠${NC} metald user doesn't exist. Run 'make -C metald create-user' first"
fi

echo ""
echo -e "${GREEN}✓ Metald environment setup complete!${NC}"