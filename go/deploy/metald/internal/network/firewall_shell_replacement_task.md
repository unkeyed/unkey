# Task 4: Replace Firewall Shell Commands with Go Libraries

**Objective**: Replace all nftables and iptables shell command executions with proper Go libraries to eliminate shell injection risks and improve reliability.

**Priority**: High (Critical for security - eliminates shell command injection attack vectors)

## Files to Modify

1. **port_forwarding.go** (5 shell commands):
   - Line 37: `exec.Command("bash", "-c", rule)` - nftables DNAT rule creation
   - Line 63: `exec.Command("bash", "-c", listCmd)` - nftables rule listing
   - Line 94: `exec.Command("bash", "-c", deleteCmd)` - nftables rule deletion
   - Line 120: `exec.Command("bash", "-c", tableCmd)` - nftables table creation
   - Line 126: `exec.Command("bash", "-c", chainCmd)` - nftables chain creation

2. **bridge_manager.go**:
   - Line 282: `exec.Command("iptables", rule...)` - iptables NAT rules

3. **bridge_lifecycle.go**:
   - Line 230: `exec.Command("bash", "-c", "iptables "+deleteRule)` - iptables cleanup

## Recommended Libraries

1. **github.com/google/nftables** (for nftables operations)
   - Well-maintained by Google
   - Pure Go implementation
   - Supports tables, chains, rules, and NAT operations

2. **github.com/coreos/go-iptables** (for iptables operations)
   - Maintained by CoreOS
   - Simple, reliable interface
   - Supports rule addition, deletion, and listing

## Implementation Strategy

### Phase 1: Add Dependencies
```bash
go get github.com/google/nftables
go get github.com/coreos/go-iptables/iptables
```

### Phase 2: Port Forwarding (port_forwarding.go) - HIGHEST PRIORITY

**Current nftables commands to replace:**
1. `nft add rule ip nat PREROUTING tcp dport %d dnat to %s:%d`
2. `nft add rule ip nat PREROUTING udp dport %d dnat to %s:%d`
3. `nft --handle list chain ip nat PREROUTING`
4. `nft delete rule ip nat PREROUTING handle %s`
5. `nft add table ip nat`
6. `nft add chain ip nat PREROUTING { type nat hook prerouting priority -100; }`

**Implementation requirements:**
- Use `nftables.Conn` to manage nftables connection
- Create table and chain using `AddTable()` and `AddChain()`
- Add DNAT rules using `AddRule()` with `expr.NAT` expressions
- List rules using `GetRules()` and delete using `DelRule()`
- Handle both TCP and UDP protocols
- Maintain identical logging and error handling

### Phase 3: Bridge NAT Rules (bridge_manager.go)

**Current iptables commands to replace:**
- Various NAT rules for bridge networking (examine the `rules` slice)

**Implementation requirements:**
- Use `iptables.New()` to create iptables instance
- Replace `exec.Command("iptables", rule...)` with appropriate `iptables.Append()` calls
- Update cleanup to use `iptables.Delete()` instead of shell commands
- Preserve rule tracking in `iptablesRules` slice for cleanup

### Phase 4: Cleanup Rules (bridge_lifecycle.go)

**Current cleanup logic to replace:**
- String manipulation to convert ADD → DELETE rules
- Shell execution of iptables delete commands

**Implementation requirements:**
- Use stored rule references instead of string manipulation
- Call `iptables.Delete()` with proper parameters
- Maintain existing error handling (log warnings, don't fail)

## Detailed Function Specifications

### nftables Implementation (port_forwarding.go)

```go
type NFTManager struct {
    conn *nftables.Conn
    table *nftables.Table
    chain *nftables.Chain
}

func (m *Manager) ensureNftablesTable() error {
    // Create connection
    // Add table if not exists
    // Add chain if not exists
    // Call conn.Flush()
}

func (m *Manager) setupPortForwarding(vmIP net.IP, mapping PortMapping) error {
    // Create DNAT rule using expr.NAT
    // Handle both TCP and UDP protocols
    // Add rule to chain
    // Flush changes
}

func (m *Manager) removePortForwarding(vmIP net.IP, mapping PortMapping) error {
    // List existing rules
    // Find matching rule by comparing expressions
    // Delete rule by reference
    // Flush changes
}
```

### iptables Implementation (bridge_manager.go)

```go
func (m *Manager) setupNAT(bridgeName, bridgeIP string) error {
    ipt, err := iptables.New()
    if err != nil {
        return err
    }
    
    // Convert shell rules to library calls
    // Use ipt.Append() instead of exec.Command()
    // Store rule identifiers for cleanup
}
```

## Testing Requirements

1. **Unit Tests**: Create tests that verify rule creation without requiring root privileges
2. **Integration Tests**: Test with actual nftables/iptables (requires privileged environment)
3. **Functionality Tests**: Ensure port forwarding works identically
4. **Cleanup Tests**: Verify proper rule removal on errors and shutdown
5. **Protocol Tests**: Test both TCP and UDP forwarding

## Error Handling

1. **Connection Errors**: Handle nftables/iptables daemon unavailability
2. **Permission Errors**: Provide clear messages about required privileges
3. **Rule Conflicts**: Handle existing rules gracefully
4. **Cleanup Failures**: Log warnings but don't fail critical operations

## Documentation Updates

1. Update function comments to reflect library usage
2. Add AIDEV-NOTE comments for complex nftables/iptables logic
3. Document any behavior changes from shell → library transition
4. Update error messages to be more specific

## Rollback Strategy

1. Keep shell command implementations as fallback functions initially
2. Add feature flag to switch between implementations
3. Test thoroughly before removing shell commands entirely
4. Ensure existing functionality is preserved exactly

## Success Criteria

- [ ] Zero shell command executions for firewall operations
- [ ] All existing functionality preserved
- [ ] Comprehensive test coverage
- [ ] Proper error handling and logging
- [ ] Clean rule cleanup on failures
- [ ] Performance equal or better than shell commands
- [ ] Security improvement: no shell injection possible

This implementation will significantly improve security by eliminating shell command injection attack vectors while maintaining all existing firewall functionality.