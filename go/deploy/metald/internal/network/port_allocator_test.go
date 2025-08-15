package network

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// AIDEV-NOTE: Regression tests for port allocator concurrency issues
// These tests verify that port allocation/deallocation operations are thread-safe

// TestPortAllocatorConcurrency tests concurrent port allocation and deallocation
func TestPortAllocatorConcurrency(t *testing.T) {
	allocator := NewPortAllocator(40000, 41000)

	numGoroutines := 10
	portsPerGoroutine := 50
	var wg sync.WaitGroup
	var totalAllocated int32
	var totalFailed int32

	// Concurrent allocation
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			var allocated []int
			vmID := fmt.Sprintf("vm-%d", goroutineID)

			// Allocate ports
			for j := 0; j < portsPerGoroutine; j++ {
				containerPort := 8000 + j
				hostPort, err := allocator.AllocatePort(vmID, containerPort, "tcp")
				if err != nil {
					atomic.AddInt32(&totalFailed, 1)
					continue
				}
				allocated = append(allocated, hostPort)
				atomic.AddInt32(&totalAllocated, 1)
			}

			// Release half the ports
			for k := 0; k < len(allocated)/2; k++ {
				if err := allocator.ReleasePort(allocated[k]); err != nil {
					t.Errorf("Failed to release port %d: %v", allocated[k], err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify state consistency
	allocatedCount := allocator.GetAllocatedCount()
	expectedRemaining := int(totalAllocated) / 2 // Half were released

	t.Logf("Total attempted: %d, succeeded: %d, failed: %d",
		numGoroutines*portsPerGoroutine, totalAllocated, totalFailed)
	t.Logf("Allocated after partial release: %d, expected ~%d",
		allocatedCount, expectedRemaining)

	// Should have roughly half remaining (allowing some variance)
	if allocatedCount < expectedRemaining/2 || allocatedCount > expectedRemaining*2 {
		t.Errorf("Unexpected allocation count: got %d, expected around %d",
			allocatedCount, expectedRemaining)
	}
}

// TestPortAllocatorRaceCondition tests for race conditions in map updates
func TestPortAllocatorRaceCondition(t *testing.T) {
	allocator := NewPortAllocator(42000, 43000)

	numOperations := 100
	var wg sync.WaitGroup

	// Perform many rapid allocate/release cycles on same VM
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			vmID := "test-race-vm"
			containerPort := 8000 + (idx % 10) // Reuse container ports

			// Try to allocate
			hostPort, err := allocator.AllocatePort(vmID, containerPort, "tcp")
			if err != nil {
				// Some failures expected due to port reuse
				return
			}

			// Immediately release
			if err := allocator.ReleasePort(hostPort); err != nil {
				t.Errorf("Failed to release port %d: %v", hostPort, err)
			}
		}(i)
	}

	wg.Wait()

	// Final state should be clean
	finalCount := allocator.GetAllocatedCount()
	if finalCount != 0 {
		t.Errorf("Expected 0 allocated ports, got %d", finalCount)
	}

	// Verify VM has no ports left
	vmPorts := allocator.GetVMPorts("test-race-vm")
	if len(vmPorts) != 0 {
		t.Errorf("Expected VM to have 0 ports, got %d", len(vmPorts))
	}
}

// TestPortAllocatorMapConsistency tests that internal maps remain consistent
func TestPortAllocatorMapConsistency(t *testing.T) {
	allocator := NewPortAllocator(44000, 45000)

	// Allocate some ports
	testVMs := []string{"vm1", "vm2", "vm3"}
	expectedPorts := make(map[string][]int)

	for _, vmID := range testVMs {
		for i := 0; i < 5; i++ {
			containerPort := 9000 + i
			hostPort, err := allocator.AllocatePort(vmID, containerPort, "tcp")
			if err != nil {
				t.Fatalf("Failed to allocate port for %s: %v", vmID, err)
			}
			expectedPorts[vmID] = append(expectedPorts[vmID], hostPort)
		}
	}

	// Verify consistency across different access methods
	totalAllocated := allocator.GetAllocatedCount()
	expectedTotal := len(testVMs) * 5

	if totalAllocated != expectedTotal {
		t.Errorf("Allocated count mismatch: got %d, expected %d", totalAllocated, expectedTotal)
	}

	// Verify each VM's ports are tracked correctly
	for vmID, expectedHostPorts := range expectedPorts {
		vmPorts := allocator.GetVMPorts(vmID)
		if len(vmPorts) != len(expectedHostPorts) {
			t.Errorf("VM %s port count mismatch: got %d, expected %d",
				vmID, len(vmPorts), len(expectedHostPorts))
			continue
		}

		// Verify reverse lookup works
		for _, hostPort := range expectedHostPorts {
			foundVMID, exists := allocator.GetPortVM(hostPort)
			if !exists {
				t.Errorf("Port %d should be allocated but wasn't found", hostPort)
				continue
			}
			if foundVMID != vmID {
				t.Errorf("Port %d reverse lookup returned %s, expected %s",
					hostPort, foundVMID, vmID)
			}
		}
	}

	// Clean up and verify
	for vmID := range expectedPorts {
		released := allocator.ReleaseVMPorts(vmID)
		if len(released) != 5 {
			t.Errorf("Released %d ports for %s, expected 5", len(released), vmID)
		}
	}

	finalCount := allocator.GetAllocatedCount()
	if finalCount != 0 {
		t.Errorf("Expected 0 allocated ports after cleanup, got %d", finalCount)
	}
}

// TestConcurrentVMPortOperations tests concurrent operations on same VM
func TestConcurrentVMPortOperations(t *testing.T) {
	allocator := NewPortAllocator(46000, 47000)

	vmID := "concurrent-test-vm"
	numOperations := 50
	var wg sync.WaitGroup
	var successCount int32

	// Multiple goroutines trying to allocate ports for same VM
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			containerPort := 10000 + idx // Different container ports
			hostPort, err := allocator.AllocatePort(vmID, containerPort, "tcp")
			if err != nil {
				t.Logf("Failed to allocate port %d: %v", containerPort, err)
				return
			}

			atomic.AddInt32(&successCount, 1)

			// Verify port was actually allocated
			if !allocator.IsPortAllocated(hostPort) {
				t.Errorf("Port %d should be allocated but wasn't found", hostPort)
			}

			// Verify reverse lookup
			foundVM, exists := allocator.GetPortVM(hostPort)
			if !exists || foundVM != vmID {
				t.Errorf("Port %d reverse lookup failed: got %s, expected %s",
					hostPort, foundVM, vmID)
			}
		}(i)
	}

	wg.Wait()

	// Verify final state
	vmPorts := allocator.GetVMPorts(vmID)
	if len(vmPorts) != int(successCount) {
		t.Errorf("VM port count mismatch: got %d, expected %d",
			len(vmPorts), successCount)
	}

	// Clean up
	released := allocator.ReleaseVMPorts(vmID)
	if len(released) != int(successCount) {
		t.Errorf("Released port count mismatch: got %d, expected %d",
			len(released), successCount)
	}

	t.Logf("Successfully handled %d concurrent allocations for single VM", successCount)
}
