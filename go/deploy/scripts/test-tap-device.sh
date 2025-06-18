#!/bin/bash
# Script to test tap device creation and permissions

set -x

# Create a test namespace
sudo ip netns add test-tap-ns

# Create a tap device in the namespace
sudo ip netns exec test-tap-ns ip tuntap add tap-test mode tap

# Check the tap device
sudo ip netns exec test-tap-ns ip link show tap-test

# Check device permissions
sudo ip netns exec test-tap-ns ls -la /sys/class/net/tap-test/

# Try to open the tap device as non-root
sudo ip netns exec test-tap-ns su -s /bin/bash -c "cat /dev/net/tun" metald 2>&1 || echo "Failed to open as metald user"

# Cleanup
sudo ip netns del test-tap-ns