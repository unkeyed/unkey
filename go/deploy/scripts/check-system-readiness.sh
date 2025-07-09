#!/bin/bash
# AIDEV-BUSINESS_RULE: System readiness check for deploying Unkey services on Fedora 42 or Ubuntu
# This script checks for all prerequisites before service installation

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Tracking variables
ERRORS=0
WARNINGS=0

# Detect OS
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$NAME
        VER=$VERSION_ID
    else
        echo -e "${RED}Cannot detect OS. /etc/os-release not found.${NC}"
        exit 1
    fi
}

# Print check result
check_result() {
    local check_name=$1
    local result=$2
    local message=$3

    if [ "$result" -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $check_name: $message"
    else
        echo -e "${RED}✗${NC} $check_name: $message"
        ((ERRORS++))
    fi
}

# Print warning
check_warning() {
    local check_name=$1
    local message=$2
    echo -e "${YELLOW}⚠${NC} $check_name: $message"
    ((WARNINGS++))
}

# Check if running as root or with sudo
check_sudo() {
    if [ "$EUID" -ne 0 ] && ! sudo -n true 2>/dev/null; then
        check_result "Sudo Access" 1 "Script must be run as root or with sudo privileges"
    else
        check_result "Sudo Access" 0 "Sufficient privileges available"
    fi
}

# Check systemd
check_systemd() {
    if command -v systemctl &> /dev/null && systemctl --version &> /dev/null; then
        check_result "systemd" 0 "systemd is installed"
    else
        check_result "systemd" 1 "systemd is required but not found"
    fi
}

# Check Go version
check_go() {
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        REQUIRED_VERSION="1.24"

        if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" = "$REQUIRED_VERSION" ]; then
            check_result "Go Version" 0 "Go $GO_VERSION installed (requires >= $REQUIRED_VERSION)"
        else
            check_result "Go Version" 1 "Go $GO_VERSION installed but version >= $REQUIRED_VERSION required"
        fi
    else
        check_result "Go Version" 1 "Go is not installed (requires >= 1.24)"
    fi
}

# Check Make
check_make() {
    if command -v make &> /dev/null; then
        check_result "Make" 0 "Make is installed"
    else
        check_result "Make" 1 "Make is required but not found"
    fi
}

# Check Git
check_git() {
    if command -v git &> /dev/null; then
        check_result "Git" 0 "Git is installed"
    else
        check_result "Git" 1 "Git is required but not found"
    fi
}

# Check Docker/Podman (for builderd and observability)
check_container_runtime() {
    local docker_found=false
    local podman_found=false

    if command -v docker &> /dev/null; then
        docker_found=true
        if docker info &> /dev/null; then
            check_result "Docker" 0 "Docker is installed and running"
            # Check for docker compose
            if docker compose version &> /dev/null; then
                check_result "Docker Compose" 0 "Docker Compose plugin is available"
            else
                check_warning "Docker Compose" "Docker Compose plugin not found (required for SPIRE quickstart)"
            fi
        else
            check_warning "Docker" "Docker is installed but not running or accessible"
        fi
    fi

    if command -v podman &> /dev/null; then
        podman_found=true
        if podman info &> /dev/null; then
            check_result "Podman" 0 "Podman is installed and running"
        else
            check_warning "Podman" "Podman is installed but not running or accessible"
        fi
    fi

    if [ "$docker_found" = false ] && [ "$podman_found" = false ]; then
        check_warning "Container Runtime" "Neither Docker nor Podman found (required for builderd service and observability stack)"
    fi
}

# Check Firecracker (for metald)
check_firecracker() {
    local fc_found=false

    if command -v firecracker &> /dev/null; then
        echo "nope!!"
        fc_found=true
        check_result "Firecracker" 0 "Firecracker is installed"
    fi
}

# Check KVM support
check_kvm() {
    if [ -e /dev/kvm ]; then
        if [ -r /dev/kvm ] && [ -w /dev/kvm ]; then
            check_result "KVM" 0 "KVM is available and accessible"
        else
            check_warning "KVM" "KVM exists but may not be accessible to current user (required for metald)"
        fi
    else
        check_warning "KVM" "/dev/kvm not found - virtualization may not be enabled (required for metald)"
    fi
}

# Check required tools for the build process
check_build_tools() {
    local tools=("curl" "wget" "tar" "gzip")
    local missing=()

    for tool in "${tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            missing+=("$tool")
        fi
    done

    if [ ${#missing[@]} -eq 0 ]; then
        check_result "Build Tools" 0 "All build tools are installed"
    else
        check_result "Build Tools" 1 "Missing tools: ${missing[*]}"
    fi
}

# Check buf for protobuf generation
check_buf() {
    if command -v buf &> /dev/null; then
        check_result "Buf" 0 "Buf is installed ($(buf --version))"
    else
        check_result "Buf" 1 "Buf is required for protobuf generation but not found"
        echo "  To install buf:"
        echo "    # Using the install script (recommended):"
        echo "    curl -sSL https://github.com/bufbuild/buf/releases/download/v1.28.1/buf-Linux-x86_64 -o /tmp/buf"
        echo "    sudo install -m 755 /tmp/buf /usr/local/bin/buf"
        echo ""
        echo "    # Or via Go:"
        echo "    go install github.com/bufbuild/buf/cmd/buf@latest"
    fi
}

# Check disk space (at least 5GB free)
check_disk_space() {
    AVAILABLE_SPACE=$(df -BG . | awk 'NR==2 {print $4}' | sed 's/G//')
    if [ "$AVAILABLE_SPACE" -ge 5 ]; then
        check_result "Disk Space" 0 "${AVAILABLE_SPACE}GB available (requires >= 5GB)"
    else
        check_result "Disk Space" 1 "${AVAILABLE_SPACE}GB available (requires >= 5GB)"
    fi
}

# Check network connectivity
check_network() {
    if ping -c 1 -W 2 github.com &> /dev/null; then
        check_result "Network" 0 "Network connectivity confirmed"
    else
        check_warning "Network" "Cannot reach github.com - network issues may prevent dependency downloads"
    fi
}

# Check for conflicting services
check_port_availability() {
    local ports=("8080" "8081" "8082" "8083" "9464" "9465" "9466")
    local conflicts=()

    for port in "${ports[@]}"; do
        if ss -tlnp 2>/dev/null | grep -q ":$port "; then
            conflicts+=("$port")
        fi
    done

    if [ ${#conflicts[@]} -eq 0 ]; then
        check_result "Port Availability" 0 "All required ports are available"
    else
        check_warning "Port Availability" "Ports already in use: ${conflicts[*]}"
    fi
}

# Check cgroup version
check_cgroup_version() {
    if [ -f /sys/fs/cgroup/cgroup.controllers ]; then
        check_result "Cgroup" 0 "cgroup v2 is active"
    else
        check_result "Cgroup" 1 "cgroup v2 is required but not active"
        echo "  To enable cgroup v2:"
        echo "    sudo grubby --update-kernel=ALL --args='systemd.unified_cgroup_hierarchy=1'"
        echo "    Then reboot your system"
    fi
}

# Main execution
main() {
    echo "==================================="
    echo "Unkey Services System Readiness Check"
    echo "==================================="
    echo

    detect_os
    echo "Detected OS: $OS $VER"
    echo

    # Verify supported OS
    case "$OS" in
        "Fedora Linux")
            if [ "$VER" -lt 40 ]; then
                check_warning "OS Version" "Fedora $VER detected. Fedora 42 or later recommended"
            else
                check_result "OS Version" 0 "Fedora $VER is supported"
            fi
            ;;
        "Ubuntu")
            if [ "${VER%%.*}" -lt 22 ]; then
                check_warning "OS Version" "Ubuntu $VER detected. Ubuntu 22.04 or later recommended"
            else
                check_result "OS Version" 0 "Ubuntu $VER is supported"
            fi
            ;;
        *)
            check_warning "OS Version" "$OS is not officially tested. Fedora 42 or Ubuntu 22.04+ recommended"
            ;;
    esac

    echo
    echo "Checking system requirements..."
    echo "--------------------------------"

    # Core requirements
    check_sudo
    check_systemd
    check_go
    check_make
    check_git
    check_buf
    check_build_tools
    check_disk_space
    check_network

    echo
    echo "Checking service-specific requirements..."
    echo "-----------------------------------------"

    # Service-specific requirements
    check_container_runtime
    check_firecracker
    check_kvm
    check_cgroup_version
    check_port_availability

    echo
    echo "==================================="
    echo "Summary:"
    echo "-----------------------------------"

    if [ $ERRORS -eq 0 ]; then
        if [ $WARNINGS -eq 0 ]; then
            echo -e "${GREEN}✓ System is ready for deployment!${NC}"
            echo "All requirements are met."
        else
            echo -e "${GREEN}✓ System meets minimum requirements.${NC}"
            echo -e "${YELLOW}  $WARNINGS warning(s) found - some services may have limited functionality.${NC}"
        fi
        echo
        echo "You can proceed with the installation."
        exit 0
    else
        echo -e "${RED}✗ System is not ready for deployment.${NC}"
        echo "  $ERRORS error(s) found that must be resolved."
        [ $WARNINGS -gt 0 ] && echo "  $WARNINGS warning(s) found."
        echo
        echo "Please resolve the errors before proceeding."
        exit 1
    fi
}

# Run main function
main "$@"
