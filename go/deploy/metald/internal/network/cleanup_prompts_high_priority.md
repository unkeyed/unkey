# High Priority Cleanup Tasks for metald/internal/network

## 1. Consolidate Duplicate calculateVethHostIP Functions

**Task**: Remove duplicate implementations of IP calculation logic

**Files to modify**:
- `types.go:140` - Remove the method implementation
- `types.go:120` - Update to call standalone function
- `vm_network_setup.go:298` - Keep this standalone function

**Prompt**:
```
Consolidate the duplicate calculateVethHostIP functions in the metald network package:

1. Remove the calculateVethHostIP method from types.go:140
2. Update the KernelCmdlineArgs method in types.go:120 to call the standalone calculateVethHostIP function from vm_network_setup.go:298
3. Ensure all tests still pass
4. Verify that the standalone function in vm_network_setup.go handles all the same cases as the removed method
5. Update any other references to use the standalone function

The standalone function should remain as the canonical implementation since it's more testable and used by multiple components.
```

## 2. Replace Sysctl Shell Command with Direct File Write

**Task**: Eliminate shell dependency for IP forwarding configuration

**File to modify**: `bridge_lifecycle.go:135`

**Prompt**:
```
Replace the sysctl shell command with direct file system write in bridge_lifecycle.go:

Current code around line 135:
```go
cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
```

Replace with:
```go
err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1"), 0644)
```

Requirements:
1. Remove the exec.Command usage
2. Add proper error handling for the file write
3. Update log messages to reflect the new approach
4. Ensure the same functionality (enabling IP forwarding)
5. Add appropriate comments explaining the direct proc filesystem approach
6. Test that this works equivalently to the sysctl command
```

## 3. Replace Bridge VLAN Shell Commands with Netlink Library

**Task**: Use existing netlink library instead of shell commands

**Files to modify**:
- `workspace.go:214` - VLAN add command
- `workspace.go:303` - VLAN add with untagged/pvid
- `workspace.go:373` - VLAN delete command

**Prompt**:
```
Replace bridge VLAN shell commands with netlink library calls in workspace.go:

Current shell commands to replace:
1. Line 214: `exec.Command("bridge", "vlan", "add", "vid", fmt.Sprintf("%d", vlanID), "dev", wm.bridgeName, "self")`
2. Line 303: `exec.Command("bridge", "vlan", "add", "vid", fmt.Sprintf("%d", workspaceVLAN.VLANBase), "dev", interfaceName, "untagged", "pvid")`
3. Line 373: `exec.Command("bridge", "vlan", "del", "vid", fmt.Sprintf("%d", workspaceVLAN.VLANBase), "dev", wm.bridgeName, "self")`

Requirements:
1. Use the already-imported github.com/vishvananda/netlink library
2. Replace with appropriate netlink.BridgeVlan* function calls
3. Maintain the same error handling and logging patterns
4. Remove the #nosec G204 comments since shell injection is no longer possible
5. Ensure equivalent functionality for self, untagged, and pvid flags
6. Test that the netlink approach works identically to the shell commands
7. Update any related documentation or comments

The netlink library should have functions like netlink.BridgeVlanAdd() and netlink.BridgeVlanDel() that can replace these shell commands.
```

## 4. Replace Firewall Shell Commands with Go Libraries

**Task**: Replace nftables and iptables shell commands with Go libraries

**Files to modify**:
- `port_forwarding.go` (lines 37, 63, 94, 120, 126)
- `bridge_manager.go:282`
- `bridge_lifecycle.go:231`

**Prompt**:
```
Replace firewall shell commands with Go libraries in the network package:

Target files and operations:
1. port_forwarding.go: nftables commands for DNAT rules
2. bridge_manager.go:282: iptables commands for bridge rules
3. bridge_lifecycle.go:231: iptables cleanup commands

Recommended libraries (verify they're actively maintained):
- github.com/google/nftables for nftables operations
- github.com/coreos/go-iptables for iptables operations

Requirements:
1. Add the appropriate library dependencies to go.mod
2. Replace all exec.Command calls with library functions
3. Maintain identical firewall rule functionality
4. Preserve all error handling and logging
5. Remove shell command injection risks
6. Ensure proper cleanup of rules on failures
7. Add comprehensive tests for the new implementations
8. Update documentation and comments to reflect the new approach

Priority order:
1. Start with port_forwarding.go nftables commands (most critical for security)
2. Then bridge_manager.go iptables rules
3. Finally bridge_lifecycle.go cleanup commands

Each replacement should be thoroughly tested to ensure equivalent functionality.
```

## 5. Consolidate Interface Name Validation Functions

**Task**: Use the more robust validation function consistently

**Files to modify**:
- `vm_network_setup.go:430` - Replace `isValidInterfaceName()`
- Update all callers to use `validateNetworkDeviceName()` from `workspace.go:24`

**Prompt**:
```
Consolidate interface name validation functions in the network package:

Current situation:
- vm_network_setup.go:430 has isValidInterfaceName() with basic validation
- workspace.go:24 has validateNetworkDeviceName() with comprehensive validation and security checks

Task:
1. Remove the isValidInterfaceName() function from vm_network_setup.go:430
2. Update all callers of isValidInterfaceName() to use validateNetworkDeviceName()
3. Move validateNetworkDeviceName() to a more central location if needed for better reusability
4. Ensure the more comprehensive validation doesn't break existing functionality
5. Update any tests that depend on the removed function
6. Verify that all interface name validation now uses the security-conscious approach

The validateNetworkDeviceName() function is preferred because it:
- Has better error messages
- Includes security considerations (command injection prevention)
- Has more comprehensive validation rules

Ensure backward compatibility while improving security posture.
```