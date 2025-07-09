package network

import (
	"fmt"
	"net"
	"sync"
)

// IPAllocator manages IP address allocation for VMs
type IPAllocator struct {
	subnet    *net.IPNet
	allocated map[string]bool   // IP string -> allocated
	vmToIP    map[string]net.IP // VM ID -> IP
	ipToVM    map[string]string // IP string -> VM ID
	mu        sync.Mutex

	// Configuration
	startOffset int // Start allocating from subnet + startOffset
	endOffset   int // Stop allocating at subnet + endOffset
}

// NewIPAllocator creates a new IP allocator for the given subnet
func NewIPAllocator(subnet *net.IPNet) *IPAllocator {
	//exhaustruct:ignore
	return &IPAllocator{
		subnet:      subnet,
		allocated:   make(map[string]bool),
		vmToIP:      make(map[string]net.IP),
		ipToVM:      make(map[string]string),
		startOffset: 2,   // Start from .2 (reserve .1 for gateway)
		endOffset:   254, // Stop at .254 (reserve .255 for broadcast)
	}
}

// AllocateIP allocates a new IP address
func (a *IPAllocator) AllocateIP() (net.IP, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// For simplicity, we'll work with /24 subnets
	// In production, this should handle various subnet sizes
	ones, bits := a.subnet.Mask.Size()
	if ones > 24 || bits != 32 {
		return nil, fmt.Errorf("only /24 or smaller IPv4 subnets supported, got /%d", ones)
	}

	baseIP := a.subnet.IP.To4()
	if baseIP == nil {
		return nil, fmt.Errorf("invalid IPv4 subnet")
	}

	// Try to find an available IP
	for i := a.startOffset; i <= a.endOffset; i++ {
		// Create IP address
		ip := make(net.IP, 4)
		copy(ip, baseIP)
		ip[3] = byte(i)

		// Check if already allocated
		if !a.allocated[ip.String()] {
			a.allocated[ip.String()] = true
			return ip, nil
		}
	}

	return nil, fmt.Errorf("no available IPs in subnet %s", a.subnet.String())
}

// AllocateSpecificIP allocates a specific IP address if available
func (a *IPAllocator) AllocateSpecificIP(ip net.IP) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if IP is in our subnet
	if !a.subnet.Contains(ip) {
		return fmt.Errorf("IP %s not in subnet %s", ip.String(), a.subnet.String())
	}

	// Check if already allocated
	if a.allocated[ip.String()] {
		return fmt.Errorf("IP %s already allocated", ip.String())
	}

	// Check if it's a reserved IP (.0, .1, .255 for /24)
	lastOctet := ip.To4()[3]
	if lastOctet == 0 || lastOctet == 1 || lastOctet == 255 {
		return fmt.Errorf("IP %s is reserved", ip.String())
	}

	a.allocated[ip.String()] = true
	return nil
}

// ReleaseIP releases an allocated IP address
func (a *IPAllocator) ReleaseIP(ip net.IP) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.allocated, ip.String())

	// Clean up VM mappings if they exist
	if vmID, exists := a.ipToVM[ip.String()]; exists {
		delete(a.vmToIP, vmID)
		delete(a.ipToVM, ip.String())
	}
}

// AssignIPToVM records the IP-to-VM mapping
func (a *IPAllocator) AssignIPToVM(vmID string, ip net.IP) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.vmToIP[vmID] = ip
	a.ipToVM[ip.String()] = vmID
}

// GetVMIP returns the IP assigned to a VM
func (a *IPAllocator) GetVMIP(vmID string) (net.IP, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	ip, exists := a.vmToIP[vmID]
	return ip, exists
}

// GetIPVM returns the VM ID assigned to an IP
func (a *IPAllocator) GetIPVM(ip net.IP) (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	vmID, exists := a.ipToVM[ip.String()]
	return vmID, exists
}

// IsAllocated checks if an IP is allocated
func (a *IPAllocator) IsAllocated(ip net.IP) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.allocated[ip.String()]
}

// GetAllocatedCount returns the number of allocated IPs
func (a *IPAllocator) GetAllocatedCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()

	return len(a.allocated)
}

// GetAvailableCount returns the number of available IPs
func (a *IPAllocator) GetAvailableCount() int {
	total := a.endOffset - a.startOffset + 1
	return total - a.GetAllocatedCount()
}

// GetAllAllocated returns all allocated IPs
func (a *IPAllocator) GetAllAllocated() []net.IP {
	a.mu.Lock()
	defer a.mu.Unlock()

	ips := make([]net.IP, 0, len(a.allocated))
	for ipStr := range a.allocated {
		if ip := net.ParseIP(ipStr); ip != nil {
			ips = append(ips, ip)
		}
	}

	return ips
}

// Reset clears all allocations
func (a *IPAllocator) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.allocated = make(map[string]bool)
	a.vmToIP = make(map[string]net.IP)
	a.ipToVM = make(map[string]string)
}
