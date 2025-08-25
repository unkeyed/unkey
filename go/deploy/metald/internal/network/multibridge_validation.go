package network

import (
	"fmt"
	"log/slog"
	"net"
	"time"
)

// validateState performs comprehensive validation of loaded state
func (mbm *MultiBridgeManager) validateState(state *MultiBridgeState) error {
	if state == nil {
		return fmt.Errorf("state is nil")
	}

	// Validate workspaces
	for wsID, ws := range state.Workspaces {
		if err := mbm.validateWorkspace(wsID, ws); err != nil {
			return fmt.Errorf("workspace %s validation failed: %w", wsID, err)
		}
	}

	// Validate bridge usage consistency
	if err := mbm.validateBridgeUsage(state); err != nil {
		return fmt.Errorf("bridge usage validation failed: %w", err)
	}

	return nil
}

// validateWorkspace validates a single workspace allocation
func (mbm *MultiBridgeManager) validateWorkspace(wsID string, ws *WorkspaceAllocation) error {
	if ws == nil {
		return fmt.Errorf("workspace allocation is nil")
	}

	// Validate workspace ID consistency
	if ws.WorkspaceID != wsID {
		return fmt.Errorf("workspace ID mismatch: map key %s != struct field %s", wsID, ws.WorkspaceID)
	}

	// Validate bridge number bounds
	if ws.BridgeNumber < 0 || ws.BridgeNumber >= mbm.bridgeCount {
		return fmt.Errorf("bridge number %d out of range [0, %d)", ws.BridgeNumber, mbm.bridgeCount)
	}

	// Validate bridge name consistency
	expectedBridgeName := fmt.Sprintf("%s-%d", mbm.bridgePrefix, ws.BridgeNumber)
	if ws.BridgeName != expectedBridgeName {
		return fmt.Errorf("bridge name mismatch: got %s, expected %s", ws.BridgeName, expectedBridgeName)
	}

	// Validate VM count is not negative
	if ws.VMCount < 0 {
		return fmt.Errorf("VM count %d cannot be negative", ws.VMCount)
	}

	// Validate project VLANs
	if ws.ProjectVLANs == nil {
		return fmt.Errorf("project VLANs map is nil")
	}

	for projID, vlan := range ws.ProjectVLANs {
		if err := mbm.validateProjectVLAN(projID, vlan); err != nil {
			return fmt.Errorf("project %s VLAN validation failed: %w", projID, err)
		}
	}

	// Validate created_at timestamp
	if ws.CreatedAt == "" {
		return fmt.Errorf("created_at timestamp is empty")
	}
	if _, err := time.Parse(time.RFC3339, ws.CreatedAt); err != nil {
		return fmt.Errorf("invalid created_at timestamp %s: %w", ws.CreatedAt, err)
	}

	return nil
}

// validateProjectVLAN validates a project VLAN configuration
func (mbm *MultiBridgeManager) validateProjectVLAN(projID string, vlan *ProjectVLAN) error {
	if vlan == nil {
		return fmt.Errorf("project VLAN is nil")
	}

	// Validate project ID consistency
	if vlan.ProjectID != projID {
		return fmt.Errorf("project ID mismatch: map key %s != struct field %s", projID, vlan.ProjectID)
	}

	// Validate VLAN number bounds
	if vlan.VLANNumber < mbm.vlanRangeStart || vlan.VLANNumber > mbm.vlanRangeEnd {
		return fmt.Errorf("VLAN number %d out of range [%d, %d]", vlan.VLANNumber, mbm.vlanRangeStart, mbm.vlanRangeEnd)
	}

	// Validate subnet CIDR format
	if vlan.SubnetCIDR == "" {
		return fmt.Errorf("subnet CIDR is empty")
	}
	if _, _, err := net.ParseCIDR(vlan.SubnetCIDR); err != nil {
		return fmt.Errorf("invalid subnet CIDR %s: %w", vlan.SubnetCIDR, err)
	}

	// Validate VM index bounds
	if vlan.NextVMIndex < 0 {
		return fmt.Errorf("next VM index %d cannot be negative", vlan.NextVMIndex)
	}
	if vlan.NextVMIndex > 30 { // /27 subnet has max 30 usable IPs
		return fmt.Errorf("next VM index %d exceeds /27 subnet capacity", vlan.NextVMIndex)
	}

	return nil
}

// validateBridgeUsage validates bridge usage consistency with workspace allocations
func (mbm *MultiBridgeManager) validateBridgeUsage(state *MultiBridgeState) error {
	if state.BridgeUsage == nil {
		return fmt.Errorf("bridge usage map is nil")
	}

	// Build expected bridge usage from workspaces
	expectedUsage := make(map[int]map[string]bool)
	for wsID, ws := range state.Workspaces {
		bridgeNum := ws.BridgeNumber
		if expectedUsage[bridgeNum] == nil {
			expectedUsage[bridgeNum] = make(map[string]bool)
		}
		expectedUsage[bridgeNum][wsID] = true
	}

	// Validate bridge usage matches workspace allocations
	for bridgeNum, usage := range state.BridgeUsage {
		// Validate bridge number bounds
		if bridgeNum < 0 || bridgeNum >= mbm.bridgeCount {
			return fmt.Errorf("bridge usage contains invalid bridge number %d", bridgeNum)
		}

		// Validate usage map consistency
		expected := expectedUsage[bridgeNum]
		if len(usage) != len(expected) {
			return fmt.Errorf("bridge %d usage count mismatch: got %d, expected %d", bridgeNum, len(usage), len(expected))
		}

		for wsID := range usage {
			if !expected[wsID] {
				return fmt.Errorf("bridge %d usage contains unexpected workspace %s", bridgeNum, wsID)
			}
		}
	}

	// Validate all expected bridges are represented
	for bridgeNum, expected := range expectedUsage {
		actual := state.BridgeUsage[bridgeNum]
		if len(actual) != len(expected) {
			return fmt.Errorf("bridge %d missing from usage map or count mismatch", bridgeNum)
		}
	}

	return nil
}

// validateAndRepairState validates loaded state and repairs any inconsistencies
// Returns true if repairs were made, false if state was already valid
func (mbm *MultiBridgeManager) validateAndRepairState() bool {
	var repaired bool
	var repairedWorkspaces []string

	for workspaceID, allocation := range mbm.workspaces {
		originalVMCount := allocation.VMCount

		// Validate bridge number is within bounds
		if allocation.BridgeNumber < 0 || allocation.BridgeNumber >= mbm.bridgeCount {
			mbm.logger.Warn("invalid bridge number in state, resetting workspace",
				slog.String("workspace_id", workspaceID),
				slog.Int("invalid_bridge", allocation.BridgeNumber),
				slog.Int("max_bridges", mbm.bridgeCount-1),
			)
			delete(mbm.workspaces, workspaceID)
			repaired = true
			repairedWorkspaces = append(repairedWorkspaces, workspaceID+" (deleted)")
			continue
		}

		// Validate and repair VM count - enforce /29 subnet limit (5 VMs max)
		if allocation.VMCount > 5 {
			mbm.logger.Warn("workspace VM count exceeds /29 capacity, resetting to 0",
				slog.String("workspace_id", workspaceID),
				slog.Int("invalid_vm_count", allocation.VMCount),
				slog.Int("max_vm_count", 5),
			)
			allocation.VMCount = 0
			repaired = true
			repairedWorkspaces = append(repairedWorkspaces,
				fmt.Sprintf("%s (vm_count: %d->0)", workspaceID, originalVMCount))
		}

		// Validate bridge name format
		expectedBridgeName := fmt.Sprintf("%s-%d", mbm.bridgePrefix, allocation.BridgeNumber)
		if allocation.BridgeName != expectedBridgeName {
			mbm.logger.Warn("invalid bridge name in state, correcting",
				slog.String("workspace_id", workspaceID),
				slog.String("invalid_name", allocation.BridgeName),
				slog.String("expected_name", expectedBridgeName),
			)
			allocation.BridgeName = expectedBridgeName
			repaired = true
		}

		// Ensure bridge usage tracking is consistent
		if mbm.bridgeUsage[allocation.BridgeNumber] == nil {
			mbm.bridgeUsage[allocation.BridgeNumber] = make(map[string]bool)
		}
		mbm.bridgeUsage[allocation.BridgeNumber][workspaceID] = true
	}

	// Clean up orphaned bridge usage entries
	for bridgeNum, workspaceMap := range mbm.bridgeUsage {
		for workspaceID := range workspaceMap {
			if _, exists := mbm.workspaces[workspaceID]; !exists {
				delete(workspaceMap, workspaceID)
				repaired = true
				mbm.logger.Warn("removed orphaned bridge usage entry",
					slog.String("workspace_id", workspaceID),
					slog.Int("bridge_number", bridgeNum),
				)
			}
		}
		// Clean up empty bridge usage maps
		if len(workspaceMap) == 0 {
			delete(mbm.bridgeUsage, bridgeNum)
		}
	}

	// Persist repaired state
	if repaired {
		mbm.logger.Info("state validation found issues, applying repairs",
			slog.Int("repaired_workspace_count", len(repairedWorkspaces)),
			slog.Any("repaired_workspaces", repairedWorkspaces),
		)

		if err := mbm.saveState(); err != nil {
			mbm.logger.Error("failed to persist state repairs",
				slog.String("error", err.Error()),
			)
			// Don't return false - repairs were still applied in memory
		} else {
			mbm.logger.Info("state repairs persisted successfully")
		}
	}

	return repaired
}
