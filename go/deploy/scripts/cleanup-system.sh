#!/bin/bash
# System cleanup script for Unkey services
# AIDEV-NOTE: Provides options for different levels of cleanup

set -euo pipefail

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Services list
SERVICES=("metald" "builderd" "billaged" "assetmanagerd")
SPIRE_SERVICES=("spire-server" "spire-agent")
ALL_USERS=("metald" "billaged" "builderd" "assetmanagerd" "spire-server" "spire-agent")

# Parse command line options
REMOVE_DATA=false
REMOVE_USERS=false
REMOVE_FIRECRACKER=false
FORCE=false

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --remove-data        Remove all service data directories"
    echo "  --remove-users       Remove service users"
    echo "  --remove-firecracker Remove Firecracker and jailer binaries"
    echo "  --force              Skip confirmation prompts"
    echo "  --help               Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                     # Basic cleanup (stop and uninstall)"
    echo "  $0 --remove-data       # Also remove data directories"
    echo "  $0 --remove-users      # Also remove service users"
    echo "  $0 --remove-data --remove-users --remove-firecracker --force  # Complete cleanup"
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --remove-data)
            REMOVE_DATA=true
            shift
            ;;
        --remove-users)
            REMOVE_USERS=true
            shift
            ;;
        --remove-firecracker)
            REMOVE_FIRECRACKER=true
            shift
            ;;
        --force)
            FORCE=true
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Confirmation
if [ "$FORCE" != "true" ]; then
    echo "==================================="
    echo "Unkey Services Cleanup"
    echo "==================================="
    echo ""
    echo "This will:"
    echo "  ✓ Stop all running services"
    echo "  ✓ Uninstall service binaries"
    echo "  ✓ Remove systemd service files"
    
    if [ "$REMOVE_DATA" == "true" ]; then
        echo -e "  ${YELLOW}✓ Remove all service data${NC}"
    else
        echo "  ✗ Preserve service data (use --remove-data to remove)"
    fi
    
    if [ "$REMOVE_USERS" == "true" ]; then
        echo -e "  ${YELLOW}✓ Remove service users${NC}"
    else
        echo "  ✗ Preserve service users (use --remove-users to remove)"
    fi
    
    if [ "$REMOVE_FIRECRACKER" == "true" ]; then
        echo -e "  ${YELLOW}✓ Remove Firecracker and jailer${NC}"
    else
        echo "  ✗ Preserve Firecracker (use --remove-firecracker to remove)"
    fi
    
    echo ""
    read -p "Continue? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Cleanup cancelled."
        exit 0
    fi
fi

echo ""
echo "Starting cleanup..."
echo ""

# Stop all services
echo "=== Stopping Services ==="
for service in "${SERVICES[@]}"; do
    if systemctl is-active --quiet "$service"; then
        echo -n "Stopping $service... "
        sudo systemctl stop "$service" && echo -e "${GREEN}✓${NC}" || echo -e "${RED}✗${NC}"
    else
        echo "$service is not running"
    fi
done

for service in "${SPIRE_SERVICES[@]}"; do
    if systemctl is-active --quiet "$service"; then
        echo -n "Stopping $service... "
        sudo systemctl stop "$service" && echo -e "${GREEN}✓${NC}" || echo -e "${RED}✗${NC}"
    else
        echo "$service is not running"
    fi
done

# Stop observability stack
echo -n "Stopping observability stack... "
if podman ps | grep -q otel-lgtm; then
    podman stop otel-lgtm >/dev/null 2>&1 && podman rm otel-lgtm >/dev/null 2>&1 && echo -e "${GREEN}✓${NC}" || echo -e "${RED}✗${NC}"
else
    echo "not running"
fi

echo ""

# Uninstall services
echo "=== Uninstalling Services ==="
for service in "${SERVICES[@]}"; do
    if [ -f "/usr/local/bin/$service" ] || [ -f "/etc/systemd/system/$service.service" ]; then
        echo -n "Uninstalling $service... "
        sudo rm -f "/usr/local/bin/$service"
        sudo rm -f "/etc/systemd/system/$service.service"
        echo -e "${GREEN}✓${NC}"
    else
        echo "$service is not installed"
    fi
done

# Uninstall SPIRE
if [ -f "/opt/spire/bin/spire-server" ] || [ -f "/opt/spire/bin/spire-agent" ]; then
    echo -n "Uninstalling SPIRE... "
    sudo rm -f /opt/spire/bin/spire-server /opt/spire/bin/spire-agent
    sudo rm -f /etc/systemd/system/spire-server.service /etc/systemd/system/spire-agent.service
    echo -e "${GREEN}✓${NC}"
else
    echo "SPIRE is not installed"
fi

# Reload systemd
echo -n "Reloading systemd... "
sudo systemctl daemon-reload && echo -e "${GREEN}✓${NC}"

echo ""

# Remove data directories if requested
if [ "$REMOVE_DATA" == "true" ]; then
    echo "=== Removing Data Directories ==="
    
    DATA_DIRS=(
        "/opt/metald"
        "/opt/billaged"
        "/opt/builderd"
        "/opt/assetmanagerd"
        "/opt/vm-assets"
        "/opt/spire"
        "/var/lib/spire"
        "/etc/spire"
        "/run/spire"
    )
    
    for dir in "${DATA_DIRS[@]}"; do
        if [ -d "$dir" ]; then
            echo -n "Removing $dir... "
            sudo rm -rf "$dir" && echo -e "${GREEN}✓${NC}" || echo -e "${RED}✗${NC}"
        fi
    done
    echo ""
fi

# Remove users if requested
if [ "$REMOVE_USERS" == "true" ]; then
    echo "=== Removing Service Users ==="
    
    for user in "${ALL_USERS[@]}"; do
        if id "$user" >/dev/null 2>&1; then
            echo -n "Removing user $user... "
            sudo userdel "$user" && echo -e "${GREEN}✓${NC}" || echo -e "${RED}✗${NC}"
        fi
    done
    echo ""
fi

echo "==================================="
echo -e "${GREEN}✓ Cleanup complete!${NC}"
echo "==================================="

if [ "$REMOVE_DATA" != "true" ]; then
    echo ""
    echo -e "${YELLOW}Note:${NC} Service data directories were preserved."
    echo "To remove them, run: $0 --remove-data"
fi

if [ "$REMOVE_USERS" != "true" ]; then
    echo ""
    echo -e "${YELLOW}Note:${NC} Service users were preserved."
    echo "To remove them, run: $0 --remove-users"
fi

# Remove Firecracker if requested
if [ "$REMOVE_FIRECRACKER" == "true" ]; then
    echo "=== Removing Firecracker ==="
    
    if [ -f "/usr/local/bin/firecracker" ] || [ -f "/usr/local/bin/jailer" ]; then
        echo -n "Removing Firecracker and jailer... "
        ./scripts/install-firecracker.sh --uninstall >/dev/null 2>&1 && echo -e "${GREEN}✓${NC}" || echo -e "${RED}✗${NC}"
    else
        echo "Firecracker is not installed"
    fi
    echo ""
fi

if [ "$REMOVE_FIRECRACKER" != "true" ]; then
    if [ -f "/usr/local/bin/firecracker" ] || [ -f "/usr/local/bin/jailer" ]; then
        echo ""
        echo -e "${YELLOW}Note:${NC} Firecracker was preserved."
        echo "To remove it, run: $0 --remove-firecracker"
    fi
fi