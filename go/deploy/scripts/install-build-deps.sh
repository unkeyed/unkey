#!/bin/bash
set -e

# Install build dependencies for package creation
# Supports both RPM-based (RHEL, Fedora, CentOS) and Debian-based (Ubuntu, Debian) systems

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$NAME
        VER=$VERSION_ID
    else
        error "Cannot detect operating system"
    fi
    
    log "Detected OS: $OS $VER"
}

install_rpm_deps() {
    log "Installing RPM build dependencies..."
    
    local packages=(
        "rpm-build"
        "rpmdevtools"
        "golang"
        "systemd-rpm-macros"
        "rsync"
        "tar"
        "gzip"
    )
    
    # Detect package manager
    if command -v dnf &> /dev/null; then
        local pm="dnf"
    elif command -v yum &> /dev/null; then
        local pm="yum"
    else
        error "No supported package manager found (dnf or yum)"
    fi
    
    log "Using package manager: $pm"
    
    # Install packages
    for package in "${packages[@]}"; do
        log "Installing $package..."
        sudo $pm install -y "$package" || warn "Failed to install $package"
    done
    
    # Setup RPM build tree
    log "Setting up RPM build tree..."
    rpmdev-setuptree
    
    log "RPM build dependencies installed successfully"
}

install_deb_deps() {
    log "Installing Debian build dependencies..."
    
    local packages=(
        "build-essential"
        "debhelper"
        "devscripts"
        "golang-go"
        "dh-systemd"
        "rsync"
        "tar"
        "gzip"
    )
    
    # Update package list
    log "Updating package list..."
    sudo apt update
    
    # Install packages
    for package in "${packages[@]}"; do
        log "Installing $package..."
        sudo apt install -y "$package" || warn "Failed to install $package"
    done
    
    log "Debian build dependencies installed successfully"
}

install_common_deps() {
    log "Installing common build dependencies..."
    
    # Install Go if not already present with correct version
    if ! command -v go &> /dev/null; then
        warn "Go not found in PATH after package installation"
        return 1
    fi
    
    local go_version=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
    local required_version="1.21"
    
    if [[ "$(printf '%s\n' "$required_version" "$go_version" | sort -V | head -n1)" != "$required_version" ]]; then
        warn "Go version $go_version is less than required $required_version"
        return 1
    fi
    
    log "Go version $go_version is compatible"
    
    # Install additional Go tools that might be needed
    log "Installing additional Go tools..."
    go install golang.org/x/tools/cmd/goimports@latest || warn "Failed to install goimports"
    
    log "Common dependencies installed successfully"
}

main() {
    log "Installing build dependencies..."
    
    # Check if running as root
    if [[ $EUID -eq 0 ]]; then
        error "This script should not be run as root. Use sudo when needed."
    fi
    
    # Detect OS
    detect_os
    
    # Install dependencies based on OS
    case "$OS" in
        "Fedora"*|"Red Hat"*|"CentOS"*|"Rocky"*|"AlmaLinux"*)
            install_rpm_deps
            ;;
        "Ubuntu"*|"Debian"*)
            install_deb_deps
            ;;
        *)
            error "Unsupported operating system: $OS"
            ;;
    esac
    
    # Install common dependencies
    install_common_deps
    
    log "All build dependencies installed successfully!"
    log "You can now run: ./scripts/build-packages.sh --help"
}

# Run main function
main "$@"