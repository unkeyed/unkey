package network

import (
	"log/slog"
	"net"
	"os"
	"testing"
)

// testLogger returns a logger for tests that only shows errors
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestWorkspaceIdToBridgeMapping(t *testing.T) {
	// Test 8-bridge configuration
	mbm8 := NewMultiBridgeManager(8, "br-vms", testLogger())

	// Test that same workspace always maps to same bridge
	workspaceID := "workspace_id125ch235"
	bridge1 := mbm8.GetBridgeForWorkspace(workspaceID)
	bridge2 := mbm8.GetBridgeForWorkspace(workspaceID)

	if bridge1 != bridge2 {
		t.Errorf("Workspace mapping not deterministic: got %d and %d", bridge1, bridge2)
	}

	if bridge1 < 0 || bridge1 >= 8 {
		t.Errorf("Bridge number out of range for 8-bridge config: got %d", bridge1)
	}

	// Test 32-bridge configuration
	mbm32 := NewMultiBridgeManager(32, "br-vms", testLogger())
	bridge32 := mbm32.GetBridgeForWorkspace(workspaceID)

	if bridge32 < 0 || bridge32 >= 32 {
		t.Errorf("Bridge number out of range for 32-bridge config: got %d", bridge32)
	}

	t.Logf("Workspace '%s' maps to bridge %d (8-bridge) and bridge %d (32-bridge)",
		workspaceID, bridge1, bridge32)
}

func TestProjectIdToVLANMapping(t *testing.T) {
	mbm := NewMultiBridgeManager(8, "br-vms", testLogger())

	projectID := "github.com/unkeyed/unkey"
	vlan1 := mbm.GetVLANForProject(projectID)
	vlan2 := mbm.GetVLANForProject(projectID)

	if vlan1 != vlan2 {
		t.Errorf("Project VLAN mapping not deterministic: got %d and %d", vlan1, vlan2)
	}

	if vlan1 < 100 || vlan1 > 4000 {
		t.Errorf("VLAN number out of range: got %d", vlan1)
	}

	t.Logf("Project '%s' maps to VLAN %d", projectID, vlan1)
}

func TestVLANSubnetCalculation(t *testing.T) {
	mbm := NewMultiBridgeManager(8, "br-vms", testLogger())

	testCases := []struct {
		bridgeNumber int
		vlanNumber   int
		expectedCIDR string
	}{
		{0, 100, "172.16.0.128/27"}, // VLAN 100 % 8 = 4, so offset 4*32=128
		{1, 101, "172.16.1.160/27"}, // VLAN 101 % 8 = 5, so offset 5*32=160
		{7, 200, "172.16.7.0/27"},   // VLAN 200 % 8 = 0, so offset 0*32=0
	}

	for _, tc := range testCases {
		result := mbm.calculateVLANSubnet(tc.bridgeNumber, tc.vlanNumber)
		if result != tc.expectedCIDR {
			t.Errorf("Bridge %d, VLAN %d: expected %s, got %s",
				tc.bridgeNumber, tc.vlanNumber, tc.expectedCIDR, result)
		}
	}
}

func TestVMIPAllocation(t *testing.T) {
	mbm := NewMultiBridgeManager(8, "br-vms", testLogger())

	workspaceID := "test-workspace"
	projectID := "test-project"

	// Allocate first IP
	ip1, bridgeName1, err := mbm.AllocateVMIP(workspaceID, projectID)
	if err != nil {
		t.Fatalf("Failed to allocate first IP: %v", err)
	}

	// Allocate second IP
	ip2, bridgeName2, err := mbm.AllocateVMIP(workspaceID, projectID)
	if err != nil {
		t.Fatalf("Failed to allocate second IP: %v", err)
	}

	// Should be on same bridge
	if bridgeName1 != bridgeName2 {
		t.Errorf("IPs allocated on different bridges: %s vs %s", bridgeName1, bridgeName2)
	}

	// IPs should be different
	if ip1.Equal(ip2) {
		t.Errorf("Allocated same IP twice: %s", ip1.String())
	}

	// Should be sequential within the /27 subnet
	// First IP should be subnet_base + 2, second should be subnet_base + 3
	workspace, _ := mbm.GetWorkspaceInfo(workspaceID)
	projectVLAN := workspace.ProjectVLANs[projectID]
	_, network, _ := net.ParseCIDR(projectVLAN.SubnetCIDR)

	expectedIP1 := make(net.IP, len(network.IP))
	copy(expectedIP1, network.IP)
	expectedIP1[3] += 2 // First usable IP in /27

	expectedIP2 := make(net.IP, len(network.IP))
	copy(expectedIP2, network.IP)
	expectedIP2[3] += 3 // Second usable IP in /27

	if !ip1.Equal(expectedIP1) {
		t.Errorf("First IP should be %s, got %s", expectedIP1, ip1)
	}

	if !ip2.Equal(expectedIP2) {
		t.Errorf("Second IP should be %s, got %s", expectedIP2, ip2)
	}

	t.Logf("Allocated IPs: %s and %s on bridge %s", ip1, ip2, bridgeName1)
}

func TestWorkspaceInfoRetrieval(t *testing.T) {
	mbm := NewMultiBridgeManager(8, "br-vms", testLogger())

	workspaceID := "test-workspace"
	projectID := "github.com/unkeyed/unkey"

	// Allocate an IP to create the workspace
	_, _, err := mbm.AllocateVMIP(workspaceID, projectID)
	if err != nil {
		t.Fatalf("Failed to allocate IP: %v", err)
	}

	// Get workspace info
	workspace, err := mbm.GetWorkspaceInfo(workspaceID)
	if err != nil {
		t.Fatalf("Failed to get workspace info: %v", err)
	}

	if workspace.WorkspaceID != workspaceID {
		t.Errorf("Wrong workspace ID: expected %s, got %s", workspaceID, workspace.WorkspaceID)
	}

	if len(workspace.ProjectVLANs) != 1 {
		t.Errorf("Expected 1 project VLAN, got %d", len(workspace.ProjectVLANs))
	}

	projectVLAN, exists := workspace.ProjectVLANs[projectID]
	if !exists {
		t.Errorf("Project VLAN not found for %s", projectID)
	}

	if projectVLAN.NextVMIndex != 1 {
		t.Errorf("Expected NextVMIndex=1, got %d", projectVLAN.NextVMIndex)
	}

	t.Logf("Workspace info: Bridge %d (%s), Project VLAN %d (%s)",
		workspace.BridgeNumber, workspace.BridgeName,
		projectVLAN.VLANNumber, projectVLAN.SubnetCIDR)
}

func TestOUIBasedMACGeneration(t *testing.T) {
	mbm := NewMultiBridgeManager(32, "br-tenant", testLogger())

	testCases := []struct {
		workspaceID    string
		expectedBridge int
		expectedOUI    string
	}{
		{"workspace_id125ch235", 15, "02:0F:4B"}, // Bridge 15 = 0x0F
		{"dev_workspace", 10, "02:0A:4B"},        // Bridge 10 = 0x0A
		{"prod_workspace", 18, "02:12:4B"},       // Bridge 18 = 0x12
	}

	for _, tc := range testCases {
		// Test random MAC generation
		mac, err := mbm.GenerateTenantMAC(tc.workspaceID)
		if err != nil {
			t.Fatalf("Failed to generate MAC for %s: %v", tc.workspaceID, err)
		}

		// Verify OUI format
		if !containsOUI(mac, tc.expectedOUI) {
			t.Errorf("MAC %s does not contain expected OUI %s", mac, tc.expectedOUI)
		}

		// Test sequential MAC generation
		seqMAC := mbm.GenerateSequentialTenantMAC(tc.workspaceID, 42)
		expectedSeqMAC := tc.expectedOUI + ":00:00:2A" // 42 = 0x2A
		if seqMAC != expectedSeqMAC {
			t.Errorf("Sequential MAC: expected %s, got %s", expectedSeqMAC, seqMAC)
		}

		// Test bridge parsing
		parsedBridge, err := ParseTenantFromMAC(mac)
		if err != nil {
			t.Errorf("Failed to parse bridge from MAC %s: %v", mac, err)
		}
		if parsedBridge != tc.expectedBridge {
			t.Errorf("Parsed bridge %d, expected %d", parsedBridge, tc.expectedBridge)
		}

		// Test MAC validation
		if err := mbm.ValidateTenantMAC(tc.workspaceID, mac); err != nil {
			t.Errorf("MAC validation failed for %s: %v", tc.workspaceID, err)
		}

		t.Logf("Workspace '%s' → Bridge %d → OUI %s → MAC %s",
			tc.workspaceID, tc.expectedBridge, tc.expectedOUI, mac)
	}
}

func TestMACSecurityValidation(t *testing.T) {
	mbm := NewMultiBridgeManager(8, "br-tenant", testLogger())

	// Test invalid MAC formats
	invalidMACs := []string{
		"01:05:4B:12:34:56", // Wrong first byte (not locally administered)
		"02:05:4C:12:34:56", // Wrong third byte (not 4B)
		"02:05:4B:12:34",    // Too short
		"not-a-mac-address", // Invalid format
	}

	for _, invalidMAC := range invalidMACs {
		_, err := ParseTenantFromMAC(invalidMAC)
		if err == nil {
			t.Errorf("Expected error for invalid MAC %s, but got none", invalidMAC)
		}
	}

	// Test cross-tenant MAC validation
	workspaceA := "tenant-a"
	workspaceB := "tenant-b"

	macA, _ := mbm.GenerateTenantMAC(workspaceA)

	// Validate correct tenant
	if err := mbm.ValidateTenantMAC(workspaceA, macA); err != nil {
		t.Errorf("Valid MAC failed validation: %v", err)
	}

	// Validate wrong tenant (should fail)
	if err := mbm.ValidateTenantMAC(workspaceB, macA); err == nil {
		t.Errorf("Expected cross-tenant MAC validation to fail, but it passed")
	}
}

func TestMACDeterministicBehavior(t *testing.T) {
	mbm := NewMultiBridgeManager(8, "br-tenant", testLogger())

	workspaceID := "test-workspace"

	// Generate multiple MACs for same workspace - should have same OUI
	mac1, _ := mbm.GenerateTenantMAC(workspaceID)
	mac2, _ := mbm.GenerateTenantMAC(workspaceID)

	bridge1, _ := ParseTenantFromMAC(mac1)
	bridge2, _ := ParseTenantFromMAC(mac2)

	if bridge1 != bridge2 {
		t.Errorf("Same workspace mapped to different bridges: %d vs %d", bridge1, bridge2)
	}

	// Sequential MACs should be deterministic
	seqMAC1 := mbm.GenerateSequentialTenantMAC(workspaceID, 1)
	seqMAC2 := mbm.GenerateSequentialTenantMAC(workspaceID, 1)

	if seqMAC1 != seqMAC2 {
		t.Errorf("Sequential MAC generation not deterministic: %s vs %s", seqMAC1, seqMAC2)
	}

	t.Logf("Workspace '%s' consistently maps to bridge %d", workspaceID, bridge1)
}

// Helper function to check if MAC contains expected OUI
func containsOUI(mac, expectedOUI string) bool {
	if len(mac) < len(expectedOUI) {
		return false
	}
	return mac[:len(expectedOUI)] == expectedOUI
}
