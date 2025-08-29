package network

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/vishvananda/netlink"
	"go.opentelemetry.io/otel/baggage"
)

// CreateVMNetwork sets up networking for a VM
func (m *Manager) CreateVMNetwork(ctx context.Context, vmID string) (*VMNetwork, error) {
	// Default namespace name - will be overridden in CreateVMNetworkWithNamespace
	// if empty to use consistent device naming
	return m.CreateVMNetworkWithNamespace(ctx, vmID, "")
}

// CreateVMNetworkWithNamespace sets up networking for a VM with a specific namespace name
func (m *Manager) CreateVMNetworkWithNamespace(ctx context.Context, vmID, nsName string) (*VMNetwork, error) {
	startTime := time.Now()

	// Extract workspace_id from context baggage for multi-tenant networking
	workspaceID := m.extractWorkspaceID(ctx)
	if workspaceID == "" {
		workspaceID = "default" // Fallback to default workspace
	}

	m.logger.InfoContext(ctx, "creating VM network",
		slog.String("vm_id", vmID),
		slog.String("namespace", nsName),
		slog.String("workspace_id", workspaceID),
	)
	m.logNetworkState("before VM network creation")

	// This ensures atomic check-and-create operation and prevents concurrent VM network creation
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if network already exists
	if existing, exists := m.vmNetworks[vmID]; exists {
		m.logger.WarnContext(ctx, "VM network already exists",
			slog.String("vm_id", vmID),
			slog.String("ip", existing.IPAddress.String()),
		)
		return existing, nil
	}

	// Generate internal network ID for device naming
	// AIDEV-NOTE: This ensures consistent naming across all network devices
	networkID, err := m.idGen.GenerateNetworkID()
	if err != nil {
		m.metrics.RecordVMNetworkCreate(ctx, m.config.BridgeName, false)
		m.metrics.RecordNetworkSetupDuration(ctx, time.Since(startTime), m.config.BridgeName, false)
		return nil, fmt.Errorf("failed to generate network ID: %w", err)
	}

	// Generate device names using consistent naming convention
	deviceNames := GenerateDeviceNames(networkID)

	// AIDEV-NOTE: Multi-bridge tenant allocation with deterministic workspace->bridge mapping
	// Allocate IP and bridge using MultiBridgeManager for security-focused tenant isolation
	ip, bridgeName, err := m.multiBridgeManager.AllocateIPForWorkspace(workspaceID)
	if err != nil {
		m.idGen.ReleaseID(networkID)
		m.metrics.RecordVMNetworkCreate(ctx, m.config.BridgeName, false)
		m.metrics.RecordNetworkSetupDuration(ctx, time.Since(startTime), m.config.BridgeName, false)
		return nil, fmt.Errorf("failed to allocate IP for workspace %s: %w", workspaceID, err)
	}

	m.logger.InfoContext(ctx, "multi-bridge IP allocated",
		slog.String("workspace_id", workspaceID),
		slog.String("ip", ip.String()),
		slog.String("bridge", bridgeName),
	)

	// Generate OUI-based MAC address for tenant identification
	mac, err := m.multiBridgeManager.GenerateTenantMAC(workspaceID)
	if err != nil {
		m.idGen.ReleaseID(networkID)
		m.metrics.RecordVMNetworkCreate(ctx, m.config.BridgeName, false)
		m.metrics.RecordNetworkSetupDuration(ctx, time.Since(startTime), m.config.BridgeName, false)
		return nil, fmt.Errorf("failed to generate tenant MAC for workspace %s: %w", workspaceID, err)
	}

	// Override namespace name if provided (e.g., by jailer)
	// AIDEV-NOTE: CRITICAL FIX - Use deviceNames.Namespace when nsName is empty to ensure
	// namespace name matches the veth device names (vn_{networkID}). This prevents
	// "no such device" errors when configuring veth inside the namespace.
	actualNsName := nsName
	if actualNsName == "" {
		actualNsName = deviceNames.Namespace
	}

	// AIDEV-NOTE: CRITICAL FIX - Ensure proper cleanup order on any failure
	var cleanupFunctions []func()
	defer func() {
		// Execute cleanup functions in reverse order if we haven't succeeded
		for i := len(cleanupFunctions) - 1; i >= 0; i-- {
			cleanupFunctions[i]()
		}
	}()

	// Create network namespace if it doesn't exist
	// It might have been pre-created by the jailer
	if err := m.createNamespace(actualNsName); err != nil {
		cleanupFunctions = append(cleanupFunctions, func() { m.allocator.ReleaseIP(ip) })
		cleanupFunctions = append(cleanupFunctions, func() { m.idGen.ReleaseID(networkID) })
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}
	cleanupFunctions = append(cleanupFunctions, func() { m.deleteNamespace(actualNsName) })

	// Calculate bridge subnet for networking setup
	bridgeSubnet := fmt.Sprintf("172.16.%d.0/24", m.multiBridgeManager.GetBridgeForWorkspace(workspaceID))

	// Create TAP device and configure networking
	if err := m.setupVMNetworking(actualNsName, deviceNames, ip, mac, bridgeSubnet); err != nil {
		cleanupFunctions = append(cleanupFunctions, func() { m.allocator.ReleaseIP(ip) })
		cleanupFunctions = append(cleanupFunctions, func() { m.idGen.ReleaseID(networkID) })
		return nil, fmt.Errorf("failed to setup VM networking: %w", err)
	}

	// Attach VM interface to the correct bridge for multi-tenant isolation
	if err := m.attachVMToBridge(bridgeName, deviceNames.VethHost); err != nil {
		cleanupFunctions = append(cleanupFunctions, func() { m.allocator.ReleaseIP(ip) })
		cleanupFunctions = append(cleanupFunctions, func() { m.idGen.ReleaseID(networkID) })
		return nil, fmt.Errorf("failed to attach VM to bridge %s: %w", bridgeName, err)
	}

	m.logger.InfoContext(ctx, "VM attached to tenant bridge",
		slog.String("workspace_id", workspaceID),
		slog.String("veth_host", deviceNames.VethHost),
		slog.String("bridge", bridgeName),
	)

	// Create VM network info using bridge subnet
	_, subnet, _ := net.ParseCIDR(bridgeSubnet)
	gateway := make(net.IP, len(subnet.IP))
	copy(gateway, subnet.IP)
	gateway[len(gateway)-1] = 1 // Gateway is .1 in each bridge subnet

	vmNet := &VMNetwork{ //nolint:exhaustruct // IPv6Address and Routes fields use appropriate zero values
		VMID:        vmID,
		NetworkID:   networkID,
		WorkspaceID: workspaceID,
		Namespace:   actualNsName,
		TapDevice:   deviceNames.TAP,
		IPAddress:   ip,
		Netmask:     net.CIDRMask(29, 32), // Use /29 to match actual veth configuration
		Gateway:     gateway,
		MacAddress:  mac,
		DNSServers:  m.config.DNSServers,
		CreatedAt:   time.Now(),
		VLANID:      0, // No VLAN - using bridge-based isolation
	}

	m.vmNetworks[vmID] = vmNet

	// AIDEV-NOTE: Clear cleanup functions since we succeeded - prevent resource cleanup
	cleanupFunctions = nil

	// Record successful network creation metrics
	duration := time.Since(startTime)
	m.metrics.RecordVMNetworkCreate(ctx, m.config.BridgeName, true)
	m.metrics.RecordNetworkSetupDuration(ctx, duration, m.config.BridgeName, true)

	m.logger.InfoContext(ctx, "created VM network",
		slog.String("vm_id", vmID),
		slog.String("workspace_id", workspaceID),
		slog.String("ip", ip.String()),
		slog.String("mac", mac),
		slog.String("tap", deviceNames.TAP),
		slog.String("namespace", actualNsName),
		slog.String("network_id", networkID),
		slog.String("bridge", bridgeName),
		slog.String("bridge_subnet", bridgeSubnet),
		slog.Duration("setup_duration", duration),
	)

	return vmNet, nil
}

// DeleteVMNetwork removes networking for a VM
func (m *Manager) DeleteVMNetwork(ctx context.Context, vmID string) error {
	startTime := time.Now()

	m.logger.InfoContext(ctx, "deleting VM network",
		slog.String("vm_id", vmID),
	)

	m.mu.Lock()
	defer m.mu.Unlock()

	vmNet, exists := m.vmNetworks[vmID]
	if !exists {
		m.logger.InfoContext(ctx, "VM network already deleted",
			slog.String("vm_id", vmID),
		)
		return nil // Already deleted
	}

	// Release IP from multi-bridge manager
	// AIDEV-NOTE: CRITICAL FIX - Use multi-bridge manager to release IP properly
	if err := m.multiBridgeManager.ReleaseIPForWorkspace(vmNet.WorkspaceID, vmNet.IPAddress); err != nil {
		m.logger.WarnContext(ctx, "failed to release IP from multi-bridge manager",
			slog.String("workspace_id", vmNet.WorkspaceID),
			slog.String("ip", vmNet.IPAddress.String()),
			slog.String("error", err.Error()),
		)
		// Continue with cleanup anyway
	}

	// Also release from old allocator for backward compatibility
	m.allocator.ReleaseIP(vmNet.IPAddress)

	// Delete network namespace FIRST
	// AIDEV-NOTE: Deleting namespace automatically cleans up all interfaces inside it
	// This prevents issues with trying to delete interfaces that no longer exist
	m.deleteNamespace(vmNet.Namespace)

	// Delete veth pair (if it still exists on host)
	// AIDEV-NOTE: After namespace deletion, only the host side of veth pair remains
	deviceNames := GenerateDeviceNames(vmNet.NetworkID)
	if link, err := netlink.LinkByName(deviceNames.VethHost); err == nil {
		if delErr := netlink.LinkDel(link); delErr != nil {
			m.logger.WarnContext(ctx, "Failed to delete veth pair", "device", deviceNames.VethHost, "error", delErr)
		} else {
			m.logger.InfoContext(ctx, "Deleted veth pair", "device", deviceNames.VethHost)
		}
	}

	// AIDEV-NOTE: Delete TAP device (CRITICAL FIX - this was missing!)
	// TAP devices are created in host namespace for Firecracker access and must be explicitly cleaned up
	if link, err := netlink.LinkByName(deviceNames.TAP); err == nil {
		if delErr := netlink.LinkDel(link); delErr != nil {
			m.logger.WarnContext(ctx, "Failed to delete TAP device",
				"device", deviceNames.TAP, "error", delErr)
		} else {
			m.logger.InfoContext(ctx, "Deleted TAP device", "device", deviceNames.TAP)
		}
	}

	// Verify cleanup completed successfully
	if err := m.verifyNetworkCleanup(ctx, vmID, deviceNames); err != nil {
		m.logger.WarnContext(ctx, "Network cleanup verification failed",
			"vm_id", vmID, "error", err)
	}

	// Release the network ID for reuse
	m.idGen.ReleaseID(vmNet.NetworkID)

	delete(m.vmNetworks, vmID)

	// Record successful network deletion metrics
	duration := time.Since(startTime)
	m.metrics.RecordVMNetworkDelete(ctx, m.config.BridgeName, true)
	m.metrics.RecordNetworkCleanupDuration(ctx, duration, m.config.BridgeName, true)

	m.logger.InfoContext(ctx, "deleted VM network",
		slog.String("vm_id", vmID),
		slog.String("network_id", vmNet.NetworkID),
		slog.String("ip", vmNet.IPAddress.String()),
		slog.Duration("cleanup_duration", duration),
	)

	return nil
}

// GetVMNetwork returns network information for a VM
func (m *Manager) GetVMNetwork(vmID string) (*VMNetwork, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vmNet, exists := m.vmNetworks[vmID]
	if !exists {
		return nil, fmt.Errorf("network not found for VM %s", vmID)
	}

	return vmNet, nil
}

// extractWorkspaceID extracts workspace_id from OpenTelemetry baggage context
func (m *Manager) extractWorkspaceID(ctx context.Context) string {
	// Extract baggage from context
	bag := baggage.FromContext(ctx)

	// Get workspace_id from baggage
	workspaceID := bag.Member("workspace_id")
	if workspaceID.Key() != "" {
		return workspaceID.Value()
	}

	// Fallback: check for workspace_id in different baggage keys
	for _, member := range bag.Members() {
		if member.Key() == "workspace_id" || member.Key() == "workspaceId" {
			return member.Value()
		}
	}

	return "" // No workspace_id found
}
