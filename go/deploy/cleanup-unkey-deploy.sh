#!/bin/bash
# Cleanup script for Unkey Deploy services and components
# This script removes all installed services, configurations, and data

set -euo pipefail

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "============================================="
echo "Unkey Deploy Complete Cleanup Script"
echo "============================================="
echo ""
echo -e "${YELLOW}WARNING: This will remove all Unkey Deploy services and data!${NC}"
echo "Services to be removed:"
echo "  - metald"
echo "  - builderd"
echo "  - assetmanagerd"
echo "  - SPIRE Server and Agent"
echo "  - All VM bridges and network configurations"
echo "  - All data directories"
echo ""
read -p "Are you sure you want to continue? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cleanup cancelled."
    exit 0
fi

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: This script must be run as root${NC}"
    exit 1
fi

echo ""
echo "Starting cleanup process..."

# Function to safely stop and disable a service
stop_and_disable_service() {
    local service=$1
    if systemctl list-unit-files | grep -q "^${service}.service"; then
        echo "Stopping and disabling ${service}..."
        systemctl stop "${service}" 2>/dev/null || true
        systemctl disable "${service}" 2>/dev/null || true
    fi
}

# Function to remove systemd service files
remove_service_files() {
    local service=$1
    echo "Removing ${service} service files..."
    rm -f "/etc/systemd/system/${service}.service"
    rm -f "/etc/systemd/system/${service}@.service"
    rm -f "/usr/lib/systemd/system/${service}.service"
    rm -f "/usr/lib/systemd/system/${service}@.service"
}

# 1. Stop all services
echo ""
echo "=== Stopping Services ==="
stop_and_disable_service "metald"
stop_and_disable_service "metald-bridge-8"
stop_and_disable_service "metald-bridge-32"
stop_and_disable_service "builderd"
stop_and_disable_service "assetmanagerd"
stop_and_disable_service "spire-agent"
stop_and_disable_service "spire-server"

# 2. Kill any remaining Firecracker processes
echo ""
echo "=== Cleaning up Firecracker VMs ==="
pkill -9 firecracker 2>/dev/null || true
pkill -9 jailer 2>/dev/null || true

# Clean up any remaining VM tap interfaces
for tap in $(ip link show | grep -o 'tap[0-9a-f_-]*' | sort -u); do
    echo "Removing tap interface: $tap"
    ip link delete "$tap" 2>/dev/null || true
done

# Clean up veth interfaces
for veth in $(ip link show | grep -o 'vh_[0-9a-f]*' | sort -u); do
    echo "Removing veth interface: $veth"
    ip link delete "$veth" 2>/dev/null || true
done

# 3. Remove network bridges
echo ""
echo "=== Removing Network Bridges ==="
for i in {0..31}; do
    bridge="br-tenant-$i"
    if ip link show "$bridge" &>/dev/null; then
        echo "Removing bridge: $bridge"
        ip link set "$bridge" down 2>/dev/null || true
        ip link delete "$bridge" 2>/dev/null || true
    fi
done

# Remove systemd-networkd configurations
echo "Removing network configurations..."
rm -rf /etc/systemd/network/10-br-tenant-*.net{dev,work}
rm -rf /run/systemd/network/10-br-tenant-*.net{dev,work}

# 4. Remove binaries
echo ""
echo "=== Removing Binaries ==="
binaries=(
    "/usr/local/bin/metald"
    "/usr/local/bin/metald-cli"
    "/usr/local/bin/metald-init"
    "/usr/local/bin/builderd"
    "/usr/local/bin/builderd-cli"
    "/usr/local/bin/assetmanagerd"
    "/usr/local/bin/assetmanagerd-cli"
    "/usr/local/bin/firecracker"
    "/usr/local/bin/jailer"
    "/opt/spire/bin/spire-server"
    "/opt/spire/bin/spire-agent"
    "/opt/spire/bin/spire"
)

for binary in "${binaries[@]}"; do
    if [ -f "$binary" ]; then
        echo "Removing: $binary"
        rm -f "$binary"
    fi
done

# 5. Remove service files
echo ""
echo "=== Removing Service Files ==="
remove_service_files "metald"
remove_service_files "metald-bridge-8"
remove_service_files "metald-bridge-32"
remove_service_files "builderd"
remove_service_files "assetmanagerd"
remove_service_files "spire-server"
remove_service_files "spire-agent"

# 6. Remove configuration files
echo ""
echo "=== Removing Configuration Files ==="
rm -rf /etc/metald
rm -rf /etc/builderd
rm -rf /etc/assetmanagerd
rm -rf /etc/spire
rm -rf /etc/default/unkey-deploy
rm -f /etc/default/metald
rm -f /etc/default/builderd
rm -f /etc/default/assetmanagerd

# 7. Remove data directories
echo ""
echo "=== Removing Data Directories ==="
echo -e "${YELLOW}Warning: This will delete all VM images and assets!${NC}"
read -p "Remove all data directories? [y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    # Service data directories
    rm -rf /opt/metald
    rm -rf /opt/builderd
    rm -rf /opt/assetmanagerd
    rm -rf /opt/vm-assets
    rm -rf /opt/spire
    
    # Runtime directories
    rm -rf /var/lib/metald
    rm -rf /var/lib/builderd
    rm -rf /var/lib/assetmanagerd
    rm -rf /var/lib/spire
    rm -rf /var/lib/firecracker
    
    # Jailer directories
    rm -rf /srv/jailer
    rm -rf /var/run/firecracker
    
    # Log directories
    rm -rf /var/log/metald
    rm -rf /var/log/builderd
    rm -rf /var/log/assetmanagerd
    rm -rf /var/log/spire
    
    echo -e "${GREEN}✓${NC} Data directories removed"
else
    echo "Skipping data directory removal"
fi

# 8. Remove users and groups
echo ""
echo "=== Removing Service Users ==="
for user in metald builderd assetmanagerd firecracker spire; do
    if id -u "$user" &>/dev/null; then
        echo "Removing user: $user"
        userdel "$user" 2>/dev/null || true
    fi
    if getent group "$user" &>/dev/null; then
        echo "Removing group: $user"
        groupdel "$user" 2>/dev/null || true
    fi
done

# 9. Clean up iptables rules
echo ""
echo "=== Cleaning up iptables rules ==="
# Remove FORWARD rules for VM bridges
for i in {0..31}; do
    iptables -D FORWARD -i br-tenant-$i -j ACCEPT 2>/dev/null || true
    iptables -D FORWARD -o br-tenant-$i -j ACCEPT 2>/dev/null || true
done

# Remove NAT rules
iptables -t nat -F 2>/dev/null || true
iptables -t nat -X 2>/dev/null || true

# 10. Clean up cgroups
echo ""
echo "=== Cleaning up cgroups ==="
if [ -d /sys/fs/cgroup/firecracker ]; then
    rmdir /sys/fs/cgroup/firecracker 2>/dev/null || true
fi

# Clean up any VM-specific cgroups
for cg in $(find /sys/fs/cgroup -name "*firecracker*" -type d 2>/dev/null); do
    rmdir "$cg" 2>/dev/null || true
done

# 11. Reload systemd
echo ""
echo "=== Reloading systemd ==="
systemctl daemon-reload
systemctl restart systemd-networkd

# 12. Clean up any remaining artifacts
echo ""
echo "=== Final cleanup ==="
# Remove any temporary VM files
rm -rf /tmp/firecracker-*
rm -rf /tmp/vm-*
rm -f /tmp/*-vm-console.log

# Remove any socket files
rm -f /var/run/firecracker.sock*
rm -f /var/run/metald.sock
rm -f /var/run/builderd.sock
rm -f /var/run/assetmanagerd.sock
rm -f /var/lib/spire/agent/agent.sock

# Clean up any remaining systemd runtime directories
rm -rf /run/systemd/system/metald.service.d
rm -rf /run/systemd/system/builderd.service.d
rm -rf /run/systemd/system/assetmanagerd.service.d

echo ""
echo "============================================="
echo -e "${GREEN}✓ Cleanup completed successfully!${NC}"
echo "============================================="
echo ""
echo "The following have been removed:"
echo "  - All Unkey Deploy services and binaries"
echo "  - All network bridges and configurations"
echo "  - All service users and groups"
echo "  - All configuration files"
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "  - All data directories and VM assets"
fi
echo ""
echo "System has been restored to pre-installation state."
echo ""
echo "Note: If you want to reinstall, you'll need to:"
echo "  1. Reinstall SPIRE Server and Agent"
echo "  2. Reinstall and configure all services"
echo "  3. Re-run network bridge setup"
echo "  4. Re-download base VM assets"
