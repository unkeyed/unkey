package network

import (
	"fmt"
	"hash/fnv"
	"log/slog"
	"sync"
	"time"
)

// AIDEV-NOTE: Security-focused multi-tenant architecture with Layer 2 isolation
// - workspace_id (tenant) deterministically maps to dedicated bridge (0-31)
// - project_id deterministically maps to VLAN within tenant bridge
// - Each tenant bridge gets /24 subnet: 172.16.{bridge_num}.0/24
// - Each project VLAN gets /27 subnet: 172.16.{bridge_num}.{vlan_offset}/27
// - OUI-based MAC addresses for tenant identification: 02:{bridge_hex}:4B:XX:XX:XX

// MultiBridgeManager manages workspace allocation across multiple bridges
type MultiBridgeManager struct {
	bridgeCount    int                             // 8 or 32 bridges
	bridgePrefix   string                          // "br-vms" -> br-vms-0, br-vms-1, etc.
	workspaces     map[string]*WorkspaceAllocation // workspace_id -> allocation
	bridgeUsage    map[int]map[string]bool         // bridge_num -> workspace_id -> exists
	mu             sync.RWMutex
	vlanRangeStart int          // Starting VLAN ID (100)
	vlanRangeEnd   int          // Ending VLAN ID (4000)
	statePath      string       // Path to state persistence file
	logger         *slog.Logger // Structured logger for state operations
}

// WorkspaceAllocation represents a workspace's network allocation
type WorkspaceAllocation struct {
	WorkspaceID  string                  `json:"workspace_id"`
	BridgeNumber int                     `json:"bridge_number"` // 0-31
	BridgeName   string                  `json:"bridge_name"`   // br-vms-N
	ProjectVLANs map[string]*ProjectVLAN `json:"project_vlans"` // project_id -> VLAN info
	CreatedAt    string                  `json:"created_at"`
	VMCount      int                     `json:"vm_count"` // Track VM count for IP allocation
}

// ProjectVLAN represents a project's VLAN within a bridge
type ProjectVLAN struct {
	ProjectID   string `json:"project_id"`    // e.g., "github.com/unkeyed/unkey"
	VLANNumber  int    `json:"vlan_number"`   // VLAN ID within bridge
	SubnetCIDR  string `json:"subnet_cidr"`   // /27 subnet within bridge
	NextVMIndex int    `json:"next_vm_index"` // For IP allocation within VLAN
	CreatedAt   string `json:"created_at"`
}

// NewMultiBridgeManager creates a new multi-bridge workspace manager
func NewMultiBridgeManager(bridgeCount int, bridgePrefix string, logger *slog.Logger) *MultiBridgeManager {
	statePath := "/var/lib/metald/multibridge-state.json"

	mbm := &MultiBridgeManager{
		bridgeCount:    bridgeCount,
		bridgePrefix:   bridgePrefix,
		workspaces:     make(map[string]*WorkspaceAllocation),
		bridgeUsage:    make(map[int]map[string]bool),
		vlanRangeStart: 100,
		vlanRangeEnd:   4000,
		statePath:      statePath,
		logger:         logger.With("component", "multibridge-manager"),
	}

	// Load existing state if available
	if err := mbm.loadState(); err != nil {
		mbm.logger.Warn("failed to load state, starting with empty state",
			slog.String("error", err.Error()),
			slog.String("state_path", statePath),
		)
	} else {
		mbm.logger.Info("state loaded successfully",
			slog.String("state_path", statePath),
			slog.Int("workspace_count", len(mbm.workspaces)),
		)

		// Validate and repair state after loading
		if repaired := mbm.validateAndRepairState(); repaired {
			mbm.logger.Info("state validation completed with repairs applied")
		} else {
			mbm.logger.Debug("state validation completed, no repairs needed")
		}
	}

	return mbm
}

// GetBridgeForWorkspace deterministically maps workspace_id to bridge number
func (mbm *MultiBridgeManager) GetBridgeForWorkspace(workspaceID string) int {
	// AIDEV-BUSINESS_RULE: Use FNV hash for deterministic, even distribution
	hash := fnv.New32a()
	hash.Write([]byte(workspaceID))
	return int(hash.Sum32()) % mbm.bridgeCount
}

// GetVLANForProject deterministically maps project_id to VLAN number
func (mbm *MultiBridgeManager) GetVLANForProject(projectID string) int {
	// AIDEV-BUSINESS_RULE: Use FNV hash for deterministic VLAN assignment within range
	hash := fnv.New32a()
	hash.Write([]byte(projectID))
	vlanRange := mbm.vlanRangeEnd - mbm.vlanRangeStart + 1
	return mbm.vlanRangeStart + int(hash.Sum32())%vlanRange
}

// GetOrCreateProjectVLAN gets or creates a VLAN allocation for a project within a workspace
func (mbm *MultiBridgeManager) GetOrCreateProjectVLAN(workspaceID, projectID string) (*ProjectVLAN, *WorkspaceAllocation, error) {
	mbm.mu.Lock()
	defer mbm.mu.Unlock()

	// Get or create workspace allocation
	workspace, exists := mbm.workspaces[workspaceID]
	if !exists {
		bridgeNum := mbm.GetBridgeForWorkspace(workspaceID)
		workspace = &WorkspaceAllocation{
			WorkspaceID:  workspaceID,
			BridgeNumber: bridgeNum,
			BridgeName:   fmt.Sprintf("%s-%d", mbm.bridgePrefix, bridgeNum),
			ProjectVLANs: make(map[string]*ProjectVLAN),
			CreatedAt:    time.Now().Format(time.RFC3339),
			VMCount:      0,
		}
		mbm.workspaces[workspaceID] = workspace

		// Track bridge usage
		if mbm.bridgeUsage[bridgeNum] == nil {
			mbm.bridgeUsage[bridgeNum] = make(map[string]bool)
		}
		mbm.bridgeUsage[bridgeNum][workspaceID] = true

		mbm.logger.Info("created new workspace allocation",
			slog.String("workspace_id", workspaceID),
			slog.Int("bridge_number", bridgeNum),
			slog.String("bridge_name", workspace.BridgeName),
		)
	}

	// Get or create project VLAN
	projectVLAN, exists := workspace.ProjectVLANs[projectID]
	if !exists {
		vlanNum := mbm.GetVLANForProject(projectID)
		subnetCIDR := mbm.calculateVLANSubnet(workspace.BridgeNumber, vlanNum)

		projectVLAN = &ProjectVLAN{
			ProjectID:   projectID,
			VLANNumber:  vlanNum,
			SubnetCIDR:  subnetCIDR,
			NextVMIndex: 0,
			CreatedAt:   time.Now().Format(time.RFC3339),
		}
		workspace.ProjectVLANs[projectID] = projectVLAN

		mbm.logger.Info("created new project VLAN",
			slog.String("workspace_id", workspaceID),
			slog.String("project_id", projectID),
			slog.Int("vlan_number", vlanNum),
			slog.String("subnet_cidr", subnetCIDR),
		)

		// Save state after creating new allocations
		if err := mbm.saveState(); err != nil {
			mbm.logger.Warn("failed to save state after creating project VLAN",
				slog.String("error", err.Error()),
			)
		}
	}

	return projectVLAN, workspace, nil
}

// calculateVLANSubnet calculates the /27 subnet for a VLAN within a bridge
func (mbm *MultiBridgeManager) calculateVLANSubnet(bridgeNumber, vlanNumber int) string {
	// AIDEV-BUSINESS_RULE: Each bridge gets /24, each VLAN gets /27 within that
	// Bridge subnet: 172.16.{bridge_num}.0/24
	// VLAN subnet: 172.16.{bridge_num}.{vlan_offset}/27
	vlanOffset := (vlanNumber % 8) * 32 // 8 VLANs per /24, each gets /27 (32 IPs)
	return fmt.Sprintf("172.16.%d.%d/27", bridgeNumber, vlanOffset)
}

// GetWorkspaceInfo returns information about a workspace allocation
func (mbm *MultiBridgeManager) GetWorkspaceInfo(workspaceID string) (*WorkspaceAllocation, error) {
	mbm.mu.RLock()
	defer mbm.mu.RUnlock()

	workspace, exists := mbm.workspaces[workspaceID]
	if !exists {
		return nil, fmt.Errorf("workspace %s not found", workspaceID)
	}

	// Return a copy to prevent external modification
	workspaceCopy := &WorkspaceAllocation{
		WorkspaceID:  workspace.WorkspaceID,
		BridgeNumber: workspace.BridgeNumber,
		BridgeName:   workspace.BridgeName,
		ProjectVLANs: make(map[string]*ProjectVLAN),
		CreatedAt:    workspace.CreatedAt,
		VMCount:      workspace.VMCount,
	}

	// Deep copy project VLANs
	for projID, vlan := range workspace.ProjectVLANs {
		workspaceCopy.ProjectVLANs[projID] = &ProjectVLAN{
			ProjectID:   vlan.ProjectID,
			VLANNumber:  vlan.VLANNumber,
			SubnetCIDR:  vlan.SubnetCIDR,
			NextVMIndex: vlan.NextVMIndex,
			CreatedAt:   vlan.CreatedAt,
		}
	}

	return workspaceCopy, nil
}

// GetBridgeUsageStats returns statistics about bridge utilization
func (mbm *MultiBridgeManager) GetBridgeUsageStats() map[int]int {
	mbm.mu.RLock()
	defer mbm.mu.RUnlock()

	stats := make(map[int]int)
	for bridgeNum, workspaceMap := range mbm.bridgeUsage {
		stats[bridgeNum] = len(workspaceMap)
	}

	// Fill in zero values for unused bridges
	for i := 0; i < mbm.bridgeCount; i++ {
		if _, exists := stats[i]; !exists {
			stats[i] = 0
		}
	}

	return stats
}

// ListWorkspaces returns a list of all workspace allocations
func (mbm *MultiBridgeManager) ListWorkspaces() []*WorkspaceAllocation {
	mbm.mu.RLock()
	defer mbm.mu.RUnlock()

	workspaces := make([]*WorkspaceAllocation, 0, len(mbm.workspaces))
	for _, workspace := range mbm.workspaces {
		workspaces = append(workspaces, workspace)
	}

	return workspaces
}
