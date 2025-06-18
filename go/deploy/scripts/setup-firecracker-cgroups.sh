#!/bin/bash
set -euo pipefail

# AIDEV-NOTE: This script sets up cgroups for Firecracker jailer according to requirements:
# (a) Jailer user must have access to firecracker cgroups or permissions to create them
# (b) Jailer binary must have cap_sys_admin,cap_mknod capabilities
# (c) User/group id for firecracker must match the jailer user

# Default values
JAILER_USER="${JAILER_USER:-metald}"
JAILER_GROUP="${JAILER_GROUP:-metald}"
JAILER_BINARY="${JAILER_BINARY:-/usr/local/bin/jailer}"
CGROUP_VERSION="${CGROUP_VERSION:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Function to print colored output
print_status() {
    echo -e "${2}${1}${NC}"
}

# Function to check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_status "This script must be run as root" "$RED"
        exit 1
    fi
}

# Function to detect cgroup version
detect_cgroup_version() {
    if [[ -n "$CGROUP_VERSION" ]]; then
        echo "Using specified cgroup version: v$CGROUP_VERSION"
        return
    fi

    # Check if cgroup2 is mounted at /sys/fs/cgroup
    if mount | grep -q "^cgroup2 on /sys/fs/cgroup type cgroup2"; then
        CGROUP_VERSION=2
        print_status "Detected cgroup v2 (unified hierarchy)" "$GREEN"
    elif [ -f /sys/fs/cgroup/cgroup.controllers ]; then
        # Another way to detect cgroup v2
        CGROUP_VERSION=2
        print_status "Detected cgroup v2" "$GREEN"
    elif [ -d /sys/fs/cgroup/memory ] && [ -d /sys/fs/cgroup/cpu ] && [ ! -f /sys/fs/cgroup/cgroup.controllers ]; then
        # Traditional cgroup v1 setup
        CGROUP_VERSION=1
        print_status "Detected cgroup v1" "$GREEN"
    else
        print_status "Could not detect cgroup version" "$RED"
        print_status "You can force a version with: CGROUP_VERSION=2 $0" "$YELLOW"
        exit 1
    fi
    
    # Recommend cgroup v2 if v1 is detected
    if [ "$CGROUP_VERSION" = "1" ]; then
        print_status "Note: cgroup v1 detected. Consider upgrading to cgroup v2 for better performance and features." "$YELLOW"
        print_status "You can force cgroup v2 setup with: CGROUP_VERSION=2 $0" "$YELLOW"
    fi
}

# Function to setup cgroup v1
setup_cgroup_v1() {
    print_status "Setting up cgroup v1 for Firecracker..." "$YELLOW"
    
    # Create firecracker cgroups
    local cgroups=("cpu" "cpuset" "pids" "memory" "blkio" "net_cls" "net_prio" "devices")
    
    for cgroup in "${cgroups[@]}"; do
        local cgroup_path="/sys/fs/cgroup/$cgroup/firecracker"
        
        if [ ! -d "$cgroup_path" ]; then
            print_status "Creating $cgroup_path" "$YELLOW"
            mkdir -p "$cgroup_path"
        fi
        
        # For cpuset, we need to initialize cpus and mems BEFORE changing ownership
        if [ "$cgroup" = "cpuset" ]; then
            # Copy parent values as root
            if [ -f "/sys/fs/cgroup/cpuset/cpuset.cpus" ]; then
                cat "/sys/fs/cgroup/cpuset/cpuset.cpus" > "$cgroup_path/cpuset.cpus"
            fi
            if [ -f "/sys/fs/cgroup/cpuset/cpuset.mems" ]; then
                cat "/sys/fs/cgroup/cpuset/cpuset.mems" > "$cgroup_path/cpuset.mems"
            fi
        fi
        
        # Now set ownership to jailer user
        chown -R "${JAILER_USER}:${JAILER_GROUP}" "$cgroup_path"
    done
    
    print_status "Cgroup v1 setup complete" "$GREEN"
}

# Function to setup cgroup v2
setup_cgroup_v2() {
    print_status "Setting up cgroup v2 for Firecracker..." "$YELLOW"
    
    # Enable cgroup delegation for systemd
    local systemd_conf="/etc/systemd/system/metald.service.d/cgroup.conf"
    mkdir -p "$(dirname "$systemd_conf")"
    
    cat > "$systemd_conf" << EOF
[Service]
# Enable cgroup delegation for Firecracker
Delegate=cpu cpuset io memory pids
EOF
    
    # Create firecracker cgroup
    local cgroup_path="/sys/fs/cgroup/firecracker"
    
    if [ ! -d "$cgroup_path" ]; then
        print_status "Creating $cgroup_path" "$YELLOW"
        mkdir -p "$cgroup_path"
    fi
    
    # Enable controllers
    echo "+cpu +cpuset +io +memory +pids" > "/sys/fs/cgroup/cgroup.subtree_control" 2>/dev/null || true
    
    # Set ownership
    chown -R "${JAILER_USER}:${JAILER_GROUP}" "$cgroup_path"
    
    # Enable controllers in firecracker cgroup
    echo "+cpu +cpuset +io +memory +pids" > "$cgroup_path/cgroup.subtree_control" 2>/dev/null || true
    
    print_status "Cgroup v2 setup complete" "$GREEN"
    print_status "Note: You may need to reload systemd and restart metald service" "$YELLOW"
}

# Function to set capabilities on jailer binary
setup_jailer_capabilities() {
    print_status "Setting up jailer binary capabilities..." "$YELLOW"
    
    if [ ! -f "$JAILER_BINARY" ]; then
        print_status "Jailer binary not found at $JAILER_BINARY" "$RED"
        exit 1
    fi
    
    # Set required capabilities
    # AIDEV-NOTE: cap_sys_admin is needed for namespace operations
    # cap_mknod is needed for device node creation in chroot
    setcap 'cap_sys_admin,cap_mknod+ep' "$JAILER_BINARY"
    
    # Verify capabilities were set
    if getcap "$JAILER_BINARY" | grep -q "cap_sys_admin,cap_mknod"; then
        print_status "Capabilities set successfully" "$GREEN"
    else
        print_status "Failed to set capabilities" "$RED"
        exit 1
    fi
}

# Function to verify user exists
verify_user() {
    if ! id "$JAILER_USER" &>/dev/null; then
        print_status "User $JAILER_USER does not exist" "$RED"
        print_status "Please create the user first with: useradd -r -s /bin/false $JAILER_USER" "$YELLOW"
        exit 1
    fi
    
    print_status "User $JAILER_USER exists" "$GREEN"
}

# Function to setup permissions for jailer chroot directory
setup_jailer_chroot_permissions() {
    print_status "Setting up jailer chroot directory permissions..." "$YELLOW"
    
    local jailer_chroot_base="/srv/jailer"
    
    if [ ! -d "$jailer_chroot_base" ]; then
        mkdir -p "$jailer_chroot_base"
    fi
    
    # AIDEV-BUSINESS_RULE: Jailer needs to create VM-specific directories under this path
    chown "${JAILER_USER}:${JAILER_GROUP}" "$jailer_chroot_base"
    chmod 755 "$jailer_chroot_base"
    
    print_status "Jailer chroot directory permissions set" "$GREEN"
}

# Function to create a script for runtime cgroup setup
create_runtime_script() {
    print_status "Creating runtime cgroup setup script..." "$YELLOW"
    
    local script_path="/usr/local/bin/firecracker-cgroup-setup"
    
    cat > "$script_path" << 'EOF'
#!/bin/bash
# Runtime script to ensure cgroups are properly set up before starting Firecracker VMs
# This can be called by metald or systemd before launching VMs

set -euo pipefail

JAILER_USER="${JAILER_USER:-metald}"
JAILER_GROUP="${JAILER_GROUP:-metald}"

# Function to ensure cgroup v1 is ready
ensure_cgroup_v1() {
    local cgroups=("cpu" "cpuset" "pids" "memory")
    
    for cgroup in "${cgroups[@]}"; do
        local cgroup_path="/sys/fs/cgroup/$cgroup/firecracker"
        
        if [ ! -d "$cgroup_path" ]; then
            mkdir -p "$cgroup_path"
            chown -R "${JAILER_USER}:${JAILER_GROUP}" "$cgroup_path"
            
            if [ "$cgroup" = "cpuset" ]; then
                cat "/sys/fs/cgroup/cpuset/cpuset.cpus" > "$cgroup_path/cpuset.cpus" 2>/dev/null || true
                cat "/sys/fs/cgroup/cpuset/cpuset.mems" > "$cgroup_path/cpuset.mems" 2>/dev/null || true
            fi
        fi
    done
}

# Function to ensure cgroup v2 is ready
ensure_cgroup_v2() {
    local cgroup_path="/sys/fs/cgroup/firecracker"
    
    if [ ! -d "$cgroup_path" ]; then
        mkdir -p "$cgroup_path"
        chown -R "${JAILER_USER}:${JAILER_GROUP}" "$cgroup_path"
        echo "+cpu +cpuset +io +memory +pids" > "$cgroup_path/cgroup.subtree_control" 2>/dev/null || true
    fi
}

# Detect and ensure cgroups
if grep -q cgroup2 /proc/filesystems && [ -d /sys/fs/cgroup/unified ]; then
    ensure_cgroup_v2
elif [ -d /sys/fs/cgroup/cpu ]; then
    ensure_cgroup_v1
fi

exit 0
EOF
    
    chmod +x "$script_path"
    print_status "Runtime script created at $script_path" "$GREEN"
}

# Main execution
main() {
    print_status "Firecracker Cgroups Setup Script" "$GREEN"
    print_status "================================" "$GREEN"
    
    check_root
    verify_user
    detect_cgroup_version
    
    if [ "$CGROUP_VERSION" = "1" ]; then
        setup_cgroup_v1
    elif [ "$CGROUP_VERSION" = "2" ]; then
        setup_cgroup_v2
    else
        print_status "Unsupported cgroup version" "$RED"
        exit 1
    fi
    
    setup_jailer_capabilities
    setup_jailer_chroot_permissions
    create_runtime_script
    
    print_status "\nSetup complete!" "$GREEN"
    print_status "Next steps:" "$YELLOW"
    print_status "1. If using cgroup v2, reload systemd: systemctl daemon-reload" "$YELLOW"
    print_status "2. Restart metald service: systemctl restart metald" "$YELLOW"
    print_status "3. The runtime script at /usr/local/bin/firecracker-cgroup-setup can be used to ensure cgroups are ready" "$YELLOW"
    
    # AIDEV-NOTE: Additional considerations for production:
    # - Consider using systemd's Delegate= directive for better cgroup management
    # - Monitor cgroup usage and adjust limits as needed
    # - Implement proper cleanup of stale cgroups from terminated VMs
}

# Run main function
main "$@"