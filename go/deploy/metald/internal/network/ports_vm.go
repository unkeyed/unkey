package network

// GetVMPorts returns all port mappings for a VM
func (m *Manager) GetVMPorts(vmID string) []PortMapping {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.portAllocator.GetVMPorts(vmID)
}

// GetPortVM returns the VM ID that has allocated the given host port
func (m *Manager) GetPortVM(hostPort int) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.portAllocator.GetPortVM(hostPort)
}

// IsPortAllocated checks if a host port is allocated
func (m *Manager) IsPortAllocated(hostPort int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.portAllocator.IsPortAllocated(hostPort)
}

// GetPortAllocationStats returns port allocation statistics
func (m *Manager) GetPortAllocationStats() (allocated, available int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.portAllocator.GetAllocatedCount(), m.portAllocator.GetAvailableCount()
}
