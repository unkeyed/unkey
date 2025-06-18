#!/bin/bash
# Install or uninstall Firecracker and Jailer from GitHub releases
# AIDEV-NOTE: Installs both firecracker and jailer binaries which are required for production deployments

set -euo pipefail

# Configuration
FIRECRACKER_VERSION="${FIRECRACKER_VERSION:-v1.12.0}"
ARCH="${ARCH:-x86_64}"
INSTALL_DIR="/usr/local/bin"

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check for uninstall flag
if [ "${1:-}" = "--uninstall" ]; then
    echo "Uninstalling Firecracker..."
    if [ "$EUID" -ne 0 ] && ! sudo -n true 2>/dev/null; then
        echo -e "${RED}Error: Uninstall requires root privileges${NC}"
        echo "Please run with sudo: sudo $0 --uninstall"
        exit 1
    fi
    
    removed=0
    if [ -f "$INSTALL_DIR/firecracker" ]; then
        sudo rm -f "$INSTALL_DIR/firecracker"
        echo -e "${GREEN}✓${NC} Removed firecracker"
        removed=1
    fi
    
    if [ -f "$INSTALL_DIR/jailer" ]; then
        sudo rm -f "$INSTALL_DIR/jailer"
        echo -e "${GREEN}✓${NC} Removed jailer"
        removed=1
    fi
    
    # Ask about removing user and directories
    if [ $removed -eq 1 ]; then
        echo ""
        read -p "Remove firecracker user and directories? [y/N] " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            if id -u firecracker >/dev/null 2>&1; then
                sudo userdel firecracker
                echo -e "${GREEN}✓${NC} Removed firecracker user"
            fi
            
            if [ -d "/var/lib/firecracker" ]; then
                sudo rm -rf /var/lib/firecracker
                echo -e "${GREEN}✓${NC} Removed /var/lib/firecracker"
            fi
            
            if [ -d "/srv/jailer" ]; then
                sudo rm -rf /srv/jailer
                echo -e "${GREEN}✓${NC} Removed /srv/jailer"
            fi
            
            if [ -d "/sys/fs/cgroup/firecracker" ]; then
                sudo rmdir /sys/fs/cgroup/firecracker 2>/dev/null || true
                echo -e "${GREEN}✓${NC} Removed firecracker cgroup"
            fi
        fi
    fi
    
    if [ $removed -eq 0 ]; then
        echo "Firecracker was not installed"
    else
        echo -e "${GREEN}✓ Firecracker uninstalled successfully${NC}"
    fi
    exit 0
fi

echo "==================================="
echo "Firecracker Installation"
echo "==================================="
echo "Version: $FIRECRACKER_VERSION"
echo "Architecture: $ARCH"
echo ""

# Check if running as root or with sudo
if [ "$EUID" -ne 0 ] && ! sudo -n true 2>/dev/null; then
    echo -e "${RED}Error: This script requires root privileges${NC}"
    echo "Please run with sudo: sudo $0"
    exit 1
fi

# Create temporary directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

echo "Downloading Firecracker release..."
RELEASE_URL="https://github.com/firecracker-microvm/firecracker/releases/download/${FIRECRACKER_VERSION}/firecracker-${FIRECRACKER_VERSION}-${ARCH}.tgz"

# Download the release
if ! curl -sL "$RELEASE_URL" -o "$TEMP_DIR/firecracker.tgz"; then
    echo -e "${RED}Error: Failed to download Firecracker from $RELEASE_URL${NC}"
    echo "Please check the version and try again."
    exit 1
fi

# Extract the tarball
echo "Extracting Firecracker..."
cd "$TEMP_DIR"
if ! tar -xzf firecracker.tgz; then
    echo -e "${RED}Error: Failed to extract Firecracker archive${NC}"
    exit 1
fi

# Find the release directory
RELEASE_DIR=$(find . -type d -name "release-${FIRECRACKER_VERSION}-${ARCH}" | head -1)
if [ -z "$RELEASE_DIR" ]; then
    echo -e "${RED}Error: Could not find release directory${NC}"
    echo "Archive contents:"
    tar -tzf firecracker.tgz
    exit 1
fi

# Install firecracker binary
echo "Installing firecracker binary..."
if [ -f "$RELEASE_DIR/firecracker-${FIRECRACKER_VERSION}-${ARCH}" ]; then
    sudo install -m 755 "$RELEASE_DIR/firecracker-${FIRECRACKER_VERSION}-${ARCH}" "$INSTALL_DIR/firecracker"
    echo -e "${GREEN}✓${NC} Installed firecracker to $INSTALL_DIR/firecracker"
else
    echo -e "${RED}Error: firecracker binary not found in release${NC}"
    exit 1
fi

# Install jailer binary
echo "Installing jailer binary..."
if [ -f "$RELEASE_DIR/jailer-${FIRECRACKER_VERSION}-${ARCH}" ]; then
    sudo install -m 755 "$RELEASE_DIR/jailer-${FIRECRACKER_VERSION}-${ARCH}" "$INSTALL_DIR/jailer"
    echo -e "${GREEN}✓${NC} Installed jailer to $INSTALL_DIR/jailer"
else
    echo -e "${RED}Error: jailer binary not found in release${NC}"
    exit 1
fi

# Verify installation
echo ""
echo "Verifying installation..."
if firecracker --version >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} firecracker: $(firecracker --version)"
else
    echo -e "${RED}✗${NC} firecracker verification failed"
fi

if jailer --version >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} jailer: $(jailer --version)"
else
    echo -e "${RED}✗${NC} jailer verification failed"
fi

# Check KVM access
echo ""
echo "Checking KVM access..."
if [ -e /dev/kvm ]; then
    if [ -r /dev/kvm ] && [ -w /dev/kvm ]; then
        echo -e "${GREEN}✓${NC} KVM is accessible"
    else
        echo -e "${YELLOW}⚠${NC} KVM exists but may not be accessible to current user"
        echo "   You may need to add your user to the kvm group:"
        echo "   sudo usermod -aG kvm $USER"
    fi
else
    echo -e "${RED}✗${NC} /dev/kvm not found - virtualization may not be enabled"
fi

# Set up jailer requirements for production
echo ""
echo "Setting up jailer requirements..."

# Create firecracker user if it doesn't exist
if ! id -u firecracker >/dev/null 2>&1; then
    echo -n "Creating firecracker user... "
    sudo useradd -r -s /bin/false -d /var/lib/firecracker firecracker
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${GREEN}✓${NC} firecracker user already exists"
fi

# Create jailer directory structure
echo -n "Creating jailer directories... "
sudo mkdir -p /srv/jailer
sudo mkdir -p /var/lib/firecracker/images
sudo mkdir -p /var/lib/firecracker/kernels
sudo chown -R firecracker:firecracker /var/lib/firecracker
sudo chmod 750 /var/lib/firecracker
echo -e "${GREEN}✓${NC}"

# Set capabilities on jailer binary
echo -n "Setting jailer capabilities... "
sudo setcap cap_sys_admin,cap_dac_override,cap_dac_read_search,cap_fowner,cap_kill,cap_sys_ptrace,cap_net_admin,cap_sys_chroot,cap_mknod,cap_setfcap,cap_setuid,cap_setgid,cap_sys_rawio,cap_sys_resource+ep $INSTALL_DIR/jailer
echo -e "${GREEN}✓${NC}"

# Configure cgroup v2 if needed
echo ""
echo "Checking cgroup configuration..."
# Check if cgroup v2 is active by looking for the controllers file
if [ -f /sys/fs/cgroup/cgroup.controllers ]; then
    echo -e "${GREEN}✓${NC} cgroup v2 detected"
    
    # Show available controllers
    controllers=$(cat /sys/fs/cgroup/cgroup.controllers)
    echo "Available controllers: $controllers"
    
    # Create a cgroup for firecracker if it doesn't exist
    if [ ! -d /sys/fs/cgroup/firecracker ]; then
        echo -n "Creating firecracker cgroup... "
        sudo mkdir -p /sys/fs/cgroup/firecracker
        echo -e "${GREEN}✓${NC}"
    fi
else
    echo -e "${YELLOW}⚠${NC} cgroup v1 detected. Firecracker will work but cgroup v2 is recommended"
    
    # Check if the system can support cgroup v2
    if grep -q cgroup2 /proc/filesystems; then
        echo ""
        echo "Your system supports cgroup v2. To enable it (optional):"
        echo ""
        echo "For systemd-based systems (Fedora/Ubuntu):"
        echo "  1. Add kernel parameter:"
        echo "     sudo grubby --update-kernel=ALL --args='systemd.unified_cgroup_hierarchy=1'"
        echo "  2. Reboot your system"
        echo ""
        echo "Note: Firecracker works fine with cgroup v1, this is just a recommendation."
        echo "For Fedora 31+ and Ubuntu 21.10+, cgroup v2 is usually the default."
    fi
fi

echo ""
echo "==================================="
echo -e "${GREEN}✓ Firecracker installation and setup complete!${NC}"
echo "==================================="
echo ""
echo "Installed components:"
echo "  - firecracker: $INSTALL_DIR/firecracker"
echo "  - jailer: $INSTALL_DIR/jailer (with capabilities set)"
echo ""
echo "Created resources:"
echo "  - User: firecracker"
echo "  - Directories: /var/lib/firecracker/{images,kernels}"
echo "  - Jailer root: /srv/jailer"
echo ""
echo "The jailer is now ready for production use!"
echo ""
echo "For more details, see: https://github.com/firecracker-microvm/firecracker/blob/main/docs/jailer.md"