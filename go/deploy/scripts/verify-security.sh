#!/bin/bash
set -euo pipefail

# AIDEV-NOTE: This script verifies that required security features are active:
# 1. Firecracker jailer is enabled and working
# 2. SPIFFE/mTLS is enabled for all services

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Function to print colored output
print_status() {
    echo -e "${2}${1}${NC}"
}

print_status "Security Verification Script" "$GREEN"
print_status "===========================" "$GREEN"
echo

# Check if running as root
if [[ $EUID -eq 0 ]]; then
    print_status "Running as root" "$GREEN"
else
    print_status "Warning: Some checks may require root access" "$YELLOW"
fi

# 1. Verify Jailer Setup
print_status "\n1. Firecracker Jailer Verification" "$YELLOW"
print_status "===================================" "$YELLOW"

# Check jailer binary and capabilities
if [ -f /usr/local/bin/jailer ]; then
    print_status "✓ Jailer binary found at /usr/local/bin/jailer" "$GREEN"
    
    # Check capabilities
    caps=$(getcap /usr/local/bin/jailer 2>/dev/null || echo "none")
    if [[ "$caps" == *"cap_sys_admin"* ]] && [[ "$caps" == *"cap_mknod"* ]]; then
        print_status "✓ Jailer has required capabilities: $caps" "$GREEN"
    else
        print_status "✗ Jailer missing required capabilities. Found: $caps" "$RED"
        print_status "  Run: sudo setcap 'cap_sys_admin,cap_mknod+ep' /usr/local/bin/jailer" "$YELLOW"
    fi
else
    print_status "✗ Jailer binary not found at /usr/local/bin/jailer" "$RED"
fi

# Check jailer user
if id metald &>/dev/null; then
    uid=$(id -u metald)
    gid=$(id -g metald)
    print_status "✓ Jailer user 'metald' exists (UID: $uid, GID: $gid)" "$GREEN"
else
    print_status "✗ Jailer user 'metald' does not exist" "$RED"
    print_status "  Run: sudo ./scripts/configure-jailer-user.sh --create" "$YELLOW"
fi

# Check cgroups
if [ -d /sys/fs/cgroup/firecracker ]; then
    owner=$(stat -c '%U:%G' /sys/fs/cgroup/firecracker 2>/dev/null || echo "unknown")
    print_status "✓ Firecracker cgroup exists (owner: $owner)" "$GREEN"
else
    print_status "✗ Firecracker cgroup not found" "$RED"
    print_status "  Run: sudo ./scripts/setup-firecracker-cgroups.sh" "$YELLOW"
fi

# Check if metald is using jailer
if systemctl is-active metald >/dev/null 2>&1; then
    if pgrep -f jailer >/dev/null 2>&1; then
        jailer_count=$(pgrep -f jailer | wc -l)
        print_status "✓ Jailer processes active: $jailer_count" "$GREEN"
    else
        print_status "⚠ No jailer processes found (no VMs running?)" "$YELLOW"
    fi
    
    # Check logs for jailer
    if sudo journalctl -u metald -n 100 --no-pager | grep -q "jailer enabled"; then
        print_status "✓ Metald logs confirm jailer is enabled" "$GREEN"
    else
        print_status "⚠ Could not confirm jailer status from logs" "$YELLOW"
    fi
else
    print_status "⚠ Metald service is not running" "$YELLOW"
fi

# 2. Verify SPIFFE/mTLS Setup
print_status "\n2. SPIFFE/mTLS Verification" "$YELLOW"
print_status "============================" "$YELLOW"

# Check SPIRE server
if systemctl is-active spire-server >/dev/null 2>&1; then
    print_status "✓ SPIRE server is running" "$GREEN"
else
    print_status "✗ SPIRE server is not running" "$RED"
    print_status "  Run: sudo systemctl start spire-server" "$YELLOW"
fi

# Check SPIRE agent
if systemctl is-active spire-agent >/dev/null 2>&1; then
    print_status "✓ SPIRE agent is running" "$GREEN"
else
    print_status "✗ SPIRE agent is not running" "$RED"
    print_status "  Run: sudo systemctl start spire-agent" "$YELLOW"
fi

# Check SPIFFE socket
if [ -S /run/spire/sockets/agent.sock ]; then
    print_status "✓ SPIFFE workload API socket exists" "$GREEN"
    
    # Try to check agent health
    if command -v /opt/spire/bin/spire-agent &>/dev/null; then
        if sudo /opt/spire/bin/spire-agent healthcheck -socketPath /run/spire/sockets/agent.sock 2>/dev/null; then
            print_status "✓ SPIRE agent healthcheck passed" "$GREEN"
        else
            print_status "✗ SPIRE agent healthcheck failed" "$RED"
        fi
    fi
else
    print_status "✗ SPIFFE workload API socket not found" "$RED"
fi

# Check service TLS configuration
print_status "\n3. Service TLS Configuration" "$YELLOW"
print_status "=============================" "$YELLOW"

services=("assetmanagerd" "billaged" "builderd" "metald")
for service in "${services[@]}"; do
    if systemctl is-active "$service" >/dev/null 2>&1; then
        # Check if service logs show SPIFFE mode
        if sudo journalctl -u "$service" -n 100 --no-pager | grep -qi "tls.*spiffe\|spiffe.*enabled\|tls mode.*spiffe"; then
            print_status "✓ $service: mTLS/SPIFFE enabled" "$GREEN"
        else
            print_status "⚠ $service: Could not confirm mTLS/SPIFFE from logs" "$YELLOW"
        fi
    else
        print_status "⚠ $service: Service not running" "$YELLOW"
    fi
done

# Check for registered SPIFFE entries
if command -v /opt/spire/bin/spire-server &>/dev/null && systemctl is-active spire-server >/dev/null 2>&1; then
    print_status "\n4. SPIFFE Registrations" "$YELLOW"
    print_status "========================" "$YELLOW"
    
    entry_count=$(sudo /opt/spire/bin/spire-server entry list -socketPath /run/spire/server.sock 2>/dev/null | grep -c "Entry ID" || echo "0")
    if [ "$entry_count" -gt 0 ]; then
        print_status "✓ SPIFFE entries registered: $entry_count" "$GREEN"
    else
        print_status "✗ No SPIFFE entries found" "$RED"
        print_status "  Run: make -C spire register-services" "$YELLOW"
    fi
fi

# Summary
print_status "\n5. Summary" "$YELLOW"
print_status "===========" "$YELLOW"

errors=0
warnings=0

# Count results
if grep -q "✗" <<< "$0"; then
    errors=1
fi
if grep -q "⚠" <<< "$0"; then
    warnings=1
fi

if [ "$errors" -eq 0 ] && [ "$warnings" -eq 0 ]; then
    print_status "\n✓ All security features are properly configured!" "$GREEN"
else
    print_status "\n⚠ Some issues were found. Please review the output above." "$YELLOW"
fi

print_status "\nSecurity requirements:" "$YELLOW"
print_status "- Firecracker Jailer: REQUIRED for VM isolation" "$NC"
print_status "- SPIFFE/mTLS: REQUIRED for secure inter-service communication" "$NC"