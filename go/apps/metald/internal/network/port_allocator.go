package network

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
)

// PortMapping represents a mapping from VM port to host port
type PortMapping struct {
	VMPort   int    `json:"vm_port"`
	HostPort int    `json:"host_port"`
	Protocol string `json:"protocol"` // tcp or udp
	VMID     string `json:"vm_id"`
}

// PortAllocator manages host port allocation for VMs
type PortAllocator struct {
	// Port ranges for allocation
	minPort int
	maxPort int

	// Port tracking
	allocated map[int]bool             // host port -> allocated
	vmPorts   map[string][]PortMapping // VM ID -> port mappings
	portToVM  map[int]string           // host port -> VM ID

	mu sync.Mutex
}

// NewPortAllocator creates a new port allocator
func NewPortAllocator(minPort, maxPort int) *PortAllocator {
	if minPort <= 0 || maxPort <= 0 || minPort >= maxPort {
		// Use default ephemeral port range if invalid
		minPort = 32768
		maxPort = 65535
	}

	//exhaustruct:ignore
	return &PortAllocator{
		minPort:   minPort,
		maxPort:   maxPort,
		allocated: make(map[int]bool),
		vmPorts:   make(map[string][]PortMapping),
		portToVM:  make(map[int]string),
	}
}

// AllocatePort allocates a host port for the given container port
func (p *PortAllocator) AllocatePort(vmID string, containerPort int, protocol string) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Validate protocol
	if protocol != "tcp" && protocol != "udp" {
		return 0, fmt.Errorf("unsupported protocol: %s", protocol)
	}

	// Use random port allocation for better security and distribution
	portRange := p.maxPort - p.minPort + 1
	maxAttempts := portRange
	if maxAttempts > 1000 {
		maxAttempts = 1000 // Limit attempts to avoid long search times
	}

	// Try random ports first using crypto/rand for security
	for attempt := 0; attempt < maxAttempts; attempt++ {
		randomOffset, err := rand.Int(rand.Reader, big.NewInt(int64(portRange)))
		if err != nil {
			// If crypto/rand fails, fall through to sequential search
			break
		}
		hostPort := p.minPort + int(randomOffset.Int64())
		if !p.allocated[hostPort] {
			return p.doAllocatePort(vmID, hostPort, containerPort, protocol)
		}
	}

	// Fallback: sequential search if random didn't work (very rare case)
	for hostPort := p.minPort; hostPort <= p.maxPort; hostPort++ {
		if !p.allocated[hostPort] {
			return p.doAllocatePort(vmID, hostPort, containerPort, protocol)
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", p.minPort, p.maxPort)
}

// AllocateSpecificPort allocates a specific host port if available
func (p *PortAllocator) AllocateSpecificPort(vmID string, hostPort, containerPort int, protocol string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Validate protocol
	if protocol != "tcp" && protocol != "udp" {
		return fmt.Errorf("unsupported protocol: %s", protocol)
	}

	// Check if port is in range
	if hostPort < p.minPort || hostPort > p.maxPort {
		return fmt.Errorf("port %d outside allocation range %d-%d", hostPort, p.minPort, p.maxPort)
	}

	// Check if already allocated
	if p.allocated[hostPort] {
		return fmt.Errorf("port %d already allocated to VM %s", hostPort, p.portToVM[hostPort])
	}

	_, err := p.doAllocatePort(vmID, hostPort, containerPort, protocol)
	return err
}

// doAllocatePort performs the actual port allocation (internal helper)
func (p *PortAllocator) doAllocatePort(vmID string, hostPort, vmPort int, protocol string) (int, error) {
	// Check for conflicting mapping for same VM
	if mappings, exists := p.vmPorts[vmID]; exists {
		for _, mapping := range mappings {
			if mapping.VMPort == vmPort && mapping.Protocol == protocol {
				return 0, fmt.Errorf("VM %s already has mapping for %s:%d", vmID, protocol, vmPort)
			}
		}
	}

	// Mark port as allocated
	p.allocated[hostPort] = true
	p.portToVM[hostPort] = vmID

	// Create mapping
	mapping := PortMapping{
		VMPort:   vmPort,
		HostPort: hostPort,
		Protocol: protocol,
		VMID:     vmID,
	}

	// Add to VM's port list
	p.vmPorts[vmID] = append(p.vmPorts[vmID], mapping)

	return hostPort, nil
}

// ReleasePort releases a specific host port
func (p *PortAllocator) ReleasePort(hostPort int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if port is allocated
	vmID, allocated := p.portToVM[hostPort]
	if !allocated {
		return fmt.Errorf("port %d is not allocated", hostPort)
	}

	// Remove from allocated ports
	delete(p.allocated, hostPort)
	delete(p.portToVM, hostPort)

	// Remove from VM's port list
	if mappings, exists := p.vmPorts[vmID]; exists {
		newMappings := make([]PortMapping, 0, len(mappings))
		for _, mapping := range mappings {
			if mapping.HostPort != hostPort {
				newMappings = append(newMappings, mapping)
			}
		}

		if len(newMappings) == 0 {
			delete(p.vmPorts, vmID)
		} else {
			p.vmPorts[vmID] = newMappings
		}
	}

	return nil
}

// ReleaseVMPorts releases all ports allocated to a VM
func (p *PortAllocator) ReleaseVMPorts(vmID string) []PortMapping {
	p.mu.Lock()
	defer p.mu.Unlock()

	mappings, exists := p.vmPorts[vmID]
	if !exists {
		return nil
	}

	// Release all host ports for this VM
	for _, mapping := range mappings {
		delete(p.allocated, mapping.HostPort)
		delete(p.portToVM, mapping.HostPort)
	}

	// Remove VM from tracking
	delete(p.vmPorts, vmID)

	return mappings
}

// GetVMPorts returns all port mappings for a VM
func (p *PortAllocator) GetVMPorts(vmID string) []PortMapping {
	p.mu.Lock()
	defer p.mu.Unlock()

	mappings, exists := p.vmPorts[vmID]
	if !exists {
		return nil
	}

	// Return a copy to prevent race conditions
	result := make([]PortMapping, len(mappings))
	copy(result, mappings)
	return result
}

// IsPortAllocated checks if a host port is allocated
func (p *PortAllocator) IsPortAllocated(hostPort int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.allocated[hostPort]
}

// GetPortVM returns the VM ID that has allocated the given host port
func (p *PortAllocator) GetPortVM(hostPort int) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	vmID, exists := p.portToVM[hostPort]
	return vmID, exists
}

// GetAllocatedCount returns the number of allocated ports
func (p *PortAllocator) GetAllocatedCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.allocated)
}

// GetAvailableCount returns the number of available ports
func (p *PortAllocator) GetAvailableCount() int {
	total := p.maxPort - p.minPort + 1
	return total - p.GetAllocatedCount()
}

// GetAllAllocated returns all allocated port mappings
func (p *PortAllocator) GetAllAllocated() []PortMapping {
	p.mu.Lock()
	defer p.mu.Unlock()

	var result []PortMapping
	for _, mappings := range p.vmPorts {
		result = append(result, mappings...)
	}

	return result
}

// Reset clears all port allocations
func (p *PortAllocator) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.allocated = make(map[int]bool)
	p.vmPorts = make(map[string][]PortMapping)
	p.portToVM = make(map[int]string)
}

// GetPortRange returns the port allocation range
func (p *PortAllocator) GetPortRange() (int, int) {
	return p.minPort, p.maxPort
}
