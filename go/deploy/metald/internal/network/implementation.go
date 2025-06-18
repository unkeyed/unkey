package network

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// Config holds network configuration
type Config struct {
	BridgeName      string // Default: "br-vms"
	BridgeIP        string // Default: "10.100.0.1/16"
	VMSubnet        string // Default: "10.100.0.0/16"
	EnableIPv6      bool
	DNSServers      []string // Default: ["8.8.8.8", "8.8.4.4"]
	EnableRateLimit bool
	RateLimitMbps   int // Per VM rate limit in Mbps
}

// DefaultConfig returns default network configuration
func DefaultConfig() *Config {
	return &Config{
		BridgeName:      "br-vms",
		BridgeIP:        "10.100.0.1/16",
		VMSubnet:        "10.100.0.0/16",
		DNSServers:      []string{"8.8.8.8", "8.8.4.4"},
		EnableRateLimit: true,
		RateLimitMbps:   100, // 100 Mbps default
	}
}

// Manager handles VM networking
type Manager struct {
	logger     *slog.Logger
	config     *Config
	allocator  *IPAllocator
	idGen      *IDGenerator
	mu         sync.RWMutex
	vmNetworks map[string]*VMNetwork

	// Runtime state
	bridgeCreated bool
	iptablesRules []string
}

// NewManager creates a new network manager
func NewManager(logger *slog.Logger, config *Config) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	logger = logger.With("component", "network-manager")
	logger.Info("creating network manager",
		slog.String("bridge_name", config.BridgeName),
		slog.String("bridge_ip", config.BridgeIP),
		slog.String("vm_subnet", config.VMSubnet),
	)

	_, subnet, err := net.ParseCIDR(config.VMSubnet)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet: %w", err)
	}

	m := &Manager{
		logger:     logger,
		config:     config,
		allocator:  NewIPAllocator(subnet),
		idGen:      NewIDGenerator(),
		vmNetworks: make(map[string]*VMNetwork),
	}

	// Log current network state before initialization
	m.logNetworkState("before initialization")

	// Initialize host networking
	if err := m.initializeHost(); err != nil {
		m.logger.Error("failed to initialize host networking",
			slog.String("error", err.Error()),
		)
		m.logNetworkState("after failed initialization")
		return nil, fmt.Errorf("failed to initialize host networking: %w", err)
	}

	// Log network state after initialization
	m.logNetworkState("after successful initialization")

	return m, nil
}

// initializeHost sets up the host networking infrastructure
func (m *Manager) initializeHost() error {
	m.logger.Info("starting host network initialization")
	
	// Enable IP forwarding using sysctl (now running as root)
	m.logger.Info("enabling IP forwarding")
	cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		m.logger.Error("failed to enable IP forwarding",
			slog.String("error", err.Error()),
			slog.String("output", string(output)),
		)
		return fmt.Errorf("failed to enable IP forwarding: %w", err)
	}
	
	// Make it persistent across reboots
	// AIDEV-NOTE: Creates sysctl config to persist IP forwarding
	sysctlConfig := []byte("# Enable IP forwarding for metald VM networking\nnet.ipv4.ip_forward = 1\n")
	sysctlPath := "/etc/sysctl.d/99-metald.conf"
	
	if err := os.WriteFile(sysctlPath, sysctlConfig, 0644); err != nil {
		m.logger.Warn("failed to create persistent sysctl config",
			slog.String("path", sysctlPath),
			slog.String("error", err.Error()),
		)
		// Not fatal - IP forwarding is enabled for this session
	}
	
	m.logger.Info("IP forwarding enabled successfully")

	// Create bridge if it doesn't exist
	if err := m.ensureBridge(); err != nil {
		return fmt.Errorf("failed to create bridge: %w", err)
	}

	// Setup NAT rules (best effort - may fail without root or if already configured)
	m.logNetworkState("before NAT setup")
	if err := m.setupNAT(); err != nil {
		m.logger.Warn("failed to setup NAT (may already be configured)",
			slog.String("error", err.Error()),
		)
		m.logNetworkState("after failed NAT setup")
		// Continue anyway - NAT might already be set up
	} else {
		m.logNetworkState("after successful NAT setup")
	}

	m.logger.Info("host networking initialized",
		slog.String("bridge", m.config.BridgeName),
		slog.String("subnet", m.config.VMSubnet),
	)

	return nil
}

// ensureBridge creates the bridge if it doesn't exist
func (m *Manager) ensureBridge() error {
	m.logger.Info("checking if bridge exists",
		slog.String("bridge", m.config.BridgeName),
	)
	
	// Check if bridge exists
	if link, err := netlink.LinkByName(m.config.BridgeName); err == nil {
		m.bridgeCreated = true
		m.logger.Info("bridge already exists",
			slog.String("bridge", m.config.BridgeName),
			slog.String("type", link.Type()),
			slog.String("state", link.Attrs().OperState.String()),
		)
		return nil // Bridge already exists
	} else {
		m.logger.Info("bridge does not exist, will create",
			slog.String("bridge", m.config.BridgeName),
			slog.String("error", err.Error()),
		)
	}

	// Create bridge
	m.logger.Info("creating new bridge",
		slog.String("bridge", m.config.BridgeName),
	)
	
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: m.config.BridgeName,
		},
	}

	m.logger.Info("CRITICAL: About to create bridge - network may be affected",
		slog.String("bridge", m.config.BridgeName),
	)
	
	if err := netlink.LinkAdd(bridge); err != nil {
		m.logger.Error("failed to create bridge",
			slog.String("bridge", m.config.BridgeName),
			slog.String("error", err.Error()),
		)
		m.logNetworkState("after failed bridge creation")
		return fmt.Errorf("failed to create bridge: %w", err)
	}
	m.logger.Info("bridge created successfully - checking network state",
		slog.String("bridge", m.config.BridgeName),
	)
	m.logNetworkState("immediately after bridge creation")

	// Get the created bridge
	br, err := netlink.LinkByName(m.config.BridgeName)
	if err != nil {
		return fmt.Errorf("failed to get bridge: %w", err)
	}

	// Add IP address to bridge
	m.logger.Info("parsing bridge IP address",
		slog.String("ip", m.config.BridgeIP),
	)
	addr, err := netlink.ParseAddr(m.config.BridgeIP)
	if err != nil {
		return fmt.Errorf("failed to parse bridge IP: %w", err)
	}

	m.logger.Info("adding IP address to bridge",
		slog.String("bridge", m.config.BridgeName),
		slog.String("ip", m.config.BridgeIP),
	)
	if err := netlink.AddrAdd(br, addr); err != nil {
		m.logger.Error("failed to add IP to bridge",
			slog.String("bridge", m.config.BridgeName),
			slog.String("ip", m.config.BridgeIP),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to add IP to bridge: %w", err)
	}
	m.logger.Info("IP address added to bridge successfully")

	// Bring bridge up
	m.logger.Info("bringing bridge up",
		slog.String("bridge", m.config.BridgeName),
	)
	if err := netlink.LinkSetUp(br); err != nil {
		m.logger.Error("failed to bring bridge up",
			slog.String("bridge", m.config.BridgeName),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to bring bridge up: %w", err)
	}
	m.logger.Info("bridge is now up",
		slog.String("bridge", m.config.BridgeName),
	)

	m.bridgeCreated = true
	return nil
}

// setupNAT configures iptables NAT rules
func (m *Manager) setupNAT() error {
	m.logger.Info("setting up NAT rules")
	
	// Get the default route interface
	m.logger.Info("listing routes to find default interface")
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		m.logger.Error("failed to list routes",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to list routes: %w", err)
	}
	m.logger.Info("found routes",
		slog.Int("count", len(routes)),
	)

	var defaultIface string
	for _, route := range routes {
		if route.Dst == nil { // Default route
			m.logger.Info("found default route",
				slog.Int("link_index", route.LinkIndex),
			)
			link, err := netlink.LinkByIndex(route.LinkIndex)
			if err == nil {
				defaultIface = link.Attrs().Name
				m.logger.Info("identified default interface",
					slog.String("interface", defaultIface),
					slog.String("type", link.Type()),
					slog.String("state", link.Attrs().OperState.String()),
				)
				break
			} else {
				m.logger.Warn("failed to get link for default route",
					slog.Int("link_index", route.LinkIndex),
					slog.String("error", err.Error()),
				)
			}
		}
	}

	if defaultIface == "" {
		m.logger.Error("could not find default route interface",
			slog.Int("routes_checked", len(routes)),
		)
		return fmt.Errorf("could not find default route interface")
	}

	// Setup NAT rules
	rules := [][]string{
		// Enable NAT for VM subnet
		{"-t", "nat", "-A", "POSTROUTING", "-s", m.config.VMSubnet, "-o", defaultIface, "-j", "MASQUERADE"},

		// Allow forwarding from bridge to external
		{"-A", "FORWARD", "-i", m.config.BridgeName, "-o", defaultIface, "-j", "ACCEPT"},

		// Allow established connections back
		{"-A", "FORWARD", "-i", defaultIface, "-o", m.config.BridgeName, "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT"},

		// Allow VM to VM communication
		{"-A", "FORWARD", "-i", m.config.BridgeName, "-o", m.config.BridgeName, "-j", "ACCEPT"},
	}

	for i, rule := range rules {
		ruleStr := strings.Join(rule, " ")
		m.logger.Info("adding iptables rule",
			slog.Int("rule_number", i+1),
			slog.String("rule", ruleStr),
		)
		
		cmd := exec.Command("iptables", rule...)
		if output, err := cmd.CombinedOutput(); err != nil {
			m.logger.Error("failed to add iptables rule",
				slog.String("rule", ruleStr),
				slog.String("error", err.Error()),
				slog.String("output", string(output)),
			)
			// Try to clean up on failure
			m.cleanupIPTables()
			return fmt.Errorf("failed to add iptables rule %v: %w", rule, err)
		}
		m.logger.Info("iptables rule added successfully",
			slog.String("rule", ruleStr),
		)
		m.iptablesRules = append(m.iptablesRules, ruleStr)
	}

	return nil
}

// CreateVMNetwork sets up networking for a VM
func (m *Manager) CreateVMNetwork(ctx context.Context, vmID string) (*VMNetwork, error) {
	// Default namespace name
	nsName := fmt.Sprintf("vm-%s", vmID)
	return m.CreateVMNetworkWithNamespace(ctx, vmID, nsName)
}

// CreateVMNetworkWithNamespace sets up networking for a VM with a specific namespace name
func (m *Manager) CreateVMNetworkWithNamespace(ctx context.Context, vmID, nsName string) (*VMNetwork, error) {
	m.logger.InfoContext(ctx, "creating VM network",
		slog.String("vm_id", vmID),
		slog.String("namespace", nsName),
	)
	m.logNetworkState("before VM network creation")
	
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
		return nil, fmt.Errorf("failed to generate network ID: %w", err)
	}

	// Generate device names using consistent naming convention
	deviceNames := GenerateDeviceNames(networkID)

	// Allocate IP address
	ip, err := m.allocator.AllocateIP()
	if err != nil {
		m.idGen.ReleaseID(networkID)
		return nil, fmt.Errorf("failed to allocate IP: %w", err)
	}

	// Generate MAC address
	mac := m.generateMAC(vmID)

	// Override namespace name if provided (e.g., by jailer)
	actualNsName := nsName
	if actualNsName == "" {
		actualNsName = deviceNames.Namespace
	}

	// Create network namespace if it doesn't exist
	// It might have been pre-created by the jailer
	if err := m.createNamespace(actualNsName); err != nil {
		m.allocator.ReleaseIP(ip)
		m.idGen.ReleaseID(networkID)
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	// Create TAP device and configure networking
	if err := m.setupVMNetworking(actualNsName, deviceNames, ip, mac); err != nil {
		m.allocator.ReleaseIP(ip)
		m.idGen.ReleaseID(networkID)
		m.deleteNamespace(actualNsName)
		return nil, fmt.Errorf("failed to setup VM networking: %w", err)
	}

	// Create VM network info
	_, subnet, _ := net.ParseCIDR(m.config.VMSubnet)
	gateway := make(net.IP, len(subnet.IP))
	copy(gateway, subnet.IP)
	gateway[len(gateway)-1] = 1

	vmNet := &VMNetwork{
		VMID:       vmID,
		NetworkID:  networkID,
		Namespace:  actualNsName,
		TapDevice:  deviceNames.TAP,
		IPAddress:  ip,
		Netmask:    net.IPv4Mask(255, 255, 0, 0), // /16 to match subnet
		Gateway:    gateway,
		MacAddress: mac,
		DNSServers: m.config.DNSServers,
		CreatedAt:  time.Now(),
	}

	m.vmNetworks[vmID] = vmNet

	m.logger.InfoContext(ctx, "created VM network",
		slog.String("vm_id", vmID),
		slog.String("ip", ip.String()),
		slog.String("mac", mac),
		slog.String("tap", deviceNames.TAP),
		slog.String("namespace", actualNsName),
		slog.String("network_id", networkID),
	)

	return vmNet, nil
}

// setupVMNetworking configures the network namespace and TAP device
func (m *Manager) setupVMNetworking(nsName string, deviceNames *NetworkDeviceNames, ip net.IP, mac string) error {
	// AIDEV-NOTE: Now running as root, no need for nsenter workarounds

	// Use device names from the consistent naming convention
	vethHost := deviceNames.VethHost
	vethNS := deviceNames.VethNS

	// Create veth pair using netlink (preferred when running as root)
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{Name: vethHost},
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

	bridge, err2 := netlink.LinkByName(m.config.BridgeName)
	if err2 != nil {
		m.logger.Error("failed to get bridge",
			slog.String("bridge", m.config.BridgeName),
			slog.String("error", err2.Error()),
			slog.Time("timestamp", time.Now()),
		)
		// List all links to debug
		links, _ := netlink.LinkList()
		linkNames := make([]string, 0, len(links))
		for _, link := range links {
			linkNames = append(linkNames, link.Attrs().Name)
		}
		m.logger.Error("available network interfaces",
			slog.Any("interfaces", linkNames),
			slog.Time("timestamp", time.Now()),
		)
		return fmt.Errorf("failed to get bridge: %w", err2)
	}

	if err2 := netlink.LinkSetMaster(vethHostLink, bridge); err2 != nil {
		m.logger.Error("failed to attach veth to bridge",
			slog.String("veth", vethHost),
			slog.String("bridge", m.config.BridgeName),
			slog.String("error", err2.Error()),
			slog.Time("timestamp", time.Now()),
		)
		return fmt.Errorf("failed to attach veth to bridge: %w", err2)
	}

	m.logger.Info("veth attached to bridge successfully",
		slog.String("veth", vethHost),
		slog.String("bridge", m.config.BridgeName),
		slog.Time("timestamp", time.Now()),
	)

	// Bring up the veth host interface
	if err := netlink.LinkSetUp(vethHostLink); err != nil {
		return fmt.Errorf("failed to bring up veth host: %w", err)
	}

	// Create TAP device in host namespace (so firecracker can access it)
	if err := m.createTAPDevice(deviceNames.TAP, mac); err != nil {
		return fmt.Errorf("failed to create TAP device: %w", err)
	}

	// Configure inside namespace
	if err := m.configureNamespace(ns, vethNS, ip); err != nil {
		return err
	}

	// Success - don't cleanup veth
	cleanupVeth = false
	return nil
}

// createTAPDevice creates a TAP device in the host namespace
func (m *Manager) createTAPDevice(tapName, mac string) error {
	// Create TAP device
	tap := &netlink.Tuntap{
		LinkAttrs: netlink.LinkAttrs{
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
func (m *Manager) configureNamespace(ns netns.NsHandle, vethName string, ip net.IP) error {
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
	if err := netlink.LinkSetUp(vethLink); err != nil {
		return fmt.Errorf("failed to bring up veth: %w", err)
	}

	// Add IP directly to veth interface
	// AIDEV-NOTE: The veth acts as the default gateway for the VM
	// Using /16 to match the host bridge subnet
	addr := &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(16, 32), // Use /16 to match the bridge subnet
		},
	}
	if err := netlink.AddrAdd(vethLink, addr); err != nil {
		return fmt.Errorf("failed to add IP to veth: %w", err)
	}

	// Enable proxy ARP on veth so it responds to ARP requests for the VM
	// AIDEV-NOTE: This is necessary when not using a bridge
	proxyARPPath := fmt.Sprintf("/proc/sys/net/ipv4/conf/%s/proxy_arp", vethName)
	if err := os.WriteFile(proxyARPPath, []byte("1\n"), 0644); err != nil {
		m.logger.Warn("failed to enable proxy ARP on veth",
			slog.String("veth", vethName),
			slog.String("error", err.Error()),
		)
		// Non-fatal - continue anyway
	}

	// Add default route
	_, subnet, _ := net.ParseCIDR(m.config.VMSubnet)
	gateway := make(net.IP, len(subnet.IP))
	copy(gateway, subnet.IP)
	gateway[len(gateway)-1] = 1

	route := &netlink.Route{
		Dst: nil, // default route
		Gw:  gateway,
	}
	if err := netlink.RouteAdd(route); err != nil && !strings.Contains(err.Error(), "exists") {
		return fmt.Errorf("failed to add default route: %w", err)
	}

	return nil
}

// applyRateLimit applies traffic shaping to the interface
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

	// Release IP
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
		}
	}

	// Release the network ID for reuse
	m.idGen.ReleaseID(vmNet.NetworkID)

	delete(m.vmNetworks, vmID)

	m.logger.InfoContext(ctx, "deleted VM network",
		slog.String("vm_id", vmID),
		slog.String("network_id", vmNet.NetworkID),
		slog.String("ip", vmNet.IPAddress.String()),
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

// Shutdown cleans up all networking resources
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.InfoContext(ctx, "shutting down network manager")
	m.logNetworkState("before shutdown")

	// Delete all VM networks
	vmCount := len(m.vmNetworks)
	m.logger.InfoContext(ctx, "cleaning up VM networks",
		slog.Int("count", vmCount),
	)
	for vmID := range m.vmNetworks {
		if err := m.DeleteVMNetwork(ctx, vmID); err != nil {
			m.logger.ErrorContext(ctx, "failed to delete VM network during shutdown",
				slog.String("vm_id", vmID),
				slog.String("error", err.Error()),
			)
		}
	}

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
	if err := netns.DeleteNamed(name); err != nil {
		m.logger.Warn("Failed to delete namespace", "namespace", name, "error", err)
	}
}

func (m *Manager) generateMAC(vmID string) string {
	// Generate deterministic MAC from VM ID
	h := fnv.New32a()
	h.Write([]byte(vmID))
	hash := h.Sum32()

	// Use locally administered MAC prefix (02:xx:xx:xx:xx:xx)
	return fmt.Sprintf("02:00:%02x:%02x:%02x:%02x",
		(hash>>24)&0xff,
		(hash>>16)&0xff,
		(hash>>8)&0xff,
		hash&0xff,
	)
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
