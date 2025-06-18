# Unified Naming Convention for metald

## Overview
All network devices, processes, and namespaces now use underscores for consistency and better terminal double-click selection.

## Naming Patterns

### Network Devices (per VM)
Generated from internal 8-character network ID (e.g., `a1b2c3d4`):
- **TAP Device**: `tap_a1b2c3d4` (12 chars)
- **Veth Host**: `vh_a1b2c3d4` (10 chars)
- **Veth Namespace**: `vn_a1b2c3d4` (10 chars)
- **VM Namespace**: `ns_vm_a1b2c3d4`

### Process Management
- **Process ID**: `fc_{timestamp}` (e.g., `fc_1234567890`)
- **Jailer ID**: `vm_{timestamp}` (e.g., `vm_1234567890`)
- **Jailer Namespace**: `fc_vm_{timestamp}` (e.g., `fc_vm_1234567890`)

### File Paths
- **Socket (non-jailer)**: `/var/run/metald/sockets/fc_1234567890.sock`
- **Socket (jailer)**: `/var/lib/jailer/firecracker/vm_1234567890/root/run/firecracker.socket`
- **Log file**: `/var/log/metald/fc_1234567890.log`
- **PID file**: `/var/log/metald/fc_1234567890.pid`

### Persistent Resources
- **Bridge**: `br-vms` (kept with hyphen for backward compatibility)

## Benefits
1. **Consistent**: All generated names use underscores
2. **Selectable**: Double-click selects entire name in terminals
3. **Unique**: No collision with system or other tools' naming
4. **Traceable**: Easy to grep and debug

## Collision Analysis
Our patterns are unique and don't conflict with:
- Docker: Uses `veth` (no underscore) + hex
- KVM/libvirt: Uses `vnet0`, `virbr0` (no underscore)
- Kubernetes: Uses plugin-specific prefixes (`cali*`, `weave*`)
- System interfaces: Use `en*`, `wl*`, `ww*` prefixes

## Implementation Details
- Network IDs are generated using crypto/rand (8 hex chars)
- All names respect Linux's 15-character limit for interfaces
- Network IDs are stored with VMs to ensure correct cleanup
- Only interfaces with our specific prefixes are ever deleted