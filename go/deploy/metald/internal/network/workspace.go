package network

import (
	"fmt"
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
