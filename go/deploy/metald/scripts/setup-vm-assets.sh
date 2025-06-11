#!/bin/bash

# Firecracker VM Assets Setup Script
# This script downloads and sets up VM assets required for Firecracker VM creation
# Run as root: sudo ./scripts/setup-vm-assets.sh

set -euo pipefail

# Configuration
ASSETS_DIR="/opt/vm-assets"
FIRECRACKER_VERSION="v1.6.0"
KERNEL_URL="https://s3.amazonaws.com/spec.ccfc.min/img/quickstart_guide/x86_64/kernels/vmlinux.bin"
ROOTFS_URL="https://s3.amazonaws.com/spec.ccfc.min/img/quickstart_guide/x86_64/rootfs/bionic.rootfs.ext4"
FIRECRACKER_URL="https://github.com/firecracker-microvm/firecracker/releases/download/${FIRECRACKER_VERSION}/firecracker-${FIRECRACKER_VERSION}-x86_64.tgz"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        log_info "Usage: sudo $0"
        exit 1
    fi
}

# Check system requirements
check_requirements() {
    log_info "Checking system requirements..."

    # Check if wget is available
    if ! command -v wget &> /dev/null; then
        log_error "wget is required but not installed"
        log_info "Install with: apt install wget (Ubuntu/Debian) or dnf install wget (Fedora/RHEL)"
        exit 1
    fi

    # Check if tar is available
    if ! command -v tar &> /dev/null; then
        log_error "tar is required but not installed"
        exit 1
    fi

    # Check available disk space (at least 1GB)
    available_space=$(df / | awk 'NR==2 {print $4}')
    if [[ $available_space -lt 1048576 ]]; then
        log_warning "Low disk space detected. At least 1GB free space recommended"
    fi

    log_success "System requirements check passed"
}

# Create assets directory
create_assets_dir() {
    log_info "Creating assets directory: $ASSETS_DIR"

    if [[ -d "$ASSETS_DIR" ]]; then
        log_warning "Directory $ASSETS_DIR already exists"
        read -p "Do you want to continue and overwrite existing files? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Setup cancelled by user"
            exit 0
        fi
    fi

    mkdir -p "$ASSETS_DIR"
    log_success "Assets directory created"
}

# Download kernel
download_kernel() {
    log_info "Downloading Firecracker kernel..."

    local kernel_path="$ASSETS_DIR/vmlinux"
    local temp_file=$(mktemp)

    if wget -q --show-progress "$KERNEL_URL" -O "$temp_file"; then
        mv "$temp_file" "$kernel_path"
        chmod 644 "$kernel_path"
        log_success "Kernel downloaded: $kernel_path"
    else
        log_error "Failed to download kernel from $KERNEL_URL"
        rm -f "$temp_file"
        exit 1
    fi
}

# Download rootfs
download_rootfs() {
    log_info "Downloading Ubuntu Bionic rootfs..."

    local rootfs_path="$ASSETS_DIR/rootfs.ext4"
    local temp_file=$(mktemp)

    if wget -q --show-progress "$ROOTFS_URL" -O "$temp_file"; then
        mv "$temp_file" "$rootfs_path"
        chmod 644 "$rootfs_path"
        log_success "Rootfs downloaded: $rootfs_path"
    else
        log_error "Failed to download rootfs from $ROOTFS_URL"
        rm -f "$temp_file"
        exit 1
    fi
}

# Download and install Firecracker
download_firecracker() {
    log_info "Downloading Firecracker binary..."

    local temp_dir=$(mktemp -d)
    local firecracker_archive="$temp_dir/firecracker.tgz"

    # Download Firecracker
    if wget -q --show-progress "$FIRECRACKER_URL" -O "$firecracker_archive"; then
        log_info "Extracting Firecracker binary..."

        # Extract and install
        tar -xzf "$firecracker_archive" -C "$temp_dir"

        # Find the binary (handle different archive structures)
        local binary_path
        if [[ -f "$temp_dir/release-${FIRECRACKER_VERSION}-x86_64/firecracker-${FIRECRACKER_VERSION}-x86_64" ]]; then
            binary_path="$temp_dir/release-${FIRECRACKER_VERSION}-x86_64/firecracker-${FIRECRACKER_VERSION}-x86_64"
        elif [[ -f "$temp_dir/firecracker" ]]; then
            binary_path="$temp_dir/firecracker"
        else
            log_error "Could not find Firecracker binary in archive"
            rm -rf "$temp_dir"
            exit 1
        fi

        # Install binary
        cp "$binary_path" /usr/local/bin/firecracker
        chmod 755 /usr/local/bin/firecracker

        # Cleanup
        rm -rf "$temp_dir"

        log_success "Firecracker binary installed: /usr/local/bin/firecracker"
    else
        log_error "Failed to download Firecracker from $FIRECRACKER_URL"
        rm -rf "$temp_dir"
        exit 1
    fi
}

# Verify installation
verify_installation() {
    log_info "Verifying installation..."

    # Check kernel file
    if [[ -f "$ASSETS_DIR/vmlinux" ]]; then
        local kernel_size=$(stat -c%s "$ASSETS_DIR/vmlinux")
        log_success "Kernel file verified (size: $((kernel_size / 1024 / 1024))MB)"
    else
        log_error "Kernel file not found"
        exit 1
    fi

    # Check rootfs file
    if [[ -f "$ASSETS_DIR/rootfs.ext4" ]]; then
        local rootfs_size=$(stat -c%s "$ASSETS_DIR/rootfs.ext4")
        log_success "Rootfs file verified (size: $((rootfs_size / 1024 / 1024))MB)"
    else
        log_error "Rootfs file not found"
        exit 1
    fi

    # Check Firecracker binary
    if command -v firecracker &> /dev/null; then
        local version=$(firecracker --version 2>&1 | head -1)
        log_success "Firecracker binary verified: $version"
    else
        log_error "Firecracker binary not found or not executable"
        exit 1
    fi

    # Display final status
    echo
    log_success "VM assets setup completed successfully!"
    echo
    echo "Assets installed:"
    echo "  Kernel:      $ASSETS_DIR/vmlinux"
    echo "  Rootfs:      $ASSETS_DIR/rootfs.ext4"
    echo "  Firecracker: /usr/local/bin/firecracker"
    echo
    log_info "Next steps:"
    echo "  1. Start Firecracker: sudo firecracker --api-sock /tmp/firecracker.sock"
    echo "  2. Configure environment: export UNKEY_METALD_BACKEND=firecracker"
    echo "  3. Set endpoint: export UNKEY_METALD_FC_ENDPOINT=unix:///tmp/firecracker.sock"
    echo "  4. Start VMM control plane: ./api"
    echo
}

# Cleanup function for error handling
cleanup() {
    if [[ -n "${temp_file:-}" ]] && [[ -f "$temp_file" ]]; then
        rm -f "$temp_file"
    fi
    if [[ -n "${temp_dir:-}" ]] && [[ -d "$temp_dir" ]]; then
        rm -rf "$temp_dir"
    fi
}

# Set trap for cleanup
trap cleanup EXIT

# Main execution
main() {
    echo "========================================"
    echo "  Firecracker VM Assets Setup Script"
    echo "========================================"
    echo

    check_root
    check_requirements
    create_assets_dir
    download_kernel
    download_rootfs
    download_firecracker
    verify_installation
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Firecracker VM Assets Setup Script"
        echo
        echo "This script downloads and installs VM assets required for Firecracker:"
        echo "  - Optimized Linux kernel (vmlinux)"
        echo "  - Alpine Linux root filesystem (rootfs.ext4)"
        echo "  - Firecracker hypervisor binary"
        echo
        echo "Usage: sudo $0 [OPTIONS]"
        echo
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --version, -v  Show version information"
        echo
        echo "Assets will be installed to: $ASSETS_DIR"
        echo "Firecracker binary will be installed to: /usr/local/bin/firecracker"
        echo
        exit 0
        ;;
    --version|-v)
        echo "Firecracker VM Assets Setup Script v1.0"
        echo "Firecracker version: $FIRECRACKER_VERSION"
        exit 0
        ;;
    "")
        # No arguments, run main
        main
        ;;
    *)
        log_error "Unknown option: $1"
        log_info "Use --help for usage information"
        exit 1
        ;;
esac
