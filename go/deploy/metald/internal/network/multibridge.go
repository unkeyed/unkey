package network

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"
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

// GetVLANForProject deterministically maps project_id to VLAN number within a bridge
func (mbm *MultiBridgeManager) GetVLANForProject(projectID string) int {
	// AIDEV-BUSINESS_RULE: Use FNV hash for deterministic VLAN assignment
	hash := fnv.New32a()
	hash.Write([]byte(projectID))
	vlanRange := mbm.vlanRangeEnd - mbm.vlanRangeStart + 1
	// #nosec G115 -- Safe conversion: modulo of small VLAN range (typically 100-4000) fits in int
	return int(hash.Sum32()%uint32(vlanRange)) + mbm.vlanRangeStart
}

// GetOrCreateProjectVLAN gets or creates a VLAN for a project within a workspace
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
			CreatedAt:    "2025-08-13T22:00:00Z", // TODO: use proper timestamp
		}
		mbm.workspaces[workspaceID] = workspace

		// Track bridge usage
		if mbm.bridgeUsage[bridgeNum] == nil {
			mbm.bridgeUsage[bridgeNum] = make(map[string]bool)
		}
		mbm.bridgeUsage[bridgeNum][workspaceID] = true
	}

	// Check if project VLAN already exists
	if projectVLAN, exists := workspace.ProjectVLANs[projectID]; exists {
		return projectVLAN, workspace, nil
	}

	// Create new project VLAN
	vlanNumber := mbm.GetVLANForProject(projectID)
	subnetCIDR := mbm.calculateVLANSubnet(workspace.BridgeNumber, vlanNumber)

	projectVLAN := &ProjectVLAN{
		ProjectID:   projectID,
		VLANNumber:  vlanNumber,
		SubnetCIDR:  subnetCIDR,
		NextVMIndex: 0,
		CreatedAt:   "2025-08-13T22:00:00Z", // TODO: use proper timestamp
	}

	workspace.ProjectVLANs[projectID] = projectVLAN
	return projectVLAN, workspace, nil
}

// calculateVLANSubnet calculates the /27 subnet for a VLAN within a bridge
func (mbm *MultiBridgeManager) calculateVLANSubnet(bridgeNumber, vlanNumber int) string {
	// AIDEV-BUSINESS_RULE: Each tenant bridge gets 172.16.{bridge}.0/24, projects get /27 within that
	// /27 = 32 IPs, so 8 project VLANs per tenant bridge (256/32 = 8)
	// Project subnets: .0/27, .32/27, .64/27, .96/27, .128/27, .160/27, .192/27, .224/27

	// Use VLAN number modulo to determine position within tenant bridge subnet
	vlanOffset := (vlanNumber % 8) * 32 // Each /27 has 32 IPs
	return fmt.Sprintf("172.16.%d.%d/27", bridgeNumber, vlanOffset)
}

// AllocateVMIP allocates the next available IP in a project's VLAN
func (mbm *MultiBridgeManager) AllocateVMIP(workspaceID, projectID string) (net.IP, string, error) {
	projectVLAN, workspace, err := mbm.GetOrCreateProjectVLAN(workspaceID, projectID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get project VLAN: %w", err)
	}

	// Parse VLAN subnet
	_, network, err := net.ParseCIDR(projectVLAN.SubnetCIDR)
	if err != nil {
		return nil, "", fmt.Errorf("invalid VLAN subnet %s: %w", projectVLAN.SubnetCIDR, err)
	}

	// Calculate next IP (start from .2 to leave .1 for gateway within /27)
	ip := make(net.IP, len(network.IP))
	copy(ip, network.IP)

	// For /27, we have 30 usable IPs (.2 to .31)
	nextIP := 2 + projectVLAN.NextVMIndex
	if nextIP > 30 {
		return nil, "", fmt.Errorf("project %s VLAN %s is full", projectID, projectVLAN.SubnetCIDR)
	}

	ip[len(ip)-1] += byte(nextIP)

	// Verify IP is within subnet
	if !network.Contains(ip) {
		return nil, "", fmt.Errorf("calculated IP %s is outside VLAN subnet %s",
			ip.String(), projectVLAN.SubnetCIDR)
	}

	// Increment VM index for next allocation
	projectVLAN.NextVMIndex++

	return ip, workspace.BridgeName, nil
}

// GetWorkspaceInfo returns information about a workspace's allocation
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
	}

	for projectID, projectVLAN := range workspace.ProjectVLANs {
		workspaceCopy.ProjectVLANs[projectID] = &ProjectVLAN{
			ProjectID:   projectVLAN.ProjectID,
			VLANNumber:  projectVLAN.VLANNumber,
			SubnetCIDR:  projectVLAN.SubnetCIDR,
			NextVMIndex: projectVLAN.NextVMIndex,
			CreatedAt:   projectVLAN.CreatedAt,
		}
	}

	return workspaceCopy, nil
}

// GetBridgeUsageStats returns usage statistics for all bridges
func (mbm *MultiBridgeManager) GetBridgeUsageStats() map[int]int {
	mbm.mu.RLock()
	defer mbm.mu.RUnlock()

	stats := make(map[int]int)
	for bridgeNum := 0; bridgeNum < mbm.bridgeCount; bridgeNum++ {
		if usage, exists := mbm.bridgeUsage[bridgeNum]; exists {
			stats[bridgeNum] = len(usage)
		} else {
			stats[bridgeNum] = 0
		}
	}
	return stats
}

// ListWorkspaces returns all workspace allocations
func (mbm *MultiBridgeManager) ListWorkspaces() []*WorkspaceAllocation {
	mbm.mu.RLock()
	defer mbm.mu.RUnlock()

	workspaces := make([]*WorkspaceAllocation, 0, len(mbm.workspaces))
	for _, workspace := range mbm.workspaces {
		workspaces = append(workspaces, workspace)
	}
	return workspaces
}

// GenerateTenantMAC generates an OUI-based MAC address for tenant identification
// Format: 02:{bridge_hex}:4B:XX:XX:XX where 4B = "K" for unKey
func (mbm *MultiBridgeManager) GenerateTenantMAC(workspaceID string) (string, error) {
	bridgeNumber := mbm.GetBridgeForWorkspace(workspaceID)

	// AIDEV-BUSINESS_RULE: OUI format for security and identification
	// 02 = locally administered unicast
	// XX = bridge number (tenant identifier)
	// 4B = "K" for unKey (0x4B = 75 = ASCII 'K')
	oui := fmt.Sprintf("02:%02X:4B", bridgeNumber)

	// Generate random device identifier (last 3 bytes)
	deviceBytes := make([]byte, 3)
	if _, err := rand.Read(deviceBytes); err != nil {
		return "", fmt.Errorf("failed to generate random MAC device ID: %w", err)
	}

	deviceID := fmt.Sprintf("%02X:%02X:%02X", deviceBytes[0], deviceBytes[1], deviceBytes[2])

	return fmt.Sprintf("%s:%s", oui, deviceID), nil
}

// GenerateSequentialTenantMAC generates a sequential MAC for predictable allocation
func (mbm *MultiBridgeManager) GenerateSequentialTenantMAC(workspaceID string, vmIndex int) string {
	bridgeNumber := mbm.GetBridgeForWorkspace(workspaceID)

	// AIDEV-BUSINESS_RULE: Sequential MAC assignment within tenant bridge
	// Format: 02:{bridge}:4B:{vm_index_as_3_bytes}
	oui := fmt.Sprintf("02:%02X:4B", bridgeNumber)
	deviceID := fmt.Sprintf("%02X:%02X:%02X",
		(vmIndex>>16)&0xFF, (vmIndex>>8)&0xFF, vmIndex&0xFF)

	return fmt.Sprintf("%s:%s", oui, deviceID)
}

// ParseTenantFromMAC extracts the tenant bridge number from an OUI-based MAC
func ParseTenantFromMAC(macAddr string) (int, error) {
	// Expected format: 02:XX:4B:YY:YY:YY
	if len(macAddr) != 17 {
		return -1, fmt.Errorf("invalid MAC address length: %s", macAddr)
	}

	// Check OUI prefix for unKey format
	if macAddr[:2] != "02" || macAddr[6:8] != "4B" {
		return -1, fmt.Errorf("MAC address is not unKey tenant format: %s", macAddr)
	}

	// Extract bridge number from second byte
	var bridgeNum int
	if _, err := fmt.Sscanf(macAddr[3:5], "%02X", &bridgeNum); err != nil {
		return -1, fmt.Errorf("failed to parse bridge number from MAC: %s", macAddr)
	}

	return bridgeNum, nil
}

// ValidateTenantMAC checks if a MAC address belongs to the expected tenant
func (mbm *MultiBridgeManager) ValidateTenantMAC(workspaceID, macAddr string) error {
	expectedBridge := mbm.GetBridgeForWorkspace(workspaceID)
	actualBridge, err := ParseTenantFromMAC(macAddr)
	if err != nil {
		return fmt.Errorf("invalid tenant MAC format: %w", err)
	}

	if actualBridge != expectedBridge {
		return fmt.Errorf("MAC address bridge %d does not match workspace bridge %d",
			actualBridge, expectedBridge)
	}

	return nil
}

// VerifyBridge verifies that a specific bridge exists and has the expected IP configuration
// This is used during metald startup to ensure bridge infrastructure is ready
func (mbm *MultiBridgeManager) VerifyBridge(bridgeName, expectedIP string) error {
	// Use the existing VerifyBridge function with a minimal config
	logger := slog.Default().With("component", "multi-bridge-verify")

	netConfig := &Config{
		BridgeName: bridgeName,
		BridgeIP:   expectedIP,
	}

	// Use empty NetworkConfig since we're only verifying bridge existence and IP
	mainConfig := &config.NetworkConfig{}

	return VerifyBridge(logger, netConfig, mainConfig)
}

// AllocateIPForWorkspace allocates an IP address for a workspace VM
// Returns the IP, bridge name, and any error
func (mbm *MultiBridgeManager) AllocateIPForWorkspace(workspaceID string) (net.IP, string, error) {
	mbm.mu.Lock()
	defer mbm.mu.Unlock()

	// Get or create workspace allocation
	allocation, exists := mbm.workspaces[workspaceID]
	if !exists {
		// Create new workspace allocation
		bridgeNumber := mbm.GetBridgeForWorkspace(workspaceID)
		bridgeName := fmt.Sprintf("%s-%d", mbm.bridgePrefix, bridgeNumber)

		allocation = &WorkspaceAllocation{
			WorkspaceID:  workspaceID,
			BridgeNumber: bridgeNumber,
			BridgeName:   bridgeName,
			ProjectVLANs: make(map[string]*ProjectVLAN),
			CreatedAt:    time.Now().Format(time.RFC3339),
		}

		mbm.workspaces[workspaceID] = allocation

		// Initialize bridge usage tracking
		if mbm.bridgeUsage[bridgeNumber] == nil {
			mbm.bridgeUsage[bridgeNumber] = make(map[string]bool)
		}
		mbm.bridgeUsage[bridgeNumber][workspaceID] = true
	}

	// For now, allocate directly from bridge subnet (not project-specific VLAN)
	// TODO: Add project_id parameter for project-specific VLAN allocation
	bridgeSubnet := fmt.Sprintf("172.16.%d.0/24", allocation.BridgeNumber)
	_, network, err := net.ParseCIDR(bridgeSubnet)
	if err != nil {
		return nil, "", fmt.Errorf("invalid bridge subnet %s: %w", bridgeSubnet, err)
	}

	// Workspace-based /28 subnet allocation for multi-VM support
	// AIDEV-NOTE: Each workspace gets a /28 subnet (16 IPs) for up to 14 VMs
	// A bridge's /24 space (256 IPs) can hold 16 workspaces × 16 IPs each
	// Workspace subnets: .0/28, .16/28, .32/28, .48/28, etc.
	// Within each /28: .1=reserved, .2-.15=VMs (14 usable IPs)

	// Use deterministic hash to assign /28 subnet to workspace
	hash := fnv.New32a()
	hash.Write([]byte(workspaceID))
	workspaceSubnetIndex := int(hash.Sum32() % 16)   // 0-15 (16 possible /28 subnets per bridge)
	workspaceSubnetBase := workspaceSubnetIndex * 16 // 0, 16, 32, 48, 64, etc.

	vmIP := workspaceSubnetBase + 2 + allocation.VMCount // Start from .2, .3, .4, etc. (.1 reserved)
	allocation.VMCount++                                 // Increment for next allocation

	if vmIP > 254 {
		return nil, "", fmt.Errorf("workspace %s bridge subnet is full", workspaceID)
	}

	// Calculate IP address
	ip := make(net.IP, len(network.IP))
	copy(ip, network.IP)
	ip[len(ip)-1] += byte(vmIP)

	// Verify IP is within subnet
	if !network.Contains(ip) {
		return nil, "", fmt.Errorf("calculated IP %s is outside bridge subnet %s",
			ip.String(), bridgeSubnet)
	}

	// Log successful IP allocation
	mbm.logger.Info("IP allocated for workspace",
		slog.String("workspace_id", workspaceID),
		slog.String("ip", ip.String()),
		slog.String("bridge", allocation.BridgeName),
		slog.Int("vm_count", allocation.VMCount),
	)

	// Save state after successful allocation - rollback on failure
	if err := mbm.saveState(); err != nil {
		// Rollback the allocation
		allocation.VMCount--
		mbm.logger.Error("IP allocation failed due to state persistence error",
			slog.String("workspace_id", workspaceID),
			slog.String("ip", ip.String()),
			slog.String("error", err.Error()),
		)
		return nil, "", fmt.Errorf("failed to persist IP allocation state: %w", err)
	}

	return ip, allocation.BridgeName, nil
}

// ReleaseIPForWorkspace releases an IP address for a workspace VM
// AIDEV-NOTE: CRITICAL FIX - Properly track VM count when releasing IPs
func (mbm *MultiBridgeManager) ReleaseIPForWorkspace(workspaceID string, ip net.IP) error {
	mbm.mu.Lock()
	defer mbm.mu.Unlock()

	allocation, exists := mbm.workspaces[workspaceID]
	if !exists {
		return fmt.Errorf("workspace %s not found", workspaceID)
	}

	// Decrement VM count (but don't go below 0)
	oldVMCount := allocation.VMCount
	if allocation.VMCount > 0 {
		allocation.VMCount--
	}

	// Log IP release
	mbm.logger.Info("IP released for workspace",
		slog.String("workspace_id", workspaceID),
		slog.String("ip", ip.String()),
		slog.Int("old_vm_count", oldVMCount),
		slog.Int("new_vm_count", allocation.VMCount),
	)

	// Save state after successful release - rollback on failure
	if err := mbm.saveState(); err != nil {
		// Rollback the release
		allocation.VMCount++
		mbm.logger.Error("IP release failed due to state persistence error",
			slog.String("workspace_id", workspaceID),
			slog.String("ip", ip.String()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to persist IP release state: %w", err)
	}

	return nil
}

// State persistence methods
// AIDEV-NOTE: CRITICAL FIX - Persist IP allocation state across service restarts

// MultiBridgeState represents the serializable state for persistence
type MultiBridgeState struct {
	Workspaces  map[string]*WorkspaceAllocation `json:"workspaces"`
	BridgeUsage map[int]map[string]bool         `json:"bridge_usage"`
	LastSaved   time.Time                       `json:"last_saved"`
	Checksum    string                          `json:"checksum"` // SHA256 checksum for integrity validation
}

// saveState persists the current state to disk
func (mbm *MultiBridgeManager) saveState() error {
	start := time.Now()

	mbm.logger.Debug("saving state to disk",
		slog.String("state_path", mbm.statePath),
		slog.Int("workspace_count", len(mbm.workspaces)),
	)

	// Create state directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(mbm.statePath), 0755); err != nil {
		mbm.logger.Error("failed to create state directory",
			slog.String("error", err.Error()),
			slog.String("directory", filepath.Dir(mbm.statePath)),
		)
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	state := &MultiBridgeState{
		Workspaces:  mbm.workspaces,
		BridgeUsage: mbm.bridgeUsage,
		LastSaved:   time.Now(),
	}

	// Calculate checksum of state content (excluding checksum field)
	checksum, err := mbm.calculateStateChecksum(state)
	if err != nil {
		mbm.logger.Error("failed to calculate state checksum",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to calculate state checksum: %w", err)
	}
	state.Checksum = checksum

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		mbm.logger.Error("failed to marshal state",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temporary file first, then atomic rename
	tmpPath := mbm.statePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		mbm.logger.Error("failed to write state file",
			slog.String("error", err.Error()),
			slog.String("temp_path", tmpPath),
		)
		return fmt.Errorf("failed to write state file: %w", err)
	}

	if err := os.Rename(tmpPath, mbm.statePath); err != nil {
		mbm.logger.Error("failed to rename state file",
			slog.String("error", err.Error()),
			slog.String("temp_path", tmpPath),
			slog.String("final_path", mbm.statePath),
		)
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	mbm.logger.Debug("state saved successfully",
		slog.String("state_path", mbm.statePath),
		slog.Duration("duration", time.Since(start)),
		slog.Int("data_size_bytes", len(data)),
	)

	return nil
}

// loadState restores state from disk
func (mbm *MultiBridgeManager) loadState() error {
	if _, err := os.Stat(mbm.statePath); os.IsNotExist(err) {
		// No state file exists yet - this is fine for first run
		return nil
	}

	data, err := os.ReadFile(mbm.statePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	var state MultiBridgeState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Verify checksum first to detect corruption
	if err := mbm.verifyStateChecksum(&state); err != nil {
		return fmt.Errorf("state file checksum verification failed: %w", err)
	}

	// Validate state integrity before applying
	if err := mbm.validateState(&state); err != nil {
		return fmt.Errorf("corrupted state file: %w", err)
	}

	// Restore state
	mbm.workspaces = state.Workspaces
	mbm.bridgeUsage = state.BridgeUsage

	// Initialize maps if they're nil
	if mbm.workspaces == nil {
		mbm.workspaces = make(map[string]*WorkspaceAllocation)
	}
	if mbm.bridgeUsage == nil {
		mbm.bridgeUsage = make(map[int]map[string]bool)
	}

	// Initialize nested maps in bridgeUsage
	for bridgeNum := range mbm.bridgeUsage {
		if mbm.bridgeUsage[bridgeNum] == nil {
			mbm.bridgeUsage[bridgeNum] = make(map[string]bool)
		}
	}

	return nil
}

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

// calculateStateChecksum computes SHA256 checksum of state content (excluding checksum field)
func (mbm *MultiBridgeManager) calculateStateChecksum(state *MultiBridgeState) (string, error) {
	// Create a copy of state without the checksum field for consistent hashing
	stateForHashing := &MultiBridgeState{
		Workspaces:  state.Workspaces,
		BridgeUsage: state.BridgeUsage,
		LastSaved:   state.LastSaved,
		// Checksum field is intentionally excluded
	}

	// Marshal to JSON for consistent byte representation
	data, err := json.Marshal(stateForHashing)
	if err != nil {
		return "", fmt.Errorf("failed to marshal state for checksum: %w", err)
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// verifyStateChecksum verifies the integrity of loaded state using stored checksum
func (mbm *MultiBridgeManager) verifyStateChecksum(state *MultiBridgeState) error {
	// Skip verification if no checksum is present (legacy state files)
	if state.Checksum == "" {
		mbm.logger.Warn("state file has no checksum, skipping integrity verification",
			slog.String("state_path", mbm.statePath),
		)
		return nil
	}

	// Calculate expected checksum
	expectedChecksum, err := mbm.calculateStateChecksum(state)
	if err != nil {
		return fmt.Errorf("failed to calculate expected checksum: %w", err)
	}

	// Compare checksums
	if state.Checksum != expectedChecksum {
		mbm.logger.Error("state file integrity check failed",
			slog.String("state_path", mbm.statePath),
			slog.String("stored_checksum", state.Checksum),
			slog.String("calculated_checksum", expectedChecksum),
		)
		return fmt.Errorf("checksum mismatch: stored=%s, calculated=%s", state.Checksum, expectedChecksum)
	}

	mbm.logger.Debug("state file integrity verified",
		slog.String("state_path", mbm.statePath),
		slog.String("checksum", state.Checksum),
	)

	return nil
}
