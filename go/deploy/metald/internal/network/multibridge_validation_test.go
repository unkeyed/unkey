package network

import (
	"log/slog"
	"net"
	"os"
	"testing"
)

// Test subnet allocation validation
func TestMultiBridgeValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mbm := NewMultiBridgeManager(8, "br-tenant", logger)

	t.Run("ValidateBridgeAllocation", func(t *testing.T) {
		// Valid allocation
		validAllocation := &WorkspaceAllocation{
			WorkspaceID:  "test-workspace",
			BridgeNumber: 2,
			BridgeName:   "br-tenant-2",
			VMCount:      3,
		}

		if err := mbm.validateBridgeAllocation(validAllocation); err != nil {
			t.Errorf("Valid allocation should not fail: %v", err)
		}

		// Invalid bridge number (out of range)
		invalidBridge := &WorkspaceAllocation{
			WorkspaceID:  "test-workspace",
			BridgeNumber: 8, // Out of range for 8-bridge setup
			BridgeName:   "br-tenant-8",
			VMCount:      1,
		}

		if err := mbm.validateBridgeAllocation(invalidBridge); err == nil {
			t.Error("Out-of-range bridge should fail validation")
		}

		// Invalid VM count (too many)
		tooManyVMs := &WorkspaceAllocation{
			WorkspaceID:  "test-workspace",
			BridgeNumber: 1,
			BridgeName:   "br-tenant-1",
			VMCount:      5, // Exceeds /29 capacity
		}

		if err := mbm.validateBridgeAllocation(tooManyVMs); err == nil {
			t.Error("Excessive VM count should fail validation")
		}

		// Empty workspace ID
		emptyWorkspace := &WorkspaceAllocation{
			WorkspaceID:  "",
			BridgeNumber: 1,
			BridgeName:   "br-tenant-1",
			VMCount:      1,
		}

		if err := mbm.validateBridgeAllocation(emptyWorkspace); err == nil {
			t.Error("Empty workspace ID should fail validation")
		}
	})

	t.Run("ValidateVMIPAllocation", func(t *testing.T) {
		// Valid VM IP allocation
		if err := mbm.validateVMIPAllocation("test-workspace", 16, 18, 0); err != nil {
			t.Errorf("Valid VM IP allocation should not fail: %v", err)
		}

		// Invalid subnet base (not /29 aligned)
		if err := mbm.validateVMIPAllocation("test-workspace", 17, 19, 0); err == nil {
			t.Error("Non-aligned subnet base should fail validation")
		}

		// VM IP outside workspace range
		if err := mbm.validateVMIPAllocation("test-workspace", 16, 23, 0); err == nil {
			t.Error("VM IP outside workspace range should fail validation")
		}

		// VM count mismatch
		if err := mbm.validateVMIPAllocation("test-workspace", 16, 19, 0); err == nil {
			t.Error("VM IP calculation mismatch should fail validation")
		}
	})

	t.Run("ValidateIPWithinWorkspaceSubnet", func(t *testing.T) {
		// Valid VM IP within workspace subnet
		validIP := net.ParseIP("172.16.2.18") // .18 = base 16 + 2 (first VM IP)
		if err := mbm.validateIPWithinWorkspaceSubnet(validIP, 2, 16); err != nil {
			t.Errorf("Valid VM IP should not fail: %v", err)
		}

		// Reserved network address (.0)
		networkIP := net.ParseIP("172.16.2.16") // .16 = network address for /29
		if err := mbm.validateIPWithinWorkspaceSubnet(networkIP, 2, 16); err == nil {
			t.Error("Network address should fail validation")
		}

		// Reserved gateway address (.1)
		gatewayIP := net.ParseIP("172.16.2.17") // .17 = gateway for /29
		if err := mbm.validateIPWithinWorkspaceSubnet(gatewayIP, 2, 16); err == nil {
			t.Error("Gateway address should fail validation")
		}

		// Reserved broadcast address (.7)
		broadcastIP := net.ParseIP("172.16.2.23") // .23 = broadcast for /29
		if err := mbm.validateIPWithinWorkspaceSubnet(broadcastIP, 2, 16); err == nil {
			t.Error("Broadcast address should fail validation")
		}

		// IP outside workspace subnet entirely
		outsideIP := net.ParseIP("172.16.2.24") // Outside the /29 subnet
		if err := mbm.validateIPWithinWorkspaceSubnet(outsideIP, 2, 16); err == nil {
			t.Error("IP outside workspace subnet should fail validation")
		}
	})

	t.Run("ValidateSubnetBoundaries", func(t *testing.T) {
		// Test that /29 subnets don't overlap
		// Bridge 2: 172.16.2.0/24
		// Workspace subnets: .0/29, .8/29, .16/29, .24/29, etc.

		// First workspace: 172.16.2.0/29 (IPs .0-.7)
		workspace1Base := 0
		vm1IP := net.ParseIP("172.16.2.2") // First VM in first workspace
		if err := mbm.validateIPWithinWorkspaceSubnet(vm1IP, 2, workspace1Base); err != nil {
			t.Errorf("First workspace VM should be valid: %v", err)
		}

		// Second workspace: 172.16.2.8/29 (IPs .8-.15)
		workspace2Base := 8
		vm2IP := net.ParseIP("172.16.2.10") // First VM in second workspace
		if err := mbm.validateIPWithinWorkspaceSubnet(vm2IP, 2, workspace2Base); err != nil {
			t.Errorf("Second workspace VM should be valid: %v", err)
		}

		// Verify first workspace VM is NOT valid in second workspace
		if err := mbm.validateIPWithinWorkspaceSubnet(vm1IP, 2, workspace2Base); err == nil {
			t.Error("VM from different workspace should fail validation")
		}
	})
}
