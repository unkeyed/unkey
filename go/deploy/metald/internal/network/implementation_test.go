package network

import (
	"net"
	"testing"
)

// AIDEV-NOTE: Focused IP allocation resource cleanup tests
// These tests verify that IP address allocation and cleanup work correctly
// without requiring network infrastructure or root privileges
// Port allocation tests exist in port_allocator_test.go

// TestIPAllocatorResourceCleanup tests that IP addresses are properly allocated and released
func TestIPAllocatorResourceCleanup(t *testing.T) {
	// Create IP allocator with test subnet
	_, subnet, err := net.ParseCIDR("192.168.100.0/24")
	if err != nil {
		t.Fatalf("Failed to parse subnet: %v", err)
	}
	allocator := NewIPAllocator(subnet)

	// Get initial state
	initialCount := allocator.GetAllocatedCount()
	if initialCount != 0 {
		t.Errorf("Expected 0 initial allocations, got %d", initialCount)
	}

	// Allocate some IPs and assign to VMs
	vmIDs := []string{"vm1", "vm2", "vm3", "vm4", "vm5"}
	var allocatedIPs []net.IP

	for _, vmID := range vmIDs {
		ip, err := allocator.AllocateIP()
		if err != nil {
			t.Errorf("Failed to allocate IP: %v", err)
			continue
		}
		allocator.AssignIPToVM(vmID, ip)
		allocatedIPs = append(allocatedIPs, ip)
		t.Logf("Allocated IP %s to VM %s", ip.String(), vmID)
	}

	// Verify allocation count
	afterAllocation := allocator.GetAllocatedCount()
	expectedAllocated := len(allocatedIPs)
	if afterAllocation != expectedAllocated {
		t.Errorf("Expected %d allocations, got %d", expectedAllocated, afterAllocation)
	}

	// Verify each IP is unique
	ipSet := make(map[string]bool)
	for _, ip := range allocatedIPs {
		ipStr := ip.String()
		if ipSet[ipStr] {
			t.Errorf("Duplicate IP allocated: %s", ipStr)
		}
		ipSet[ipStr] = true
	}

	// Verify VM-IP assignments work
	for i, vmID := range vmIDs {
		assignedIP, exists := allocator.GetVMIP(vmID)
		if !exists {
			t.Errorf("VM %s not found in allocator", vmID)
			continue
		}
		if !assignedIP.Equal(allocatedIPs[i]) {
			t.Errorf("VM %s has wrong IP: got %s, expected %s", 
				vmID, assignedIP.String(), allocatedIPs[i].String())
		}
	}

	// Release individual IPs (first 3)
	for i := 0; i < 3; i++ {
		ip := allocatedIPs[i]
		allocator.ReleaseIP(ip)
		t.Logf("Released IP %s", ip.String())
	}

	// Verify count after partial release
	afterPartialRelease := allocator.GetAllocatedCount()
	expectedRemaining := len(vmIDs) - 3
	if afterPartialRelease != expectedRemaining {
		t.Errorf("Expected %d remaining allocations, got %d", expectedRemaining, afterPartialRelease)
	}

	// Release remaining IPs
	for i := 3; i < len(allocatedIPs); i++ {
		allocator.ReleaseIP(allocatedIPs[i])
	}

	// Verify all IPs are released
	finalCount := allocator.GetAllocatedCount()
	if finalCount != initialCount {
		t.Errorf("Resource leak detected: started with %d, ended with %d", initialCount, finalCount)
	}

	t.Logf("Successfully allocated and released %d IP addresses", len(vmIDs))
}

// TestIPAllocatorEdgeCases tests various edge cases and error conditions
func TestIPAllocatorEdgeCases(t *testing.T) {
	t.Run("DoubleRelease", func(t *testing.T) {
		_, subnet, err := net.ParseCIDR("192.168.200.0/24")
		if err != nil {
			t.Fatalf("Failed to parse subnet: %v", err)
		}
		allocator := NewIPAllocator(subnet)
		
		// Allocate IP
		ip, err := allocator.AllocateIP()
		if err != nil {
			t.Fatalf("Failed to allocate IP: %v", err)
		}
		
		// Release once (should succeed)
		allocator.ReleaseIP(ip)
		
		// Release again (should handle gracefully)
		allocator.ReleaseIP(ip) // This might be a no-op, which is fine
		
		// Verify clean state
		if count := allocator.GetAllocatedCount(); count != 0 {
			t.Errorf("Expected 0 allocations after cleanup, got %d", count)
		}
		
		t.Logf("IP %s allocated and cleaned up successfully", ip.String())
	})

	t.Run("MultipleAllocationCleanup", func(t *testing.T) {
		// Test allocation and cleanup of multiple IPs  
		_, subnet, err := net.ParseCIDR("192.168.250.0/24") // 254 usable IPs
		if err != nil {
			t.Fatalf("Failed to parse subnet: %v", err)
		}
		allocator := NewIPAllocator(subnet)
		
		var allocatedIPs []net.IP
		
		// Allocate several IPs to test cleanup
		for i := 0; i < 10; i++ {
			ip, err := allocator.AllocateIP()
			if err != nil {
				t.Errorf("Failed to allocate IP %d: %v", i, err)
				break
			}
			allocatedIPs = append(allocatedIPs, ip)
		}
		
		if len(allocatedIPs) == 0 {
			t.Fatal("No IPs could be allocated from subnet")
		}
		
		t.Logf("Allocated %d IPs", len(allocatedIPs))
		
		// Release all allocated IPs
		for _, ip := range allocatedIPs {
			allocator.ReleaseIP(ip)
		}
		
		// Verify clean state
		if count := allocator.GetAllocatedCount(); count != 0 {
			t.Errorf("Expected 0 allocations after cleanup, got %d", count)
		}
		
		// Verify we can allocate again after cleanup
		ip, err := allocator.AllocateIP()
		if err != nil {
			t.Errorf("Failed to allocate IP after cleanup: %v", err)
		} else {
			allocator.ReleaseIP(ip)
		}
	})

	t.Run("VMAssignments", func(t *testing.T) {
		_, subnet, err := net.ParseCIDR("192.168.201.0/24")
		if err != nil {
			t.Fatalf("Failed to parse subnet: %v", err)
		}
		allocator := NewIPAllocator(subnet)
		
		// Allocate IP and assign to VM
		ip, err := allocator.AllocateIP()
		if err != nil {
			t.Fatalf("Failed to allocate IP: %v", err)
		}
		
		allocator.AssignIPToVM("test-vm", ip)
		
		// Verify assignment works
		assignedIP, exists := allocator.GetVMIP("test-vm")
		if !exists {
			t.Error("VM not found after assignment")
		} else if !assignedIP.Equal(ip) {
			t.Errorf("Wrong IP assigned: got %s, expected %s", assignedIP.String(), ip.String())
		}
		
		// Verify reverse lookup
		vmID, exists := allocator.GetIPVM(ip)
		if !exists {
			t.Error("IP not found in reverse lookup")
		} else if vmID != "test-vm" {
			t.Errorf("Wrong VM ID in reverse lookup: got %s, expected test-vm", vmID)
		}
		
		// Cleanup
		allocator.ReleaseIP(ip)
		
		// Verify assignments are cleaned up
		_, exists = allocator.GetVMIP("test-vm")
		if exists {
			t.Error("VM assignment not cleaned up after IP release")
		}
		
		// Verify clean state
		if count := allocator.GetAllocatedCount(); count != 0 {
			t.Errorf("Expected 0 allocations after cleanup, got %d", count)
		}
	})
}