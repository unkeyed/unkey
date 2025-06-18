# Metald Root Deployment Guide

## Overview

As of version 1.1.0, metald now runs as root to simplify network namespace and interface management. This document explains the changes and deployment requirements.

## Prerequisites

### System Requirements

1. **IP Forwarding**: Must be enabled
   ```bash
   # Check current status
   sysctl net.ipv4.ip_forward
   
   # Enable temporarily
   sudo sysctl -w net.ipv4.ip_forward=1
   
   # Enable permanently (metald will do this automatically)
   echo "net.ipv4.ip_forward = 1" | sudo tee /etc/sysctl.d/99-metald.conf
   sudo sysctl -p /etc/sysctl.d/99-metald.conf
   ```

2. **Required Packages**:
   - iptables
   - iproute2
   - bridge-utils (optional, for debugging)

3. **Firecracker & Jailer**: Properly installed in `/usr/local/bin/`

## Network Architecture

### Host Network Setup
- Bridge: `br-vms` (10.100.0.1/16)
- VM Subnet: 10.100.0.0/16
- NAT/Masquerading for VM internet access

### Per-VM Network
- Network namespace: `fc-<jailer-id>`
- Device naming:
  - Host veth: `vh_<8-char-id>`
  - Namespace veth: `vn_<8-char-id>`
  - TAP device: `tap_<8-char-id>`
- IP assignment: Static from 10.100.0.0/16 pool

### Network Topology
```
Host Network
    |
    br-vms (10.100.0.1/16)
    |
    vh_<id> (veth host side)
    |
=========== Namespace Boundary ===========
    |
    vn_<id> (veth namespace side, VM IP)
    |
    tap_<id> (TAP for Firecracker)
    |
    Firecracker VM
```

The simplified architecture:
- No bridge inside namespace
- IP assigned directly to veth interface
- Proxy ARP enabled on veth for VM connectivity
- TAP device used by Firecracker for VM networking

## Installation

1. **Build and Install**:
   ```bash
   cd metald
   make install
   ```

2. **Start Service**:
   ```bash
   sudo systemctl start metald
   sudo systemctl enable metald
   ```

3. **Verify Installation**:
   ```bash
   # Check service status
   sudo systemctl status metald
   
   # Check network setup
   ip link show br-vms
   sudo iptables -t nat -L POSTROUTING -n -v
   ```

## Security Considerations

Running as root is acceptable for metald because:

1. **Single-Purpose Host**: Metald is designed to be the sole application on dedicated VM hosts
2. **Defense in Depth**: 
   - mTLS via SPIFFE/SPIRE for service communication
   - Firecracker provides strong VM isolation
   - Network namespaces isolate VM traffic
   - Jailer drops privileges for VM processes
3. **Operational Simplicity**: Eliminates complex privilege escalation workarounds

## Troubleshooting

### Network Issues

1. **Bridge not created**:
   ```bash
   # Check if bridge exists
   ip link show br-vms
   
   # Check metald logs
   journalctl -u metald -f
   ```

2. **VMs can't reach internet**:
   ```bash
   # Check IP forwarding
   sysctl net.ipv4.ip_forward
   
   # Check NAT rules
   sudo iptables -t nat -L POSTROUTING -n -v | grep 10.100.0.0
   ```

3. **Namespace issues**:
   ```bash
   # List namespaces
   ip netns list
   
   # Check namespace networking
   sudo ip netns exec fc-<id> ip addr
   ```

### Cleanup

If you need to manually clean up:
```bash
# Stop service
sudo systemctl stop metald

# Remove bridge (if no VMs running)
sudo ip link del br-vms

# Clean up any remaining namespaces
for ns in $(ip netns list | grep '^fc-' | awk '{print $1}'); do
    sudo ip netns del "$ns"
done
```

## Changes from Previous Versions

1. **Service runs as root** instead of dedicated user
2. **No more nsenter workarounds** in network code
3. **Simplified systemd unit** without capability restrictions
4. **IP forwarding persistence** via sysctl.d
5. **Better error handling** with automatic rollback
6. **Fixed cleanup order** (namespace before veth deletion)
7. **Consistent /16 subnet** usage throughout
8. **Simplified namespace networking** without internal bridge
9. **Proxy ARP** enabled for direct veth routing

## Migration from Non-Root Setup

1. Stop the old service
2. Remove old sudoers configuration
3. Install new version with `make install`
4. Start the service

The network configuration remains compatible - existing VMs will continue to work.