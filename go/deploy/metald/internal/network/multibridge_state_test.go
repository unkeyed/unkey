package network

import (
	"log/slog"
	"os"
	"testing"
)

// Test state validation and repair functionality
func TestStateValidationAndRepair(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mbm := NewMultiBridgeManager(8, "br-tenant", logger)

	t.Run("ValidateAndRepairExcessiveVMCount", func(t *testing.T) {
		// Create a workspace with invalid VM count (exceeds /29 capacity)
		badAllocation := &WorkspaceAllocation{
			WorkspaceID:  "corrupted-workspace",
			BridgeNumber: 2,
			BridgeName:   "br-tenant-2",
			VMCount:      25, // This exceeds the 5 VM limit
		}
		mbm.workspaces["corrupted-workspace"] = badAllocation

		// Run validation and repair
		repaired := mbm.validateAndRepairState()

		if !repaired {
			t.Error("Expected state validation to detect and repair excessive VM count")
		}

		// Check that VM count was reset to 0
		repairedAllocation := mbm.workspaces["corrupted-workspace"]
		if repairedAllocation.VMCount != 0 {
			t.Errorf("Expected VM count to be reset to 0, got %d", repairedAllocation.VMCount)
		}
	})

	t.Run("ValidateAndRepairInvalidBridgeNumber", func(t *testing.T) {
		// Create a workspace with invalid bridge number
		badAllocation := &WorkspaceAllocation{
			WorkspaceID:  "invalid-bridge-workspace",
			BridgeNumber: 10, // Out of range for 8-bridge setup
			BridgeName:   "br-tenant-10",
			VMCount:      2,
		}
		mbm.workspaces["invalid-bridge-workspace"] = badAllocation

		// Run validation and repair
		repaired := mbm.validateAndRepairState()

		if !repaired {
			t.Error("Expected state validation to detect and repair invalid bridge number")
		}

		// Check that workspace was deleted
		if _, exists := mbm.workspaces["invalid-bridge-workspace"]; exists {
			t.Error("Expected workspace with invalid bridge number to be deleted")
		}
	})

	t.Run("ValidateAndRepairBridgeName", func(t *testing.T) {
		// Create a workspace with incorrect bridge name
		badAllocation := &WorkspaceAllocation{
			WorkspaceID:  "wrong-name-workspace",
			BridgeNumber: 3,
			BridgeName:   "old-bridge-name", // Should be br-tenant-3
			VMCount:      2,
		}
		mbm.workspaces["wrong-name-workspace"] = badAllocation

		// Run validation and repair
		repaired := mbm.validateAndRepairState()

		if !repaired {
			t.Error("Expected state validation to detect and repair incorrect bridge name")
		}

		// Check that bridge name was corrected
		repairedAllocation := mbm.workspaces["wrong-name-workspace"]
		expectedName := "br-tenant-3"
		if repairedAllocation.BridgeName != expectedName {
			t.Errorf("Expected bridge name to be corrected to %s, got %s",
				expectedName, repairedAllocation.BridgeName)
		}
	})

	t.Run("ValidStateNoRepairNeeded", func(t *testing.T) {
		// Create a workspace with valid state
		validAllocation := &WorkspaceAllocation{
			WorkspaceID:  "valid-workspace",
			BridgeNumber: 1,
			BridgeName:   "br-tenant-1",
			VMCount:      3, // Within 5 VM limit
		}
		mbm.workspaces["valid-workspace"] = validAllocation

		// Run validation and repair
		repaired := mbm.validateAndRepairState()

		if repaired {
			t.Error("Expected no repairs needed for valid state")
		}

		// Check that valid workspace remains unchanged
		allocation := mbm.workspaces["valid-workspace"]
		if allocation.VMCount != 3 {
			t.Errorf("Expected VM count to remain 3, got %d", allocation.VMCount)
		}
		if allocation.BridgeName != "br-tenant-1" {
			t.Errorf("Expected bridge name to remain br-tenant-1, got %s", allocation.BridgeName)
		}
	})

	t.Run("ValidateOrphanedBridgeUsage", func(t *testing.T) {
		// Create orphaned bridge usage entry (workspace doesn't exist)
		if mbm.bridgeUsage[5] == nil {
			mbm.bridgeUsage[5] = make(map[string]bool)
		}
		mbm.bridgeUsage[5]["nonexistent-workspace"] = true

		// Run validation and repair
		repaired := mbm.validateAndRepairState()

		if !repaired {
			t.Error("Expected state validation to detect and repair orphaned bridge usage")
		}

		// Check that orphaned entry was removed
		if mbm.bridgeUsage[5] != nil {
			if _, exists := mbm.bridgeUsage[5]["nonexistent-workspace"]; exists {
				t.Error("Expected orphaned bridge usage entry to be removed")
			}
		}
	})
}

