package network

import (
	"fmt"
	"hash/fnv"
	"net"
	"os/exec"
	"sync"
	"time"

	"github.com/vishvananda/netlink"
)

// validateVLANID validates that a VLAN ID is within safe bounds to prevent command injection
func validateVLANID(vlanID int) error {
	// VLAN IDs must be between 1-4094 per IEEE 802.1Q standard
	// 0 is reserved, 4095 is reserved
	if vlanID < 1 || vlanID > 4094 {
		return fmt.Errorf("invalid VLAN ID %d: must be between 1-4094", vlanID)
	}
	return nil
}

// validateNetworkDeviceName validates network interface/bridge names to prevent command injection
func validateNetworkDeviceName(name string) error {
	if name == "" {
		return fmt.Errorf("network device name cannot be empty")
	}

	// Network device names in Linux have specific constraints:
	// - Maximum 15 characters (IFNAMSIZ - 1)
	// - Can contain alphanumeric, hyphens, underscores, dots
	// - Cannot start with dot or contain spaces/special chars that could be shell metacharacters
	if len(name) > 15 {
		return fmt.Errorf("network device name '%s' too long: maximum 15 characters", name)
	}

	// Only allow safe characters: alphanumeric, hyphens, underscores, dots
	// This prevents injection of shell metacharacters like $, `, ;, |, &, etc.
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.') {
			return fmt.Errorf("network device name '%s' contains invalid character '%c': only alphanumeric, hyphens, underscores, and dots allowed", name, r)
		}
	}

	// Cannot start with dot (hidden files/relative paths)
	if name[0] == '.' {
		return fmt.Errorf("network device name '%s' cannot start with dot", name)
	}

	return nil
}

// AIDEV-NOTE: Just-in-time workspace VLAN provisioning
// Creates VLANs on-demand when VMs are requested for a workspace

// WorkspaceVLAN represents a workspace's network configuration
type WorkspaceVLAN struct {
	WorkspaceID string `json:"workspace_id"`
	VLANBase    int    `json:"vlan_base"`
	SubnetCIDR  string `json:"subnet_cidr"`
	NextVMIndex int    `json:"next_vm_index"`
	CreatedAt   string `json:"created_at"`
}

// WorkspaceManager manages workspace VLANs dynamically
type WorkspaceManager struct {
	bridgeName     string
	workspaceVLANs map[string]*WorkspaceVLAN
	usedVLANs      map[int]string // VLAN ID -> workspace ID mapping
	mu             sync.RWMutex
	vlanRangeStart int // Starting VLAN ID for workspaces
	vlanRangeEnd   int // Ending VLAN ID for workspaces
}

// NewWorkspaceManager creates a new workspace VLAN manager
func NewWorkspaceManager(bridgeName string) *WorkspaceManager {
	return &WorkspaceManager{
		bridgeName:     bridgeName,
		workspaceVLANs: make(map[string]*WorkspaceVLAN),
		usedVLANs:      make(map[int]string),
		vlanRangeStart: 100, // VLANs 100-4000 for workspaces
		vlanRangeEnd:   4000,
	}
}

// GetOrCreateWorkspaceVLAN gets existing workspace VLAN or creates a new one
func (wm *WorkspaceManager) GetOrCreateWorkspaceVLAN(workspaceID string) (*WorkspaceVLAN, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Check if workspace VLAN already exists
	if vlan, exists := wm.workspaceVLANs[workspaceID]; exists {
		return vlan, nil
	}

	// Create new workspace VLAN
	return wm.createWorkspaceVLAN(workspaceID)
}

// createWorkspaceVLAN creates a new VLAN for a workspace
func (wm *WorkspaceManager) createWorkspaceVLAN(workspaceID string) (*WorkspaceVLAN, error) {
	// Generate deterministic VLAN ID based on workspace ID
	vlanBase := wm.hashToVLAN(workspaceID)

	// Check for VLAN conflicts and find next available
	attempts := 0
	maxAttempts := 100 // Prevent infinite loops

	for attempts < maxAttempts {
		if conflictWorkspace, exists := wm.usedVLANs[vlanBase]; !exists {
			// VLAN is available
			break
		} else if conflictWorkspace == workspaceID {
			// Same workspace already has this VLAN (shouldn't happen but handle it)
			return wm.workspaceVLANs[workspaceID], nil
		} else {
			// VLAN conflict, try next one
			vlanBase++
			if vlanBase > wm.vlanRangeEnd {
				vlanBase = wm.vlanRangeStart // Wrap around
			}
			attempts++
		}
	}

	if attempts >= maxAttempts {
		return nil, fmt.Errorf("failed to find available VLAN for workspace %s after %d attempts", workspaceID, maxAttempts)
	}

	// Generate subnet for this workspace
	subnetCIDR := wm.workspaceToSubnet(vlanBase)

	// Create workspace VLAN configuration
	workspaceVLAN := &WorkspaceVLAN{
		WorkspaceID: workspaceID,
		VLANBase:    vlanBase,
		SubnetCIDR:  subnetCIDR,
		NextVMIndex: 0,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	// Configure VLAN on bridge
	if err := wm.configureWorkspaceVLAN(vlanBase); err != nil {
		return nil, fmt.Errorf("failed to configure VLAN %d for workspace %s: %w", vlanBase, workspaceID, err)
	}

	// Store the mapping
	wm.workspaceVLANs[workspaceID] = workspaceVLAN
	wm.usedVLANs[vlanBase] = workspaceID

	return workspaceVLAN, nil
}

// hashToVLAN generates a deterministic VLAN ID from workspace ID
func (wm *WorkspaceManager) hashToVLAN(workspaceID string) int {
	hash := fnv.New32a()
	hash.Write([]byte(workspaceID))
	// Map to VLAN range, ensure it's within bounds
	vlanRange := wm.vlanRangeEnd - wm.vlanRangeStart + 1
	// #nosec G115 -- Safe conversion: modulo of small VLAN range (typically 100-4000) fits in int
	return int(hash.Sum32()%uint32(vlanRange)) + wm.vlanRangeStart
}

// workspaceToSubnet generates a consistent subnet for a workspace
func (wm *WorkspaceManager) workspaceToSubnet(vlanID int) string {
	// Use VLAN ID to determine subnet to avoid conflicts
	// Map VLAN 100-4000 to subnets across multiple /16 blocks
	// AIDEV-BUSINESS_RULE: Ensure subnet octets stay within valid range (0-255)

	// Calculate which /16 block and subnet within that block
	vlanOffset := vlanID - wm.vlanRangeStart // 0-based offset from start

	// Use 172.31.x.0/24 for first 255 subnets (VLANs 100-354)
	if vlanOffset < 255 {
		subnetOctet := vlanOffset + 1 // Start at .1.0
		return fmt.Sprintf("172.31.%d.0/24", subnetOctet)
	}

	// For higher VLAN IDs, use 172.30.x.0/24 but ensure octet doesn't exceed 255
	secondBlockOffset := vlanOffset - 255
	if secondBlockOffset >= 255 {
		// If we still exceed 255, use 172.29.x.0/24
		thirdBlockOffset := secondBlockOffset - 255
		if thirdBlockOffset >= 255 {
			// Shouldn't happen with current VLAN range, but prevent overflow
			thirdBlockOffset = thirdBlockOffset % 255
		}
		return fmt.Sprintf("172.29.%d.0/24", thirdBlockOffset+1)
	}

	return fmt.Sprintf("172.30.%d.0/24", secondBlockOffset+1)
}

// configureWorkspaceVLAN configures the VLAN on the bridge
func (wm *WorkspaceManager) configureWorkspaceVLAN(vlanID int) error {
	// Validate inputs to prevent command injection
	if err := validateVLANID(vlanID); err != nil {
		return fmt.Errorf("invalid VLAN ID: %w", err)
	}
	if err := validateNetworkDeviceName(wm.bridgeName); err != nil {
		return fmt.Errorf("invalid bridge name: %w", err)
	}

	// Verify bridge exists
	_, err := netlink.LinkByName(wm.bridgeName)
	if err != nil {
		return fmt.Errorf("bridge %s not found: %w", wm.bridgeName, err)
	}

	// Add VLAN to bridge (self)
	// Execute bridge command to configure VLAN (inputs validated above)
	// #nosec G204 -- VLAN ID and bridge name validated above to prevent command injection
	cmd := exec.Command("bridge", "vlan", "add", "vid", fmt.Sprintf("%d", vlanID), "dev", wm.bridgeName, "self")

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to configure VLAN %d on bridge %s: %w (output: %s)",
			vlanID, wm.bridgeName, err, string(output))
	}

	return nil
}

// AllocateWorkspaceIP allocates the next available IP in the workspace subnet
func (wm *WorkspaceManager) AllocateWorkspaceIP(workspaceID string) (net.IP, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	workspaceVLAN, exists := wm.workspaceVLANs[workspaceID]
	if !exists {
		return nil, fmt.Errorf("workspace %s has no VLAN configured", workspaceID)
	}

	// Parse workspace subnet
	_, network, err := net.ParseCIDR(workspaceVLAN.SubnetCIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace subnet %s: %w", workspaceVLAN.SubnetCIDR, err)
	}

	// Calculate next IP (start from .10 to leave room for gateway at .1)
	ip := make(net.IP, len(network.IP))
	copy(ip, network.IP)

	// Add VM index to last octet
	nextIP := 10 + workspaceVLAN.NextVMIndex
	if nextIP > 254 {
		return nil, fmt.Errorf("workspace %s subnet %s is full", workspaceID, workspaceVLAN.SubnetCIDR)
	}

	ip[len(ip)-1] += byte(nextIP)

	// Verify IP is within subnet
	if !network.Contains(ip) {
		return nil, fmt.Errorf("calculated IP %s is outside workspace subnet %s",
			ip.String(), workspaceVLAN.SubnetCIDR)
	}

	// Increment VM index for next allocation
	workspaceVLAN.NextVMIndex++

	return ip, nil
}

// AttachVMToWorkspaceVLAN attaches a VM interface to the workspace VLAN
func (wm *WorkspaceManager) AttachVMToWorkspaceVLAN(workspaceID, interfaceName string) error {
	// Validate inputs to prevent command injection
	if err := validateNetworkDeviceName(interfaceName); err != nil {
		return fmt.Errorf("invalid interface name: %w", err)
	}

	wm.mu.RLock()
	workspaceVLAN, exists := wm.workspaceVLANs[workspaceID]
	wm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("workspace %s has no VLAN configured", workspaceID)
	}

	// Validate VLAN ID from workspace configuration
	if err := validateVLANID(workspaceVLAN.VLANBase); err != nil {
		return fmt.Errorf("invalid workspace VLAN ID: %w", err)
	}

	// Get bridge and VM interface
	bridge, err := netlink.LinkByName(wm.bridgeName)
	if err != nil {
		return fmt.Errorf("bridge %s not found: %w", wm.bridgeName, err)
	}

	vmInterface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("VM interface %s not found: %w", interfaceName, err)
	}

	// Attach interface to bridge
	if err := netlink.LinkSetMaster(vmInterface, bridge); err != nil {
		return fmt.Errorf("failed to attach interface to bridge: %w", err)
	}

	// Configure VLAN on the interface
	// Execute bridge command to configure VLAN on VM interface (inputs validated above)
	// #nosec G204 -- VLAN ID and interface name validated above to prevent command injection
	cmd := exec.Command("bridge", "vlan", "add", "vid", fmt.Sprintf("%d", workspaceVLAN.VLANBase),
		"dev", interfaceName, "untagged", "pvid")

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to configure VLAN %d on interface %s: %w (output: %s)",
			workspaceVLAN.VLANBase, interfaceName, err, string(output))
	}

	return nil
}

// GetWorkspaceInfo returns information about a workspace's network configuration
func (wm *WorkspaceManager) GetWorkspaceInfo(workspaceID string) (*WorkspaceVLAN, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	workspaceVLAN, exists := wm.workspaceVLANs[workspaceID]
	if !exists {
		return nil, fmt.Errorf("workspace %s not found", workspaceID)
	}

	// Return a copy to prevent external modification
	return &WorkspaceVLAN{
		WorkspaceID: workspaceVLAN.WorkspaceID,
		VLANBase:    workspaceVLAN.VLANBase,
		SubnetCIDR:  workspaceVLAN.SubnetCIDR,
		NextVMIndex: workspaceVLAN.NextVMIndex,
		CreatedAt:   workspaceVLAN.CreatedAt,
	}, nil
}

// ListWorkspaces returns all configured workspace VLANs
func (wm *WorkspaceManager) ListWorkspaces() []*WorkspaceVLAN {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	workspaces := make([]*WorkspaceVLAN, 0, len(wm.workspaceVLANs))
	for _, workspace := range wm.workspaceVLANs {
		workspaces = append(workspaces, &WorkspaceVLAN{
			WorkspaceID: workspace.WorkspaceID,
			VLANBase:    workspace.VLANBase,
			SubnetCIDR:  workspace.SubnetCIDR,
			NextVMIndex: workspace.NextVMIndex,
			CreatedAt:   workspace.CreatedAt,
		})
	}

	return workspaces
}

// CleanupWorkspace removes a workspace and its VLAN configuration
func (wm *WorkspaceManager) CleanupWorkspace(workspaceID string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	workspaceVLAN, exists := wm.workspaceVLANs[workspaceID]
	if !exists {
		return fmt.Errorf("workspace %s not found", workspaceID)
	}

	// Validate inputs to prevent command injection
	if err := validateVLANID(workspaceVLAN.VLANBase); err != nil {
		return fmt.Errorf("invalid workspace VLAN ID: %w", err)
	}
	if err := validateNetworkDeviceName(wm.bridgeName); err != nil {
		return fmt.Errorf("invalid bridge name: %w", err)
	}

	// Remove VLAN from bridge (inputs validated above)
	// #nosec G204 -- VLAN ID and bridge name validated above to prevent command injection
	cmd := exec.Command("bridge", "vlan", "del", "vid", fmt.Sprintf("%d", workspaceVLAN.VLANBase),
		"dev", wm.bridgeName, "self")

	if output, err := cmd.CombinedOutput(); err != nil {
		// Log error but don't fail cleanup - VLAN might already be gone
		fmt.Printf("Warning: failed to remove VLAN %d from bridge %s: %v (output: %s)\n",
			workspaceVLAN.VLANBase, wm.bridgeName, err, string(output))
	}

	// Remove from mappings
	delete(wm.workspaceVLANs, workspaceID)
	delete(wm.usedVLANs, workspaceVLAN.VLANBase)

	return nil
}
