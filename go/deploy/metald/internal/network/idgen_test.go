package network

import (
	"testing"
)

func TestIDGenerator(t *testing.T) {
	gen := NewIDGenerator()

	// Test generating IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := gen.GenerateNetworkID()
		if err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}

		// Check length
		if len(id) != 8 {
			t.Errorf("Expected ID length 8, got %d", len(id))
		}

		// Check uniqueness
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}

	// Test release and reuse
	firstID, _ := gen.GenerateNetworkID()
	gen.ReleaseID(firstID)

	// The same ID could be generated again after release
	// (though not guaranteed due to randomness)
}

func TestGenerateDeviceNames(t *testing.T) {
	networkID := "a1b2c3d4"
	names := GenerateDeviceNames(networkID)

	tests := []struct {
		name   string
		got    string
		maxLen int
	}{
		{"TAP device", names.TAP, 15},
		{"Veth host", names.VethHost, 15},
		{"Veth NS", names.VethNS, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.got) > tt.maxLen {
				t.Errorf("%s name too long: %s (%d chars, max %d)",
					tt.name, tt.got, len(tt.got), tt.maxLen)
			}

			// Check it contains the network ID
			if len(tt.got) < len(networkID) {
				t.Errorf("%s name doesn't contain full network ID: %s", tt.name, tt.got)
			}
		})
	}

	// Verify expected formats
	if names.TAP != "tap_a1b2c3d4" {
		t.Errorf("Expected TAP name 'tap_a1b2c3d4', got %s", names.TAP)
	}
	if names.VethHost != "vh_a1b2c3d4" {
		t.Errorf("Expected VethHost name 'vh_a1b2c3d4', got %s", names.VethHost)
	}
	if names.VethNS != "vn_a1b2c3d4" {
		t.Errorf("Expected VethNS name 'vn_a1b2c3d4', got %s", names.VethNS)
	}
	if names.Namespace != "ns_vm_a1b2c3d4" {
		t.Errorf("Expected Namespace name 'ns_vm_a1b2c3d4', got %s", names.Namespace)
	}
}
