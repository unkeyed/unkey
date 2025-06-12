#!/bin/bash
#
# Jailer + Firecracker Readiness Check
# This script verifies that the system is ready for jailed Firecracker VMs
#

set -e

echo "🔍 Checking Jailer + Firecracker System Readiness"
echo "================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check results
CHECKS=()
FAILURES=()

check_binary() {
    local binary=$1
    local expected_path=$2
    
    if which "$binary" >/dev/null 2>&1; then
        local actual_path=$(which "$binary")
        if [ "$actual_path" = "$expected_path" ]; then
            echo -e "✅ $binary found at expected path: $expected_path"
            CHECKS+=("$binary:PASS")
        else
            echo -e "⚠️  $binary found at $actual_path, expected at $expected_path"
            CHECKS+=("$binary:WARN")
        fi
    else
        echo -e "❌ $binary not found in PATH"
        CHECKS+=("$binary:FAIL")
        FAILURES+=("$binary binary missing")
    fi
}

check_file_exists() {
    local file=$1
    local description=$2
    
    if [ -f "$file" ]; then
        echo -e "✅ $description: $file"
        CHECKS+=("$description:PASS")
    else
        echo -e "❌ $description missing: $file"
        CHECKS+=("$description:FAIL")
        FAILURES+=("$description missing")
    fi
}

check_directory() {
    local dir=$1
    local description=$2
    local owner=$3
    
    if [ -d "$dir" ]; then
        local actual_owner=$(stat -c '%U' "$dir")
        if [ "$actual_owner" = "$owner" ]; then
            echo -e "✅ $description: $dir (owner: $owner)"
            CHECKS+=("$description:PASS")
        else
            echo -e "⚠️  $description: $dir (owner: $actual_owner, expected: $owner)"
            CHECKS+=("$description:WARN")
        fi
    else
        echo -e "❌ $description missing: $dir"
        CHECKS+=("$description:FAIL")
        FAILURES+=("$description missing")
    fi
}

check_user() {
    local user=$1
    
    if id "$user" >/dev/null 2>&1; then
        local uid=$(id -u "$user")
        local gid=$(id -g "$user")
        echo -e "✅ User $user exists (uid:$uid, gid:$gid)"
        CHECKS+=("user-$user:PASS")
    else
        echo -e "❌ User $user does not exist"
        CHECKS+=("user-$user:FAIL")
        FAILURES+=("User $user missing")
    fi
}

check_cgroup_version() {
    if mount | grep -q "cgroup2.*type cgroup2"; then
        echo -e "✅ cgroup v2 detected"
        CHECKS+=("cgroup-v2:PASS")
    elif mount | grep -q "cgroup.*type cgroup"; then
        echo -e "⚠️  cgroup v1 detected (v2 preferred)"
        CHECKS+=("cgroup-v1:WARN")
    else
        echo -e "❌ No cgroup filesystem detected"
        CHECKS+=("cgroup:FAIL")
        FAILURES+=("cgroup filesystem missing")
    fi
}

check_namespace_support() {
    if [ -f "/proc/sys/kernel/ns_last_pid" ]; then
        echo -e "✅ PID namespace support available"
        CHECKS+=("pidns:PASS")
    else
        echo -e "❌ PID namespace support missing"
        CHECKS+=("pidns:FAIL")
        FAILURES+=("PID namespace support missing")
    fi
    
    if [ -d "/var/run/netns" ]; then
        echo -e "✅ Network namespace directory exists"
        CHECKS+=("netns-dir:PASS")
    else
        echo -e "⚠️  Network namespace directory missing (will be created)"
        CHECKS+=("netns-dir:WARN")
    fi
}

check_vm_assets() {
    local assets_dir="/opt/vm-assets"
    
    if [ -d "$assets_dir" ]; then
        echo -e "✅ VM assets directory exists: $assets_dir"
        
        if [ -f "$assets_dir/vmlinux" ]; then
            echo -e "✅ Kernel image found: $assets_dir/vmlinux"
            CHECKS+=("vmlinux:PASS")
        else
            echo -e "❌ Kernel image missing: $assets_dir/vmlinux"
            CHECKS+=("vmlinux:FAIL")
            FAILURES+=("Kernel image missing")
        fi
        
        if [ -f "$assets_dir/rootfs.ext4" ]; then
            echo -e "✅ Root filesystem found: $assets_dir/rootfs.ext4"
            CHECKS+=("rootfs:PASS")
        else
            echo -e "❌ Root filesystem missing: $assets_dir/rootfs.ext4"
            CHECKS+=("rootfs:FAIL")
            FAILURES+=("Root filesystem missing")
        fi
    else
        echo -e "❌ VM assets directory missing: $assets_dir"
        CHECKS+=("vm-assets:FAIL")
        FAILURES+=("VM assets directory missing")
    fi
}

echo ""
echo "🔧 Binary Dependencies"
echo "----------------------"
check_binary "jailer" "/usr/local/bin/jailer"
check_binary "firecracker" "/usr/local/bin/firecracker"

echo ""
echo "👤 User and Permissions"
echo "----------------------"
check_user "metald"

echo ""
echo "📁 Directory Structure"
echo "---------------------"
check_directory "/srv/jailer" "Jailer chroot directory" "metald"
check_directory "/var/run/netns" "Network namespace directory" "root" || mkdir -p /var/run/netns

echo ""
echo "💾 VM Assets"
echo "-----------"
check_vm_assets

echo ""
echo "🔧 System Configuration"
echo "----------------------"
check_cgroup_version
check_namespace_support

echo ""
echo "📊 Summary"
echo "========="

# Count results
PASS_COUNT=$(printf '%s\n' "${CHECKS[@]}" | grep -c ":PASS" || true)
WARN_COUNT=$(printf '%s\n' "${CHECKS[@]}" | grep -c ":WARN" || true)
FAIL_COUNT=$(printf '%s\n' "${CHECKS[@]}" | grep -c ":FAIL" || true)
TOTAL_COUNT=${#CHECKS[@]}

echo "Total checks: $TOTAL_COUNT"
echo -e "✅ Passed: $PASS_COUNT"
echo -e "⚠️  Warnings: $WARN_COUNT"
echo -e "❌ Failed: $FAIL_COUNT"

if [ ${#FAILURES[@]} -eq 0 ]; then
    echo ""
    echo -e "${GREEN}🎉 System is ready for jailer + Firecracker!${NC}"
    echo ""
    echo "To start metald with jailer enabled:"
    echo "sudo systemctl start metald"
    echo ""
    echo "Or run manually with:"
    echo "sudo UNKEY_METALD_JAILER_ENABLED=true ./build/metald"
    exit 0
else
    echo ""
    echo -e "${RED}❌ System is NOT ready. Please fix the following issues:${NC}"
    for failure in "${FAILURES[@]}"; do
        echo -e "${RED}  • $failure${NC}"
    done
    echo ""
    echo "Please resolve these issues and run the check again."
    exit 1
fi