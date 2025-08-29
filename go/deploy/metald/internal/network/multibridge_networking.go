package network

import (
	"crypto/rand"
	"fmt"
	"hash/fnv"
	"log/slog"
	"net"
	"time"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"
)

// GenerateTenantMAC generates a unique MAC address with OUI-based tenant identification
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

	// Validate bridge allocation before proceeding
	if err := mbm.validateBridgeAllocation(allocation); err != nil {
		return nil, "", fmt.Errorf("bridge allocation validation failed: %w", err)
	}

	// For now, allocate directly from bridge subnet (not project-specific VLAN)
	// TODO: Add project_id parameter for project-specific VLAN allocation
	bridgeSubnet := fmt.Sprintf("172.16.%d.0/24", allocation.BridgeNumber)
	_, network, err := net.ParseCIDR(bridgeSubnet)
	if err != nil {
		return nil, "", fmt.Errorf("invalid bridge subnet %s: %w", bridgeSubnet, err)
	}

	// Workspace-based /29 subnet allocation for multi-VM support
	// AIDEV-NOTE: Each workspace gets a /29 subnet (8 IPs) for up to 5 VMs
	// A bridge's /24 space (256 IPs) can hold 32 workspaces × 8 IPs each
	// Workspace subnets: .0/29, .8/29, .16/29, .24/29, etc.
	// Within each /29: .0=network, .1=gateway, .2-.6=VMs (5 usable IPs), .7=broadcast

	// Use deterministic hash to assign /29 subnet to workspace
	hash := fnv.New32a()
	hash.Write([]byte(workspaceID))
	workspaceSubnetIndex := int(hash.Sum32() % 32)  // 0-31 (32 possible /29 subnets per bridge)
	workspaceSubnetBase := workspaceSubnetIndex * 8 // 0, 8, 16, 24, 32, etc.

	vmIP := workspaceSubnetBase + 2 + allocation.VMCount // Start from .2, .3, .4, etc. (.1 reserved)

	// Check if workspace /29 subnet is full (max 5 VMs: .2, .3, .4, .5, .6)
	if allocation.VMCount >= 5 {
		return nil, "", fmt.Errorf("workspace %s /29 subnet is full (5/5 VMs)", workspaceID)
	}

	// Validate VM IP allocation before proceeding
	if err := mbm.validateVMIPAllocation(workspaceID, workspaceSubnetBase, vmIP, allocation.VMCount); err != nil {
		return nil, "", fmt.Errorf("VM IP allocation validation failed: %w", err)
	}

	allocation.VMCount++ // Increment for next allocation

	// Calculate IP address
	ip := make(net.IP, len(network.IP))
	copy(ip, network.IP)
	ip[len(ip)-1] += byte(vmIP)

	// Verify IP is within subnet
	if !network.Contains(ip) {
		return nil, "", fmt.Errorf("calculated IP %s is outside bridge subnet %s",
			ip.String(), bridgeSubnet)
	}

	// Final validation: ensure IP is within workspace /29 subnet
	if err := mbm.validateIPWithinWorkspaceSubnet(ip, allocation.BridgeNumber, workspaceSubnetBase); err != nil {
		return nil, "", fmt.Errorf("IP subnet validation failed: %w", err)
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

// AllocateVMIP allocates a VM IP within a project VLAN (future enhancement)
func (mbm *MultiBridgeManager) AllocateVMIP(workspaceID, projectID string) (net.IP, string, error) {
	vlan, workspace, err := mbm.GetOrCreateProjectVLAN(workspaceID, projectID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get project VLAN: %w", err)
	}

	// Parse VLAN subnet
	_, vlanNet, err := net.ParseCIDR(vlan.SubnetCIDR)
	if err != nil {
		return nil, "", fmt.Errorf("invalid VLAN subnet %s: %w", vlan.SubnetCIDR, err)
	}

	// Calculate next IP in VLAN
	// Reserve .0 for network, .1 for gateway, start VMs from .2
	vmOffset := vlan.NextVMIndex + 2

	// Check if VLAN subnet is full (/27 has 32 IPs, 30 usable)
	if vmOffset >= 30 {
		return nil, "", fmt.Errorf("project VLAN %s is full", projectID)
	}

	// Create IP address
	ip := make(net.IP, len(vlanNet.IP))
	copy(ip, vlanNet.IP)
	ip[len(ip)-1] += byte(vmOffset)

	vlan.NextVMIndex++

	// Save state after allocation
	if err := mbm.saveState(); err != nil {
		// Rollback
		vlan.NextVMIndex--
		return nil, "", fmt.Errorf("failed to save state after IP allocation: %w", err)
	}

	mbm.logger.Info("allocated project VM IP",
		slog.String("workspace_id", workspaceID),
		slog.String("project_id", projectID),
		slog.String("ip", ip.String()),
		slog.String("vlan_subnet", vlan.SubnetCIDR),
		slog.Int("vm_index", vlan.NextVMIndex-1),
	)

	return ip, workspace.BridgeName, nil
}

// validateBridgeAllocation validates that bridge allocation is within valid range
func (mbm *MultiBridgeManager) validateBridgeAllocation(allocation *WorkspaceAllocation) error {
	if allocation.BridgeNumber < 0 || allocation.BridgeNumber >= mbm.bridgeCount {
		return fmt.Errorf("bridge number %d is outside valid range [0, %d)",
			allocation.BridgeNumber, mbm.bridgeCount)
	}

	expectedBridgeName := fmt.Sprintf("%s-%d", mbm.bridgePrefix, allocation.BridgeNumber)
	if allocation.BridgeName != expectedBridgeName {
		return fmt.Errorf("bridge name mismatch: expected %s, got %s",
			expectedBridgeName, allocation.BridgeName)
	}

	return nil
}

// validateVMIPAllocation validates VM IP allocation within workspace subnet
func (mbm *MultiBridgeManager) validateVMIPAllocation(workspaceID string, workspaceSubnetBase, vmIP, vmCount int) error {
	// Validate workspace subnet base (must be multiple of 8)
	if workspaceSubnetBase%8 != 0 {
		return fmt.Errorf("workspace subnet base %d is not aligned to /29 boundary",
			workspaceSubnetBase)
	}

	// Validate VM IP is within workspace /29 subnet
	expectedMinIP := workspaceSubnetBase + 2 // .2 is first VM IP
	expectedMaxIP := workspaceSubnetBase + 6 // .6 is last VM IP (.7 is broadcast)

	if vmIP < expectedMinIP || vmIP > expectedMaxIP {
		return fmt.Errorf("VM IP offset %d is outside workspace /29 range [%d, %d]",
			vmIP, expectedMinIP, expectedMaxIP)
	}

	// Validate VM count consistency
	expectedVMIP := workspaceSubnetBase + 2 + vmCount
	if vmIP != expectedVMIP {
		return fmt.Errorf("VM IP offset %d does not match expected offset %d for VM count %d",
			vmIP, expectedVMIP, vmCount)
	}

	return nil
}

// validateIPWithinWorkspaceSubnet validates that an IP is within the expected workspace /29 subnet
func (mbm *MultiBridgeManager) validateIPWithinWorkspaceSubnet(ip net.IP, bridgeNumber, workspaceSubnetBase int) error {
	// Calculate expected /29 subnet for workspace
	expectedSubnetIP := net.IPv4(172, 16, byte(bridgeNumber), byte(workspaceSubnetBase))
	expectedSubnet := &net.IPNet{
		IP:   expectedSubnetIP,
		Mask: net.CIDRMask(29, 32), // /29 netmask
	}

	if !expectedSubnet.Contains(ip) {
		return fmt.Errorf("IP %s is not within workspace /29 subnet %s",
			ip.String(), expectedSubnet.String())
	}

	// Additional validation: IP should not be network (.0) or gateway (.1) addresses
	lastOctet := ip[3]
	subnetOffset := lastOctet - byte(workspaceSubnetBase)

	if subnetOffset <= 1 {
		return fmt.Errorf("IP %s uses reserved address in /29 subnet (offset %d)",
			ip.String(), subnetOffset)
	}

	if subnetOffset >= 7 {
		return fmt.Errorf("IP %s uses broadcast address in /29 subnet (offset %d)",
			ip.String(), subnetOffset)
	}

	return nil
}
