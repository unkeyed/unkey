package network

import (
	"fmt"
	"log/slog"
	"net"
	"strings"
)

// AllocatePortsForVM allocates host ports and sets up port forwarding for a VM
// This is the high-level workflow that coordinates allocation + forwarding + cleanup
func (m *Manager) AllocatePortsForVM(vmID string, vmIP net.IP, exposedPorts []string) ([]PortMapping, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure nftables table exists
	if err := m.ensureNftablesTable(); err != nil {
		return nil, fmt.Errorf("failed to ensure nftables setup: %w", err)
	}

	var mappings []PortMapping
	var createdRules []PortMapping // Track rules to clean up on error

	// Cleanup function to remove rules if something goes wrong
	cleanup := func() {
		for _, mapping := range createdRules {
			if err := m.removePortForwarding(vmIP, mapping); err != nil {
				m.logger.Warn("failed to cleanup port forwarding rule",
					slog.String("vm_id", vmID),
					slog.Int("host_port", mapping.HostPort),
					slog.String("error", err.Error()),
				)
			}
		}
		// Also clean up allocated ports
		m.releaseVMPortsLocked(vmID)
	}

	for _, portSpec := range exposedPorts {
		// Parse port format: can be "80", "80/tcp", "80/udp"
		parts := strings.Split(portSpec, "/")
		if len(parts) == 0 {
			continue
		}

		var containerPort int
		protocol := "tcp" // default

		if _, err := fmt.Sscanf(parts[0], "%d", &containerPort); err != nil {
			m.logger.Warn("invalid port format",
				slog.String("port_spec", portSpec),
				slog.String("error", err.Error()),
			)
			continue
		}

		if len(parts) > 1 {
			protocol = strings.ToLower(parts[1])
		}

		// Allocate host port
		hostPort, err := m.portAllocator.AllocatePort(vmID, containerPort, protocol)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("failed to allocate port %s for VM %s: %w", portSpec, vmID, err)
		}

		mapping := PortMapping{
			ContainerPort: containerPort,
			HostPort:      hostPort,
			Protocol:      protocol,
			VMID:          vmID,
		}

		// Create DNAT rule for port forwarding
		if err := m.setupPortForwarding(vmIP, mapping); err != nil {
			cleanup()
			return nil, fmt.Errorf("failed to setup port forwarding for %s: %w", portSpec, err)
		}

		createdRules = append(createdRules, mapping)
		mappings = append(mappings, mapping)

		m.logger.Info("allocated port mapping with forwarding rule",
			slog.String("vm_id", vmID),
			slog.String("vm_ip", vmIP.String()),
			slog.Int("container_port", containerPort),
			slog.Int("host_port", hostPort),
			slog.String("protocol", protocol),
		)
	}

	return mappings, nil
}

// ReleaseVMPorts releases all ports allocated to a VM
// This coordinates both port allocation cleanup and forwarding rule removal
func (m *Manager) ReleaseVMPorts(vmID string) []PortMapping {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.releaseVMPortsLocked(vmID)
}

// releaseVMPortsLocked releases VM ports and removes DNAT rules with lock already held
// This is the internal coordination function that handles the complete cleanup workflow
func (m *Manager) releaseVMPortsLocked(vmID string) []PortMapping {
	mappings := m.portAllocator.ReleaseVMPorts(vmID)

	// Get VM network info to find IP address for DNAT rule removal
	if vmNet, exists := m.vmNetworks[vmID]; exists && len(mappings) > 0 {
		// Remove DNAT rules for each port mapping
		for _, mapping := range mappings {
			if err := m.removePortForwarding(vmNet.IPAddress, mapping); err != nil {
				m.logger.Warn("failed to remove port forwarding rule",
					slog.String("vm_id", vmID),
					slog.String("vm_ip", vmNet.IPAddress.String()),
					slog.Int("host_port", mapping.HostPort),
					slog.Int("container_port", mapping.ContainerPort),
					slog.String("protocol", mapping.Protocol),
					slog.String("error", err.Error()),
				)
			} else {
				m.logger.Info("removed port forwarding rule",
					slog.String("vm_id", vmID),
					slog.String("vm_ip", vmNet.IPAddress.String()),
					slog.Int("host_port", mapping.HostPort),
					slog.Int("container_port", mapping.ContainerPort),
					slog.String("protocol", mapping.Protocol),
				)
			}
		}
	}

	for _, mapping := range mappings {
		m.logger.Info("released port mapping",
			slog.String("vm_id", vmID),
			slog.Int("container_port", mapping.ContainerPort),
			slog.Int("host_port", mapping.HostPort),
			slog.String("protocol", mapping.Protocol),
		)
	}

	return mappings
}
