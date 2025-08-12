package network

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"go.opentelemetry.io/otel/baggage"
)

// VerifyBridge verifies bridge infrastructure exists and is properly configured
// Bridge is managed by metald-bridge.service, not created by metald
// AIDEV-NOTE: Architectural change - bridge is now external infrastructure
func VerifyBridge(logger *slog.Logger, netConfig *Config, mainConfig *config.NetworkConfig) error {
	logger = logger.With("component", "bridge-verify")

	logger.Info("verifying bridge infrastructure",
		slog.String("bridge_name", netConfig.BridgeName),
	)

	// Check if bridge exists
	bridge, err := netlink.LinkByName(netConfig.BridgeName)
	if err != nil {
		return fmt.Errorf("bridge '%s' not found - ensure metald-bridge.service is running: %w", netConfig.BridgeName, err)
	}

	// Verify bridge is administratively up (bridges may show OperDown with NO-CARRIER, which is normal)
	// Check that the interface has the UP flag, not the operational state
	if (bridge.Attrs().Flags & net.FlagUp) == 0 {
		return fmt.Errorf("bridge '%s' is administratively DOWN - check metald-bridge.service",
			netConfig.BridgeName)
	}

	// Verify bridge has expected IP address for host-level routing
	addrs, err := netlink.AddrList(bridge, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to get bridge addresses: %w", err)
	}

	expectedIP, expectedNet, err := net.ParseCIDR(netConfig.BridgeIP)
	if err != nil {
		return fmt.Errorf("invalid bridge IP config '%s': %w", netConfig.BridgeIP, err)
	}

	hasExpectedIP := false
	for _, addr := range addrs {
		if addr.IP.Equal(expectedIP) && addr.Mask.String() == expectedNet.Mask.String() {
			hasExpectedIP = true
			break
		}
	}

	if !hasExpectedIP {
		return fmt.Errorf("bridge '%s' missing expected IP %s - check bridge service configuration",
			netConfig.BridgeName, netConfig.BridgeIP)
	}

	logger.Info("bridge infrastructure verified successfully",
		slog.String("bridge_name", netConfig.BridgeName),
		slog.String("bridge_ip", expectedIP.String()),
		slog.String("state", bridge.Attrs().OperState.String()),
		slog.Int("mtu", bridge.Attrs().MTU),
	)
	return nil
}

// InitializeBridge creates bridge infrastructure during startup (DEPRECATED)
// This function is kept for backward compatibility but should not be used
// Use VerifyBridge instead - bridge should be managed by metald-bridge.service
func InitializeBridge(logger *slog.Logger, netConfig *Config, mainConfig *config.NetworkConfig) error {
	logger.Warn("InitializeBridge is deprecated - bridge should be managed by metald-bridge.service")
	return VerifyBridge(logger, netConfig, mainConfig)
}

// attachVMToBridge attaches a VM interface to the specified bridge
func (m *Manager) attachVMToBridge(bridgeName, interfaceName string) error {
	// Get bridge and VM interface
	bridge, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("bridge %s not found: %w", bridgeName, err)
	}

	vmInterface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("VM interface %s not found: %w", interfaceName, err)
	}

	// AIDEV-NOTE: CRITICAL FIX - Don't attach point-to-point veth interfaces to bridge
	// Point-to-point veths with IP addresses should not be bridge members
	// They operate as routed interfaces, not bridge segments
	// Only attach to bridge if the interface doesn't have an IP (legacy TAP mode)
	addrs, err := netlink.AddrList(vmInterface, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to check interface addresses: %w", err)
	}

	if len(addrs) > 0 {
		m.logger.Info("skipping bridge attachment for point-to-point interface",
			slog.String("interface", interfaceName),
			slog.String("bridge", bridgeName),
			slog.String("reason", "interface has IP address - operating in routed mode"),
		)
		return nil // Don't attach to bridge
	}

	// Attach interface to bridge (legacy TAP-only mode)
	if err := netlink.LinkSetMaster(vmInterface, bridge); err != nil {
		return fmt.Errorf("failed to attach interface to bridge: %w", err)
	}

	m.logger.Info("VM interface attached to bridge",
		slog.String("interface", interfaceName),
		slog.String("bridge", bridgeName),
	)

	return nil
}

// Config holds network configuration
type Config struct {
	BridgeName      string // Default: "br-vms"
	BridgeIP        string // Default: "172.31.0.1/19"
	VMSubnet        string // Default: "172.31.0.0/19"
	EnableIPv6      bool
	DNSServers      []string // Default: ["8.8.8.8", "8.8.4.4"]
	EnableRateLimit bool
	RateLimitMbps   int // Per VM rate limit in Mbps

	// Port allocation configuration
	PortRangeMin int // Default: 32768
	PortRangeMax int // Default: 65535
}

// DefaultConfig returns default network configuration
func DefaultConfig() *Config {
	return &Config{ //nolint:exhaustruct // EnableIPv6 field uses zero value (false) which is appropriate for default config
		BridgeName:      "br-vms",
		BridgeIP:        "172.31.0.1/19",
		VMSubnet:        "172.31.0.0/19",
		DNSServers:      []string{"8.8.8.8", "8.8.4.4"},
		EnableRateLimit: true,
		RateLimitMbps:   100,   // 100 Mbps default
		PortRangeMin:    32768, // Ephemeral port range start
		PortRangeMax:    65535, // Ephemeral port range end
	}
}

// Manager handles VM networking
type Manager struct {
	logger        *slog.Logger
	config        *Config
	allocator     *IPAllocator
	portAllocator *PortAllocator
	idGen         *IDGenerator
	mu            sync.RWMutex
	vmNetworks    map[string]*VMNetwork

	// Runtime state (hostProtection removed - managed externally)

	// Multi-tenant workspace management
	multiBridgeManager *MultiBridgeManager
	metrics            *NetworkMetrics

	// AIDEV-NOTE: CRITICAL FIX - Bridge state synchronization
	// bridgeMu protects bridgeCreated and bridgeInitTime to prevent race conditions
	// during bridge verification and creation operations
	bridgeMu       sync.RWMutex
	bridgeCreated  bool
	bridgeInitTime time.Time

	iptablesRules []string
}

// NewManager creates a new network manager
func NewManager(logger *slog.Logger, netConfig *Config, mainConfig *config.NetworkConfig) (*Manager, error) {
	if netConfig == nil {
		netConfig = DefaultConfig()
	}

	logger = logger.With("component", "network-manager")
	logger.Info("creating network manager",
		slog.String("bridge_name", netConfig.BridgeName),
		slog.String("bridge_ip", netConfig.BridgeIP),
		slog.String("vm_subnet", netConfig.VMSubnet),
		slog.Bool("host_protection", mainConfig.EnableHostProtection),
	)

	_, subnet, err := net.ParseCIDR(netConfig.VMSubnet)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet: %w", err)
	}

	// Initialize network metrics
	networkMetrics, err := NewNetworkMetrics(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create network metrics: %w", err)
	}

	m := &Manager{ //nolint:exhaustruct // mu, bridgeCreated, and iptablesRules fields use appropriate zero values
		logger:             logger,
		config:             netConfig,
		allocator:          NewIPAllocator(subnet),
		portAllocator:      NewPortAllocator(netConfig.PortRangeMin, netConfig.PortRangeMax),
		idGen:              NewIDGenerator(),
		metrics:            networkMetrics,
		vmNetworks:         make(map[string]*VMNetwork),
		multiBridgeManager: NewMultiBridgeManager(mainConfig.BridgeCount, "br-tenant", logger),
	}

	// Set bridge max VMs based on configuration
	m.metrics.SetBridgeMaxVMs(netConfig.BridgeName, int64(mainConfig.MaxVMsPerBridge))

	// Log current network state before verification
	m.logNetworkState("before bridge verification")

	// Verify bridge infrastructure is ready (should be pre-initialized at startup)
	if err := m.verifyBridgeReady(); err != nil {
		m.logger.Error("bridge infrastructure not ready",
			slog.String("error", err.Error()),
		)
		m.logNetworkState("after failed bridge verification")
		return nil, fmt.Errorf("bridge infrastructure not ready: %w", err)
	}

	// Log network state after verification
	m.logNetworkState("after successful bridge verification")

	return m, nil
}

// verifyBridgeReady checks that bridge infrastructure is ready for VM operations
func (m *Manager) verifyBridgeReady() error {
	// AIDEV-NOTE: CRITICAL FIX - Use dedicated bridge mutex to prevent race conditions
	// Check if bridge is already verified without blocking other operations
	m.bridgeMu.RLock()
	if m.bridgeCreated {
		// Bridge was already verified, but double-check it still exists
		if link, err := netlink.LinkByName(m.config.BridgeName); err == nil {
			isUp := link.Attrs().OperState == netlink.OperUp ||
				(link.Attrs().Flags&net.FlagUp) != 0
			if isUp {
				m.bridgeMu.RUnlock()
				return nil
			}
		}
		// Bridge state changed, need to re-verify
	}
	m.bridgeMu.RUnlock()

	// Acquire write lock for verification
	m.bridgeMu.Lock()
	defer m.bridgeMu.Unlock()

	// Double-check after acquiring write lock (another goroutine might have verified)
	if m.bridgeCreated {
		if link, err := netlink.LinkByName(m.config.BridgeName); err == nil {
			isUp := link.Attrs().OperState == netlink.OperUp ||
				(link.Attrs().Flags&net.FlagUp) != 0
			if isUp {
				return nil
			}
		}
		// Bridge state changed, reset and re-verify
		m.bridgeCreated = false
	}

	// Allow some time for bridge state to stabilize after creation
	maxRetries := 5
	retryDelay := 100 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		link, err := netlink.LinkByName(m.config.BridgeName)
		if err != nil {
			return fmt.Errorf("bridge %s not found - bridge initialization may have failed", m.config.BridgeName)
		}

		// Check if bridge is UP or has the correct flags (UP flag set)
		// For bridges, OperState might be "unknown" but flags will show if it's actually up
		isUp := link.Attrs().OperState == netlink.OperUp ||
			(link.Attrs().Flags&net.FlagUp) != 0

		if isUp {
			m.bridgeCreated = true
			m.bridgeInitTime = time.Now()
			m.logger.Info("verified bridge is ready for VM operations",
				slog.String("bridge", m.config.BridgeName),
				slog.String("state", link.Attrs().OperState.String()),
				slog.String("flags", link.Attrs().Flags.String()),
				slog.Int("attempt", attempt),
			)
			return nil
		}

		if attempt < maxRetries {
			m.logger.Debug("bridge not yet ready, retrying",
				slog.String("bridge", m.config.BridgeName),
				slog.String("state", link.Attrs().OperState.String()),
				slog.String("flags", link.Attrs().Flags.String()),
				slog.Int("attempt", attempt),
				slog.Int("max_retries", maxRetries),
			)
			time.Sleep(retryDelay)
		}
	}

	// Final attempt - get current state for error message
	link, err := netlink.LinkByName(m.config.BridgeName)
	if err != nil {
		return fmt.Errorf("bridge %s not found after %d attempts", m.config.BridgeName, maxRetries)
	}

	return fmt.Errorf("bridge %s is not ready after %d attempts (current state: %s, flags: %s)",
		m.config.BridgeName, maxRetries, link.Attrs().OperState.String(), link.Attrs().Flags.String())
}

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

	// AIDEV-NOTE: CRITICAL FIX - Hold lock for entire duration to prevent race conditions
	// This ensures atomic check-and-create operation and prevents concurrent VM network creation
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if network already exists (now protected by lock)
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
		Netmask:     subnet.Mask, // Use bridge subnet mask
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

// setupVMNetworking configures the network namespace and TAP device
func (m *Manager) setupVMNetworking(nsName string, deviceNames *NetworkDeviceNames, ip net.IP, mac string, workspaceSubnet string) error {
	// AIDEV-NOTE: Now running as root, no need for nsenter workarounds

	// Use device names from the consistent naming convention
	vethHost := deviceNames.VethHost
	vethNS := deviceNames.VethNS

	// Create veth pair using netlink (preferred when running as root)
	veth := &netlink.Veth{ //nolint:exhaustruct // Only setting required fields, other veth fields use appropriate defaults
		LinkAttrs: netlink.LinkAttrs{Name: vethHost}, //nolint:exhaustruct // Only setting Name field, other link attributes use appropriate defaults
		PeerName:  vethNS,
	}

	m.logger.Info("creating veth pair",
		slog.String("host_end", vethHost),
		slog.String("ns_end", vethNS),
		slog.String("namespace", nsName),
		slog.Time("timestamp", time.Now()),
	)

	if err := netlink.LinkAdd(veth); err != nil {
		m.logger.Error("failed to create veth pair",
			slog.String("host_end", vethHost),
			slog.String("ns_end", vethNS),
			slog.String("error", err.Error()),
			slog.Time("timestamp", time.Now()),
		)
		return fmt.Errorf("failed to create veth pair: %w", err)
	}

	m.logger.Info("veth pair created successfully",
		slog.String("host_end", vethHost),
		slog.String("ns_end", vethNS),
		slog.Time("timestamp", time.Now()),
	)

	// AIDEV-NOTE: Ensure cleanup on any error after veth creation
	cleanupVeth := true
	defer func() {
		if cleanupVeth {
			if link, err := netlink.LinkByName(vethHost); err == nil {
				if delErr := netlink.LinkDel(link); delErr != nil {
					m.logger.Warn("Failed to cleanup veth pair on error", "device", vethHost, "error", delErr)
				}
			}
		}
	}()

	// Get the namespace
	ns, err := netns.GetFromName(nsName)
	if err != nil {
		// Clean up veth pair
		if vethLink, err2 := netlink.LinkByName(vethHost); err2 == nil {
			if delErr := netlink.LinkDel(vethLink); delErr != nil {
				m.logger.Warn("Failed to cleanup veth link", "link", vethHost, "error", delErr)
			}
		}
		return fmt.Errorf("failed to get namespace: %w", err)
	}
	defer ns.Close()

	// Move veth peer to namespace
	// Sometimes the link takes a moment to appear, retry a few times
	m.logger.Info("looking for veth peer to move to namespace",
		slog.String("device", vethNS),
		slog.Time("timestamp", time.Now()),
	)

	var vethNSLink netlink.Link
	for i := 0; i < 3; i++ {
		vethNSLink, err = netlink.LinkByName(vethNS)
		if err == nil {
			m.logger.Info("found veth peer",
				slog.String("device", vethNS),
				slog.Int("attempt", i+1),
				slog.Time("timestamp", time.Now()),
			)
			break
		}
		m.logger.Warn("veth peer not found, retrying",
			slog.String("device", vethNS),
			slog.Int("attempt", i+1),
			slog.String("error", err.Error()),
			slog.Time("timestamp", time.Now()),
		)
		if i < 2 {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if err != nil {
		// Clean up veth pair
		if vethLink, err2 := netlink.LinkByName(vethHost); err2 == nil {
			if delErr := netlink.LinkDel(vethLink); delErr != nil {
				m.logger.Warn("Failed to cleanup veth link", "link", vethHost, "error", delErr)
			}
		}
		return fmt.Errorf("failed to get veth peer %s: %w", vethNS, err)
	}

	// Check if both veth ends exist before moving
	hostLink, err := netlink.LinkByName(vethHost)
	if err != nil {
		m.logger.Error("veth host side missing before move",
			slog.String("device", vethHost),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("veth host side missing: %w", err)
	}
	m.logger.Debug("veth host side exists before move",
		slog.String("device", vethHost),
		slog.Int("index", hostLink.Attrs().Index),
	)

	m.logger.Info("moving veth to namespace",
		slog.String("device", vethNS),
		slog.String("namespace", nsName),
		slog.Time("timestamp", time.Now()),
	)

	if err := netlink.LinkSetNsFd(vethNSLink, int(ns)); err != nil {
		m.logger.Error("failed to move veth to namespace",
			slog.String("device", vethNS),
			slog.String("namespace", nsName),
			slog.String("error", err.Error()),
			slog.Time("timestamp", time.Now()),
		)
		// Clean up veth pair
		if vethLink, err2 := netlink.LinkByName(vethHost); err2 == nil {
			if delErr := netlink.LinkDel(vethLink); delErr != nil {
				m.logger.Warn("Failed to cleanup veth link", "link", vethHost, "error", delErr)
			}
		}
		return fmt.Errorf("failed to move veth to namespace: %w", err)
	}

	m.logger.Info("veth moved to namespace successfully",
		slog.String("device", vethNS),
		slog.String("namespace", nsName),
		slog.Time("timestamp", time.Now()),
	)

	// Check if host side still exists after move
	if _, err := netlink.LinkByName(vethHost); err != nil {
		m.logger.Error("veth host side disappeared after moving peer to namespace!",
			slog.String("device", vethHost),
			slog.String("error", err.Error()),
		)
		// List all interfaces to debug
		links, _ := netlink.LinkList()
		linkNames := make([]string, 0, len(links))
		for _, link := range links {
			linkNames = append(linkNames, link.Attrs().Name)
		}
		m.logger.Error("available interfaces after move",
			slog.Any("interfaces", linkNames),
		)
		return fmt.Errorf("veth host side disappeared: %w", err)
	}

	// Attach host end to bridge
	m.logger.Info("attaching veth to bridge",
		slog.String("veth", vethHost),
		slog.String("bridge", m.config.BridgeName),
		slog.Time("timestamp", time.Now()),
	)

	// List all interfaces before trying to get veth host
	beforeLinks, _ := netlink.LinkList()
	beforeNames := make([]string, 0, len(beforeLinks))
	for _, link := range beforeLinks {
		beforeNames = append(beforeNames, link.Attrs().Name)
	}
	m.logger.Debug("interfaces before getting veth host",
		slog.Any("interfaces", beforeNames),
	)

	vethHostLink, err2 := netlink.LinkByName(vethHost)
	if err2 != nil {
		m.logger.Error("failed to get veth host",
			slog.String("device", vethHost),
			slog.String("error", err2.Error()),
			slog.Time("timestamp", time.Now()),
		)
		return fmt.Errorf("failed to get veth host: %w", err2)
	}

	// Check that bridge exists (but don't attach yet)
	if _, err2 := netlink.LinkByName(m.config.BridgeName); err2 != nil {
		// AIDEV-BUSINESS_RULE: CRITICAL RELIABILITY - Fail gracefully with actionable guidance
		// Never auto-modify host networking during VM operations to avoid masking issues

		// List all links for debugging
		links, _ := netlink.LinkList()
		linkNames := make([]string, 0, len(links))
		for _, link := range links {
			linkNames = append(linkNames, link.Attrs().Name)
		}

		m.logger.Error("CRITICAL: Bridge infrastructure missing - VM operations cannot proceed",
			slog.String("bridge", m.config.BridgeName),
			slog.String("error", err2.Error()),
			slog.Any("available_interfaces", linkNames),
			slog.String("action_required", "restart metald service to reinitialize bridge infrastructure"),
		)

		return fmt.Errorf("bridge infrastructure missing (%s) - this indicates the metald service needs to be restarted to reinitialize networking. Available interfaces: %v",
			m.config.BridgeName, linkNames)
	}

	// AIDEV-NOTE: CRITICAL FIX - Don't attach veth to bridge here
	// Bridge attachment will be handled later by attachVMToBridge() which properly
	// checks if the interface has an IP address (point-to-point mode vs bridge mode)
	m.logger.Info("skipping early bridge attachment - will be handled by attachVMToBridge()",
		slog.String("veth", vethHost),
		slog.String("bridge", m.config.BridgeName),
	)

	// Configure point-to-point IP on host veth
	// AIDEV-NOTE: CRITICAL FIX - Add point-to-point peer address to host veth
	// If VM gets x.x.x.10/30, host veth gets x.x.x.9/30
	hostIP := make(net.IP, len(ip))
	copy(hostIP, ip)
	hostIP[len(hostIP)-1] = ip[len(ip)-1] - 1 // Host peer is VM IP - 1

	hostAddr := &netlink.Addr{ //nolint:exhaustruct // Only setting IPNet field, other address fields use appropriate defaults
		IPNet: &net.IPNet{
			IP:   hostIP,
			Mask: net.CIDRMask(30, 32), // Use /30 for point-to-point addressing
		},
	}

	if err := netlink.AddrAdd(vethHostLink, hostAddr); err != nil {
		m.logger.Error("failed to add IP to host veth",
			slog.String("veth", vethHost),
			slog.String("ip", hostIP.String()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to add IP to host veth: %w", err)
	}

	m.logger.Info("configured point-to-point IP on host veth",
		slog.String("veth", vethHost),
		slog.String("host_ip", hostIP.String()),
		slog.String("vm_ip", ip.String()),
	)

	// Bring up the veth host interface
	if err := netlink.LinkSetUp(vethHostLink); err != nil {
		return fmt.Errorf("failed to bring up veth host: %w", err)
	}

	// Add host route for VM's point-to-point subnet to enable routing
	// AIDEV-NOTE: This route tells the host how to reach the VM via the veth interface
	vmSubnet := &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(30, 32),
	}
	hostRoute := &netlink.Route{ //nolint:exhaustruct // Only setting required fields for point-to-point route
		Dst:       vmSubnet,
		LinkIndex: vethHostLink.Attrs().Index,
	}
	if err := netlink.RouteAdd(hostRoute); err != nil && !strings.Contains(err.Error(), "exists") {
		m.logger.Warn("failed to add host route for VM",
			slog.String("vm_subnet", vmSubnet.String()),
			slog.String("veth", vethHost),
			slog.String("error", err.Error()),
		)
		// Non-fatal - continue anyway
	}

	m.logger.Info("added host route for VM point-to-point subnet",
		slog.String("vm_subnet", vmSubnet.String()),
		slog.String("veth", vethHost),
	)

	// AIDEV-NOTE: No additional NAT/forwarding rules needed for point-to-point setup
	// The host's existing routing and IP forwarding handles connectivity properly

	// Create TAP device in host namespace (so firecracker can access it)
	if err := m.createTAPDevice(deviceNames.TAP, mac); err != nil {
		return fmt.Errorf("failed to create TAP device: %w", err)
	}

	// Configure inside namespace
	if err := m.configureNamespace(ns, vethNS, ip, workspaceSubnet); err != nil {
		return err
	}

	// Success - don't cleanup veth
	cleanupVeth = false
	return nil
}

// createTAPDevice creates a TAP device in the host namespace
func (m *Manager) createTAPDevice(tapName, mac string) error {
	// Create TAP device
	tap := &netlink.Tuntap{ //nolint:exhaustruct // Only setting required fields, other tap fields use appropriate defaults
		LinkAttrs: netlink.LinkAttrs{ //nolint:exhaustruct // Only setting Name field, other link attributes use appropriate defaults
			Name: tapName,
		},
		Mode: netlink.TUNTAP_MODE_TAP,
	}

	m.logger.Info("creating TAP device",
		slog.String("tap", tapName),
		slog.Time("timestamp", time.Now()),
	)

	if err := netlink.LinkAdd(tap); err != nil {
		m.logger.Error("failed to create tap device",
			slog.String("tap", tapName),
			slog.String("error", err.Error()),
			slog.Time("timestamp", time.Now()),
		)
		return fmt.Errorf("failed to create tap device: %w", err)
	}

	m.logger.Info("TAP device created successfully",
		slog.String("tap", tapName),
		slog.Time("timestamp", time.Now()),
	)

	// Set MAC address on TAP
	m.logger.Info("getting tap link to set MAC",
		slog.String("tap", tapName),
		slog.Time("timestamp", time.Now()),
	)

	tapLink, err := netlink.LinkByName(tapName)
	if err != nil {
		m.logger.Error("failed to get tap link",
			slog.String("tap", tapName),
			slog.String("error", err.Error()),
			slog.Time("timestamp", time.Now()),
		)
		return fmt.Errorf("failed to get tap link: %w", err)
	}

	hwAddr, _ := net.ParseMAC(mac)
	m.logger.Info("setting MAC on tap device",
		slog.String("tap", tapName),
		slog.String("mac", mac),
		slog.Time("timestamp", time.Now()),
	)

	if err := netlink.LinkSetHardwareAddr(tapLink, hwAddr); err != nil {
		m.logger.Error("failed to set MAC on tap device",
			slog.String("tap", tapName),
			slog.String("mac", mac),
			slog.String("error", err.Error()),
			slog.Time("timestamp", time.Now()),
		)
		return fmt.Errorf("failed to set MAC on tap device: %w", err)
	}

	// Bring TAP device up
	if err := netlink.LinkSetUp(tapLink); err != nil {
		m.logger.Error("failed to bring up tap device",
			slog.String("tap", tapName),
			slog.String("error", err.Error()),
			slog.Time("timestamp", time.Now()),
		)
		return fmt.Errorf("failed to bring up tap device: %w", err)
	}

	m.logger.Info("TAP device configured successfully",
		slog.String("tap", tapName),
		slog.String("mac", mac),
		slog.Time("timestamp", time.Now()),
	)

	return nil
}

// configureNamespace sets up networking inside the namespace (veth only)
func (m *Manager) configureNamespace(ns netns.NsHandle, vethName string, ip net.IP, workspaceSubnet string) error {
	m.logger.Debug("configuring namespace", "veth_name", vethName, "ip", ip.String(), "workspace_subnet", workspaceSubnet)

	// AIDEV-NOTE: CRITICAL FIX - Lock OS thread for namespace operations
	// Without this, Go scheduler can move goroutine to different OS thread,
	// causing namespace operations to affect wrong thread and break host networking
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Save current namespace
	origNS, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed to get current namespace: %w", err)
	}
	defer origNS.Close()

	// Switch to target namespace
	if setErr := netns.Set(ns); setErr != nil {
		return fmt.Errorf("failed to set namespace: %w", setErr)
	}
	defer func() {
		if setErr := netns.Set(origNS); setErr != nil {
			slog.Error("Failed to restore namespace", "error", setErr)
		}
	}()

	// Get veth link
	vethLink, err := netlink.LinkByName(vethName)
	if err != nil {
		return fmt.Errorf("failed to get veth link: %w", err)
	}

	// AIDEV-NOTE: Simplified networking - no bridge needed inside namespace
	// The veth device will handle routing between host and VM
	// The TAP device is created in the host namespace for firecracker access

	// Bring up veth interface
	if linkUpErr := netlink.LinkSetUp(vethLink); linkUpErr != nil {
		return fmt.Errorf("failed to bring up veth: %w", linkUpErr)
	}

	// Add IP to veth interface using point-to-point addressing
	// AIDEV-NOTE: CRITICAL FIX - Use /30 point-to-point subnet instead of /24 bridge subnet
	// This fixes L2 connectivity issues by properly configuring veth as a routed link
	// VM gets x.x.x.10/30, host veth will get x.x.x.9/30 as the point-to-point peer
	addr := &netlink.Addr{ //nolint:exhaustruct // Only setting IPNet field, other address fields use appropriate defaults
		IPNet: &net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(30, 32), // Use /30 for point-to-point addressing
		},
	}

	// AIDEV-NOTE: Retry adding IP to handle race conditions where veth might not be immediately ready
	var addErr error
	for i := 0; i < 5; i++ {
		addErr = netlink.AddrAdd(vethLink, addr)
		if addErr == nil {
			break
		}

		// Check if it's a "no such device" error specifically
		if strings.Contains(addErr.Error(), "no such device") {
			m.logger.Warn("veth device not ready for IP assignment, retrying",
				slog.String("veth", vethName),
				slog.Int("attempt", i+1),
				slog.String("error", addErr.Error()),
			)
			time.Sleep(50 * time.Millisecond)

			// Re-get the veth link in the current namespace context (we're already in the target namespace)
			vethLink, err = netlink.LinkByName(vethName)
			if err != nil {
				// Log available interfaces for debugging
				if links, listErr := netlink.LinkList(); listErr == nil {
					var linkNames []string
					for _, link := range links {
						linkNames = append(linkNames, link.Attrs().Name)
					}
					m.logger.Error("available interfaces in namespace during retry",
						slog.String("veth", vethName),
						slog.Int("attempt", i+1),
						slog.Any("interfaces", linkNames),
					)
				}
				return fmt.Errorf("failed to re-get veth link on retry %d: %w", i+1, err)
			}
			continue
		}

		// For other errors, don't retry
		break
	}

	if addErr != nil {
		return fmt.Errorf("failed to add IP to veth after retries: %w", addErr)
	}

	// Enable proxy ARP on veth so it responds to ARP requests for the VM
	// AIDEV-NOTE: This is necessary when not using a bridge
	proxyARPPath := fmt.Sprintf("/proc/sys/net/ipv4/conf/%s/proxy_arp", vethName)
	if err := os.WriteFile(proxyARPPath, []byte("1\n"), 0600); err != nil {
		m.logger.Warn("failed to enable proxy ARP on veth",
			slog.String("veth", vethName),
			slog.String("error", err.Error()),
		)
		// Non-fatal - continue anyway
	}

	// Add default route using point-to-point gateway
	// AIDEV-NOTE: CRITICAL FIX - Use point-to-point peer address as gateway
	// For point-to-point /30 subnet, if VM is x.x.x.10, peer (gateway) is x.x.x.9
	gateway := make(net.IP, len(ip))
	copy(gateway, ip)
	gateway[len(gateway)-1] = ip[len(ip)-1] - 1 // Peer address is VM IP - 1

	route := &netlink.Route{ //nolint:exhaustruct // Only setting Dst and Gw fields for default route, other route fields use appropriate defaults
		Dst: nil, // default route
		Gw:  gateway,
	}
	if err := netlink.RouteAdd(route); err != nil && !strings.Contains(err.Error(), "exists") {
		return fmt.Errorf("failed to add default route: %w", err)
	}

	return nil
}

// applyRateLimit applies traffic shaping to the interface
//
//nolint:unused // Reserved for future rate limiting implementation
func (m *Manager) applyRateLimit(link netlink.Link, mbps int) {
	// Use tc (traffic control) to limit bandwidth
	// This is a simplified example - production would use netlink directly

	// Validate interface name to prevent command injection
	ifaceName := link.Attrs().Name
	if !isValidInterfaceName(ifaceName) {
		m.logger.Error("invalid interface name",
			slog.String("interface", ifaceName),
		)
		return
	}

	// Delete any existing qdisc (ignore errors as it might not exist)
	_ = exec.Command("tc", "qdisc", "del", "dev", ifaceName, "root").Run() //nolint:gosec // Interface name validated

	// Add HTB qdisc
	cmd := exec.Command("tc", "qdisc", "add", "dev", ifaceName, "root", "handle", "1:", "htb") //nolint:gosec // Interface name validated
	if err := cmd.Run(); err != nil {
		m.logger.Warn("failed to add HTB qdisc",
			slog.String("interface", ifaceName),
			slog.String("error", err.Error()),
		)
		return // Non-fatal
	}

	// Add rate limit class
	rate := fmt.Sprintf("%dmbit", mbps)
	cmd = exec.Command("tc", "class", "add", "dev", ifaceName,
		"parent", "1:", "classid", "1:1", "htb", "rate", rate) //nolint:gosec // Interface name validated
	if err := cmd.Run(); err != nil {
		m.logger.Warn("failed to add rate limit",
			slog.String("interface", ifaceName),
			slog.String("error", err.Error()),
		)
	}
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

// verifyNetworkCleanup verifies that all network resources for a VM have been properly cleaned up
func (m *Manager) verifyNetworkCleanup(ctx context.Context, vmID string, deviceNames *NetworkDeviceNames) error {
	var remainingResources []string

	// Check if TAP device still exists
	if _, err := netlink.LinkByName(deviceNames.TAP); err == nil {
		remainingResources = append(remainingResources, fmt.Sprintf("TAP device: %s", deviceNames.TAP))
	}

	// Check if veth host device still exists
	if _, err := netlink.LinkByName(deviceNames.VethHost); err == nil {
		remainingResources = append(remainingResources, fmt.Sprintf("veth device: %s", deviceNames.VethHost))
	}

	// Check if namespace still exists
	if m.namespaceExists(deviceNames.Namespace) {
		remainingResources = append(remainingResources, fmt.Sprintf("namespace: %s", deviceNames.Namespace))
	}

	if len(remainingResources) > 0 {
		m.logger.WarnContext(ctx, "Cleanup verification detected remaining resources",
			"vm_id", vmID,
			"remaining_resources", remainingResources,
		)
		return fmt.Errorf("cleanup incomplete: %d resources remain: %v", len(remainingResources), remainingResources)
	}

	m.logger.InfoContext(ctx, "Network cleanup verification passed", "vm_id", vmID)
	return nil
}

// namespaceExists checks if a network namespace exists
func (m *Manager) namespaceExists(namespace string) bool {
	// Try to get the namespace - if it exists, this won't error
	if _, err := netns.GetFromName(namespace); err != nil {
		return false
	}
	return true
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

// Shutdown cleans up all networking resources
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.InfoContext(ctx, "shutting down network manager")
	m.logNetworkState("before shutdown")

	// AIDEV-NOTE: CRITICAL FIX - Preserve IP allocation state during service restarts
	// Only clean up physical resources, not network state, to prevent IP allocation corruption
	vmCount := len(m.vmNetworks)
	m.logger.InfoContext(ctx, "preserving VM network state during shutdown",
		slog.Int("count", vmCount),
		slog.String("reason", "service restart should not affect IP allocation"),
	)

	// Note: We intentionally do NOT call DeleteVMNetwork here during shutdown
	// DeleteVMNetwork would decrement VMCount in MultiBridgeManager, corrupting IP allocation state
	// Physical cleanup (namespaces, interfaces) will be handled by the OS or next service start

	// Clean up iptables rules
	m.logger.InfoContext(ctx, "cleaning up iptables rules",
		slog.Int("rule_count", len(m.iptablesRules)),
	)
	m.cleanupIPTables()

	// AIDEV-NOTE: We intentionally keep the bridge to avoid network disruption
	// Deleting the bridge can cause host network issues if there are dependencies
	m.logger.InfoContext(ctx, "keeping bridge intact to avoid network disruption",
		slog.String("bridge", m.config.BridgeName),
		slog.Bool("bridge_created", m.bridgeCreated),
	)

	m.logNetworkState("after shutdown")
	m.logger.InfoContext(ctx, "network manager shutdown complete")

	return nil
}

// Helper functions

func (m *Manager) createNamespace(name string) error {
	// Check if namespace already exists
	if _, err := netns.GetFromName(name); err == nil {
		m.logger.Debug("namespace already exists", slog.String("namespace", name))
		return nil // Already exists
	}

	// Save current namespace to ensure we don't accidentally switch
	origNS, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed to get current namespace: %w", err)
	}
	defer origNS.Close()

	m.logger.Info("creating network namespace", slog.String("namespace", name))

	// Create new namespace
	newNS, err := netns.NewNamed(name)
	if err != nil {
		m.logger.Error("failed to create namespace",
			slog.String("namespace", name),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to create namespace %s: %w", name, err)
	}
	newNS.Close() // Close the handle immediately, we don't need it

	// Ensure we're back in the original namespace
	if err := netns.Set(origNS); err != nil {
		m.logger.Error("failed to restore original namespace after creation",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to restore namespace: %w", err)
	}

	m.logger.Info("namespace created successfully", slog.String("namespace", name))

	// Create namespace directory for runtime data
	nsDir := filepath.Join("/var/run/netns", name)
	if err := os.MkdirAll(filepath.Dir(nsDir), 0755); err != nil {
		return fmt.Errorf("failed to create namespace directory: %w", err)
	}

	return nil
}

func (m *Manager) deleteNamespace(name string) {
	m.logger.Info("deleting network namespace", "namespace", name)
	if err := netns.DeleteNamed(name); err != nil {
		m.logger.Error("Failed to delete namespace - manual cleanup may be required",
			"namespace", name, "error", err,
			"cleanup_command", fmt.Sprintf("sudo ip netns delete %s", name))

		// Record namespace deletion failure metric
		if m.metrics != nil {
			m.metrics.RecordNamespaceDeletionFailure(context.Background(), name, err.Error())
		}

		// AIDEV-NOTE: Network namespace deletion can fail if interfaces are busy
		// This leaves orphaned namespaces that can cause IP conflicts
		// Operators should monitor for this error and clean up manually if needed
	} else {
		m.logger.Info("successfully deleted network namespace", "namespace", name)
	}
}

func (m *Manager) cleanupIPTables() {
	m.logger.Info("starting iptables cleanup",
		slog.Int("rules_to_remove", len(m.iptablesRules)),
	)

	// Remove our iptables rules in reverse order
	for i := len(m.iptablesRules) - 1; i >= 0; i-- {
		rule := m.iptablesRules[i]
		// Convert -A to -D to delete the rule
		deleteRule := strings.Replace(rule, "-A", "-D", 1)
		args := strings.Fields(deleteRule)

		m.logger.Info("removing iptables rule",
			slog.Int("rule_index", i),
			slog.String("original_rule", rule),
			slog.String("delete_command", strings.Join(args, " ")),
		)

		cmd := exec.Command("iptables", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			m.logger.Warn("failed to remove iptables rule",
				slog.String("rule", rule),
				slog.String("error", err.Error()),
				slog.String("output", string(output)),
			)
		} else {
			m.logger.Info("iptables rule removed successfully",
				slog.String("rule", rule),
			)
		}
	}
	m.iptablesRules = nil
	m.logger.Info("iptables cleanup completed")
}

// GetNetworkStats returns network statistics for a VM
func (m *Manager) GetNetworkStats(vmID string) (*NetworkStats, error) {
	// AIDEV-NOTE: CRITICAL FIX - Lock OS thread for namespace operations
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	m.mu.RLock()
	vmNet, exists := m.vmNetworks[vmID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("network not found for VM %s", vmID)
	}

	// Get stats from the TAP device in the namespace
	ns, err := netns.GetFromName(vmNet.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}
	defer ns.Close()

	origNS, err := netns.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get current namespace: %w", err)
	}
	defer origNS.Close()

	if setErr := netns.Set(ns); setErr != nil {
		return nil, fmt.Errorf("failed to set namespace: %w", setErr)
	}
	defer func() {
		if setErr := netns.Set(origNS); setErr != nil {
			slog.Error("Failed to restore namespace", "error", setErr)
		}
	}()

	// Get TAP device stats
	link, err := netlink.LinkByName(vmNet.TapDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to get tap device: %w", err)
	}

	stats := link.Attrs().Statistics
	if stats == nil {
		return nil, fmt.Errorf("no statistics available")
	}

	return &NetworkStats{
		RxBytes:   stats.RxBytes,
		TxBytes:   stats.TxBytes,
		RxPackets: stats.RxPackets,
		TxPackets: stats.TxPackets,
		RxDropped: stats.RxDropped,
		TxDropped: stats.TxDropped,
		RxErrors:  stats.RxErrors,
		TxErrors:  stats.TxErrors,
	}, nil
}

// isValidInterfaceName validates that an interface name is safe to use in commands
//
//nolint:unused // Used by applyRateLimit function which is reserved for future implementation
func isValidInterfaceName(name string) bool {
	// Linux interface names must be 1-15 characters
	if len(name) == 0 || len(name) > 15 {
		return false
	}

	// Must contain only alphanumeric, dash, underscore, or dot
	for _, ch := range name {
		if (ch < 'a' || ch > 'z') &&
			(ch < 'A' || ch > 'Z') &&
			(ch < '0' || ch > '9') &&
			ch != '-' && ch != '_' && ch != '.' {
			return false
		}
	}

	return true
}

// logNetworkState logs the current state of network interfaces and routes
func (m *Manager) logNetworkState(context string) {
	m.logger.Info("network state check",
		slog.String("context", context),
	)

	// Check bridge state
	if link, err := netlink.LinkByName(m.config.BridgeName); err == nil {
		addrs, _ := netlink.AddrList(link, netlink.FAMILY_V4)
		var addrStrs []string
		for _, addr := range addrs {
			addrStrs = append(addrStrs, addr.IPNet.String())
		}
		m.logger.Info("bridge state",
			slog.String("bridge", m.config.BridgeName),
			slog.String("state", link.Attrs().OperState.String()),
			slog.String("flags", link.Attrs().Flags.String()),
			slog.Any("addresses", addrStrs),
		)
	} else {
		m.logger.Info("bridge not found",
			slog.String("bridge", m.config.BridgeName),
			slog.String("error", err.Error()),
		)
	}

	// List all interfaces
	links, err := netlink.LinkList()
	if err == nil {
		var interfaces []string
		for _, link := range links {
			interfaces = append(interfaces, fmt.Sprintf("%s(%s)", link.Attrs().Name, link.Attrs().OperState.String()))
		}
		m.logger.Info("all interfaces",
			slog.Any("interfaces", interfaces),
		)
	}

	// Check default route
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err == nil {
		for _, route := range routes {
			if route.Dst == nil {
				if link, err := netlink.LinkByIndex(route.LinkIndex); err == nil {
					m.logger.Info("default route",
						slog.String("interface", link.Attrs().Name),
						slog.String("gateway", route.Gw.String()),
					)
				}
			}
		}
	}
}

// AIDEV-NOTE: Port management methods for container-like networking

// AllocatePortsForVM allocates host ports for container ports based on metadata
func (m *Manager) AllocatePortsForVM(vmID string, exposedPorts []string) ([]PortMapping, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var mappings []PortMapping

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
			// Clean up any already allocated ports
			m.releaseVMPortsLocked(vmID)
			return nil, fmt.Errorf("failed to allocate port %s for VM %s: %w", portSpec, vmID, err)
		}

		mapping := PortMapping{
			ContainerPort: containerPort,
			HostPort:      hostPort,
			Protocol:      protocol,
			VMID:          vmID,
		}
		mappings = append(mappings, mapping)

		m.logger.Info("allocated port mapping",
			slog.String("vm_id", vmID),
			slog.Int("container_port", containerPort),
			slog.Int("host_port", hostPort),
			slog.String("protocol", protocol),
		)
	}

	return mappings, nil
}

// ReleaseVMPorts releases all ports allocated to a VM
func (m *Manager) ReleaseVMPorts(vmID string) []PortMapping {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.releaseVMPortsLocked(vmID)
}

// releaseVMPortsLocked releases VM ports with lock already held
func (m *Manager) releaseVMPortsLocked(vmID string) []PortMapping {
	mappings := m.portAllocator.ReleaseVMPorts(vmID)

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

// CleanupOrphanedResources performs administrative cleanup of orphaned network resources
// This function scans for and removes network interfaces that are no longer associated with active VMs
func (m *Manager) CleanupOrphanedResources(ctx context.Context, dryRun bool) (*CleanupReport, error) {
	m.logger.InfoContext(ctx, "starting orphaned resource cleanup",
		slog.Bool("dry_run", dryRun),
	)

	report := &CleanupReport{
		DryRun: dryRun,
	}

	// Get all network links
	links, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %w", err)
	}

	// Find orphaned TAP devices
	for _, link := range links {
		name := link.Attrs().Name
		if strings.HasPrefix(name, "tap_") && len(name) == 12 { // tap_<8-char-id>
			networkID := name[4:] // Extract the 8-char ID
			if !m.isNetworkIDActive(networkID) {
				report.OrphanedTAPs = append(report.OrphanedTAPs, name)
				if !dryRun {
					if delErr := netlink.LinkDel(link); delErr != nil {
						report.Errors = append(report.Errors, fmt.Sprintf("Failed to delete TAP %s: %v", name, delErr))
					} else {
						report.CleanedTAPs = append(report.CleanedTAPs, name)
					}
				}
			}
		}
	}

	// Find orphaned veth pairs
	for _, link := range links {
		name := link.Attrs().Name
		if strings.HasPrefix(name, "vh_") && len(name) == 11 { // vh_<8-char-id>
			networkID := name[3:] // Extract the 8-char ID
			if !m.isNetworkIDActive(networkID) {
				report.OrphanedVeths = append(report.OrphanedVeths, name)
				if !dryRun {
					if delErr := netlink.LinkDel(link); delErr != nil {
						report.Errors = append(report.Errors, fmt.Sprintf("Failed to delete veth %s: %v", name, delErr))
					} else {
						report.CleanedVeths = append(report.CleanedVeths, name)
					}
				}
			}
		}
	}

	// Find orphaned namespaces
	// Note: This is a simplified check - in practice you'd scan /var/run/netns or use netns.ListNamed()
	for vmID := range m.vmNetworks {
		expectedNS := fmt.Sprintf("vm-%s", vmID)
		if m.namespaceExists(expectedNS) {
			// This namespace should exist, it's not orphaned
			continue
		}
	}

	m.logger.InfoContext(ctx, "orphaned resource cleanup completed",
		slog.Bool("dry_run", dryRun),
		slog.Int("orphaned_taps", len(report.OrphanedTAPs)),
		slog.Int("orphaned_veths", len(report.OrphanedVeths)),
		slog.Int("cleaned_taps", len(report.CleanedTAPs)),
		slog.Int("cleaned_veths", len(report.CleanedVeths)),
		slog.Int("errors", len(report.Errors)),
	)

	return report, nil
}

// isNetworkIDActive checks if a network ID is currently associated with an active VM
func (m *Manager) isNetworkIDActive(networkID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, vmNet := range m.vmNetworks {
		if vmNet.NetworkID == networkID {
			return true
		}
	}
	return false
}

// CleanupReport contains the results of orphaned resource cleanup
type CleanupReport struct {
	DryRun        bool
	OrphanedTAPs  []string
	OrphanedVeths []string
	OrphanedNS    []string
	CleanedTAPs   []string
	CleanedVeths  []string
	CleanedNS     []string
	Errors        []string
}

// GetBridgeCapacityStatus returns current bridge capacity and utilization
func (m *Manager) GetBridgeCapacityStatus() *BridgeCapacityStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bridgeStats := m.metrics.GetBridgeStats()
	alerts := m.metrics.GetBridgeCapacityAlerts()

	// Calculate overall statistics
	totalVMs := int64(0)
	totalCapacity := int64(0)
	bridgeCount := len(bridgeStats)

	bridgeDetails := make([]BridgeDetails, 0, bridgeCount)
	for _, stats := range bridgeStats {
		totalVMs += stats.VMCount
		totalCapacity += stats.MaxVMs

		utilization := float64(stats.VMCount) / float64(stats.MaxVMs)
		bridgeDetails = append(bridgeDetails, BridgeDetails{
			Name:         stats.BridgeName,
			VMCount:      stats.VMCount,
			MaxVMs:       stats.MaxVMs,
			Utilization:  utilization,
			IsHealthy:    stats.IsHealthy,
			CreatedAt:    stats.CreatedAt,
			LastActivity: stats.LastActivity,
		})
	}

	overallUtilization := float64(0)
	if totalCapacity > 0 {
		overallUtilization = float64(totalVMs) / float64(totalCapacity)
	}

	return &BridgeCapacityStatus{
		TotalVMs:           totalVMs,
		TotalCapacity:      totalCapacity,
		OverallUtilization: overallUtilization,
		BridgeCount:        int64(bridgeCount),
		Bridges:            bridgeDetails,
		Alerts:             alerts,
		Timestamp:          time.Now(),
	}
}

// GetNetworkMetrics returns the network metrics instance for external access
func (m *Manager) GetNetworkMetrics() *NetworkMetrics {
	return m.metrics
}

// BridgeCapacityStatus provides comprehensive bridge capacity information
type BridgeCapacityStatus struct {
	TotalVMs           int64                 `json:"total_vms"`
	TotalCapacity      int64                 `json:"total_capacity"`
	OverallUtilization float64               `json:"overall_utilization"`
	BridgeCount        int64                 `json:"bridge_count"`
	Bridges            []BridgeDetails       `json:"bridges"`
	Alerts             []BridgeCapacityAlert `json:"alerts"`
	Timestamp          time.Time             `json:"timestamp"`
}

// BridgeDetails provides detailed information about a specific bridge
type BridgeDetails struct {
	Name         string    `json:"name"`
	VMCount      int64     `json:"vm_count"`
	MaxVMs       int64     `json:"max_vms"`
	Utilization  float64   `json:"utilization"`
	IsHealthy    bool      `json:"is_healthy"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
}

// extractWorkspaceID extracts workspace_id from OpenTelemetry baggage context
// AIDEV-NOTE: This enables just-in-time workspace VLAN provisioning
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
