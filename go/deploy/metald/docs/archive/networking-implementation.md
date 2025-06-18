# Metald Networking Implementation

## Overview

This document summarizes the networking implementation for Firecracker microVMs in metald, including the dual-stack network architecture, systemd integration challenges, and the solutions implemented.

## Architecture

### Network Topology
- **Bridge**: `br-vms` (10.100.0.1/16) - Central bridge for VM connectivity
- **VM Networks**: Each VM gets:
  - Unique IP from 10.100.0.0/16 subnet
  - TAP device for Firecracker integration
  - veth pair connecting to bridge
  - Isolated network namespace
  - Rate limiting (1 Gbps default)

### Components

1. **Network Manager** (`internal/network/`)
   - IP allocation with sequential assignment
   - Network namespace management
   - TAP/veth device creation
   - Bridge attachment
   - Dual-stack ready (IPv4 + IPv6)

2. **Process Manager Integration**
   - Creates network namespace before jailer starts
   - Passes network info to Firecracker process
   - Coordinates namespace naming with jailer

3. **API Enhancement**
   - `GetVmInfo` now returns network details
   - Network info includes IP, MAC, TAP device, namespace, gateway, DNS

## Implementation Details

### Key Files Modified
- `internal/network/implementation.go` - Core network manager
- `internal/network/types.go` - Network data structures
- `internal/process/manager.go` - Network integration with jailer
- `proto/vmprovisioner/v1/vm.proto` - Added VmNetworkInfo message
- `internal/service/vm.go` - Return network info in API
- `internal/backend/firecracker/managed_client.go` - Populate network info

### Systemd Challenge and Solution

**Problem**: Systemd's network isolation prevented metald from accessing the host bridge, even with:
- `PrivateNetwork=no`
- `CAP_NET_ADMIN` capability
- All security restrictions disabled

**Root Cause**: Systemd applies subtle network namespace restrictions that make host network interfaces invisible to services.

**Solution**: Use `nsenter -t 1 -n` to execute network commands in PID 1's namespace:

```go
if useNsenter {
    cmd = exec.Command("nsenter", "-t", "1", "-n", "ip", "link", "add", vethHost, "type", "veth", "peer", "name", vethNS)
} else {
    cmd = exec.Command("ip", "link", "add", vethHost, "type", "veth", "peer", "name", vethNS)
}
```

This workaround allows network operations to succeed even when running under systemd's restrictions.

## Configuration

### Environment Variables
```bash
UNKEY_METALD_NETWORK_ENABLED=true
UNKEY_METALD_NETWORK_IPV4_ENABLED=true
UNKEY_METALD_NETWORK_IPV6_ENABLED=true
UNKEY_METALD_NETWORK_BRIDGE=br-vms
UNKEY_METALD_NETWORK_BRIDGE_IPV4=10.100.0.1/16
UNKEY_METALD_NETWORK_BRIDGE_IPV6=fd00::1/64
UNKEY_METALD_NETWORK_RATE_LIMIT_MBPS=1000
```

### Systemd Service Requirements
```ini
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_RAW CAP_SYS_ADMIN CAP_DAC_OVERRIDE CAP_MKNOD
PrivateNetwork=no
ProtectSystem=false
ReadWritePaths=/srv/jailer /opt/metald /var/run/netns /tmp /opt/metald/assets /sys
```

## Usage

### Create VM with Network
```bash
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_customer_test" \
  -d '{
    "config": {
      "cpu": {"vcpu_count": 1},
      "memory": {"size_bytes": 134217728},
      "boot": {
        "kernel_path": "/opt/vm-assets/vmlinux",
        "kernel_args": "console=ttyS0 reboot=k panic=1 pci=off"
      },
      "storage": [{
        "path": "/opt/vm-assets/rootfs.ext4",
        "readonly": false
      }]
    }
  }'
```

### Get VM Info with Network Details
```bash
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/GetVmInfo \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_customer_test" \
  -d '{"vmId": "YOUR_VM_ID"}' | jq .networkInfo
```

Response includes:
```json
{
  "ipAddress": "10.100.0.2",
  "macAddress": "02:00:66:68:8d:36",
  "tapDevice": "tapud-4f507",
  "networkNamespace": "fc-vm-1749937352193000395",
  "gateway": "10.100.0.1",
  "dnsServers": ["8.8.8.8", "8.8.4.4"]
}
```

### Test Connectivity
```bash
# Ping VM from host
ping 10.100.0.2

# Check network interfaces
ip link show br-vms
brctl show br-vms
```

## Future Enhancements

1. **IPv6 Support**: Framework is ready for dual-stack, needs:
   - IPv6 address allocation
   - Router advertisements
   - DHCPv6 or SLAAC

2. **Network Policies**: Could add:
   - Per-VM firewall rules
   - Traffic shaping policies
   - VLAN tagging

3. **Advanced Features**:
   - Multiple network interfaces per VM
   - SR-IOV support
   - Custom DNS configuration per VM

## Troubleshooting

### Common Issues

1. **"Link not found" errors**: Check if using systemd service, may need nsenter workaround
2. **Bridge not accessible**: Verify systemd service has correct capabilities
3. **VMs not pingable**: Check iptables NAT rules and IP forwarding

### Debug Commands
```bash
# Check bridge and interfaces
ip link show | grep -E "(br-vms|tap|veth)"

# Check network namespaces
ip netns list

# Check iptables NAT
sudo iptables -t nat -L POSTROUTING -v

# Check metald logs
sudo journalctl -u metald -f | grep -i network
```