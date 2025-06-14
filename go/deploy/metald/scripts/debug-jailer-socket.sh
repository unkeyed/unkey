#!/bin/bash
#
# Debug script for Firecracker jailer socket communication issues
#

set -e

echo "🔍 Debugging Firecracker Jailer Socket Communication"
echo "==================================================="
echo ""

# Check if running as root or with sudo
if [ "$EUID" -ne 0 ]; then 
    echo "⚠️  This script should be run with sudo for full debugging capabilities"
fi

# Configuration
JAILER_BASE_DIR=${UNKEY_METALD_JAILER_CHROOT_DIR:-/srv/jailer}
JAILER_UID=${UNKEY_METALD_JAILER_UID:-1000}
JAILER_GID=${UNKEY_METALD_JAILER_GID:-1000}

echo "📋 Configuration:"
echo "  Jailer base directory: $JAILER_BASE_DIR"
echo "  Jailer UID: $JAILER_UID"
echo "  Jailer GID: $JAILER_GID"
echo ""

# Check jailer processes
echo "🔍 Active jailer processes:"
ps aux | grep -E "(jailer|firecracker)" | grep -v grep || echo "  No jailer/firecracker processes found"
echo ""

# Check chroot directories
echo "📁 Jailer chroot directories:"
if [ -d "$JAILER_BASE_DIR" ]; then
    ls -la "$JAILER_BASE_DIR/" 2>/dev/null || echo "  No chroot directories found"
    
    # Check each VM's socket
    for vm_dir in "$JAILER_BASE_DIR"/vm-*; do
        if [ -d "$vm_dir" ]; then
            echo ""
            echo "  VM Directory: $vm_dir"
            SOCKET_PATH="$vm_dir/root/tmp/firecracker.socket"
            
            # Check if socket exists
            if [ -e "$SOCKET_PATH" ]; then
                echo "  ✅ Socket exists: $SOCKET_PATH"
                ls -la "$SOCKET_PATH"
                
                # Check socket connectivity
                echo "  Testing socket connectivity..."
                if timeout 2 curl -s --unix-socket "$SOCKET_PATH" http://localhost/ >/dev/null 2>&1; then
                    echo "  ✅ Socket is responsive"
                else
                    echo "  ❌ Socket is not responsive (curl test failed)"
                    
                    # Try with nc (netcat) as alternative
                    if command -v nc >/dev/null 2>&1; then
                        echo "  Trying netcat test..."
                        if echo -e "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n" | timeout 2 nc -U "$SOCKET_PATH" >/dev/null 2>&1; then
                            echo "  ✅ Socket is responsive (netcat)"
                        else
                            echo "  ❌ Socket is not responsive (netcat)"
                        fi
                    fi
                fi
                
                # Check permissions
                echo "  Socket permissions:"
                stat -c "    Owner: %U:%G (UID:%u GID:%g)" "$SOCKET_PATH"
                stat -c "    Mode: %a" "$SOCKET_PATH"
                
            else
                echo "  ❌ Socket does not exist: $SOCKET_PATH"
                
                # Check if directory exists
                SOCKET_DIR="$vm_dir/root/tmp"
                if [ -d "$SOCKET_DIR" ]; then
                    echo "  Socket directory exists: $SOCKET_DIR"
                    ls -la "$SOCKET_DIR"
                else
                    echo "  ❌ Socket directory missing: $SOCKET_DIR"
                fi
            fi
            
            # Check jailer log
            LOG_FILE="/opt/metald/logs/fc-*.log"
            if ls $LOG_FILE >/dev/null 2>&1; then
                echo "  Recent log entries:"
                tail -n 5 $LOG_FILE | sed 's/^/    /'
            fi
        fi
    done
else
    echo "  ❌ Jailer base directory does not exist: $JAILER_BASE_DIR"
fi

echo ""
echo "🔍 System checks:"

# Check if metald user can access sockets
if id metald >/dev/null 2>&1; then
    echo "  metald user exists:"
    id metald
    
    # Check if metald can access jailer directory
    if sudo -u metald test -r "$JAILER_BASE_DIR" 2>/dev/null; then
        echo "  ✅ metald can read jailer base directory"
    else
        echo "  ❌ metald cannot read jailer base directory"
    fi
else
    echo "  ❌ metald user does not exist"
fi

# Check SELinux/AppArmor
if command -v getenforce >/dev/null 2>&1; then
    SELINUX_STATUS=$(getenforce)
    echo "  SELinux status: $SELINUX_STATUS"
    if [ "$SELINUX_STATUS" = "Enforcing" ]; then
        echo "  ⚠️  SELinux is enforcing - may block socket access"
    fi
fi

if command -v aa-status >/dev/null 2>&1; then
    echo "  AppArmor is installed"
    if aa-status --enabled 2>/dev/null; then
        echo "  ⚠️  AppArmor is enabled - may block socket access"
    fi
fi

echo ""
echo "💡 Troubleshooting tips:"
echo "  1. Ensure metald is running with appropriate privileges"
echo "  2. Check if jailer binary has correct permissions"
echo "  3. Verify firecracker binary is statically linked"
echo "  4. Check systemd/journalctl logs for errors:"
echo "     sudo journalctl -u metald -n 50"
echo "  5. Try running metald with debug logging:"
echo "     RUST_LOG=debug ./metald"
echo ""
echo "📝 To test a manual jailer command:"
echo "   sudo jailer --id test-vm --exec-file /usr/bin/firecracker \\"
echo "     --uid $JAILER_UID --gid $JAILER_GID \\"
echo "     --chroot-base-dir $JAILER_BASE_DIR \\"
echo "     -- --api-sock /tmp/firecracker.socket"