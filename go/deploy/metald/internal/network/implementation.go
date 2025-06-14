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
	BridgeName     string // Default: "br-vms"
	BridgeIP       string // Default: "10.100.0.1/16"
	VMSubnet       string // Default: "10.100.0.0/16"
	EnableIPv6     bool
	DNSServers     []string // Default: ["8.8.8.8", "8.8.4.4"]
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
	
	_, subnet, err := net.ParseCIDR(config.VMSubnet)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet: %w", err)
	}
	
	m := &Manager{
		logger:     logger.With("component", "network-manager"),
		config:     config,
		allocator:  NewIPAllocator(subnet),
		vmNetworks: make(map[string]*VMNetwork),
	}
	
	// Initialize host networking
	if err := m.initializeHost(); err != nil {
		return nil, fmt.Errorf("failed to initialize host networking: %w", err)
	}
	
	return m, nil
}

// initializeHost sets up the host networking infrastructure
func (m *Manager) initializeHost() error {
	// Enable IP forwarding (best effort - may fail without root)
	if err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1\n"), 0644); err != nil {
		m.logger.Warn("failed to enable IP forwarding (may already be enabled)",
			slog.String("error", err.Error()),
		)
		// Continue anyway - IP forwarding might already be enabled
	}
	
	// Create bridge if it doesn't exist
	if err := m.ensureBridge(); err != nil {
		return fmt.Errorf("failed to create bridge: %w", err)
	}
	
	// Setup NAT rules (best effort - may fail without root or if already configured)
	if err := m.setupNAT(); err != nil {
		m.logger.Warn("failed to setup NAT (may already be configured)",
			slog.String("error", err.Error()),
		)
		// Continue anyway - NAT might already be set up
	}
	
	m.logger.Info("host networking initialized",
		slog.String("bridge", m.config.BridgeName),
		slog.String("subnet", m.config.VMSubnet),
	)
	
	return nil
}

// ensureBridge creates the bridge if it doesn't exist
func (m *Manager) ensureBridge() error {
	// Check if bridge exists
	if _, err := netlink.LinkByName(m.config.BridgeName); err == nil {
		m.bridgeCreated = true
		m.logger.Info("bridge already exists",
			slog.String("bridge", m.config.BridgeName),
		)
		return nil // Bridge already exists
	}
	
	// Create bridge
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: m.config.BridgeName,
		},
	}
	
	if err := netlink.LinkAdd(bridge); err != nil {
		return fmt.Errorf("failed to create bridge: %w", err)
	}
	
	// Get the created bridge
	br, err := netlink.LinkByName(m.config.BridgeName)
	if err != nil {
		return fmt.Errorf("failed to get bridge: %w", err)
	}
	
	// Add IP address to bridge
	addr, err := netlink.ParseAddr(m.config.BridgeIP)
	if err != nil {
		return fmt.Errorf("failed to parse bridge IP: %w", err)
	}
	
	if err := netlink.AddrAdd(br, addr); err != nil {
		return fmt.Errorf("failed to add IP to bridge: %w", err)
	}
	
	// Bring bridge up
	if err := netlink.LinkSetUp(br); err != nil {
		return fmt.Errorf("failed to bring bridge up: %w", err)
	}
	
	m.bridgeCreated = true
	return nil
}

// setupNAT configures iptables NAT rules
func (m *Manager) setupNAT() error {
	// Get the default route interface
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to list routes: %w", err)
	}
	
	var defaultIface string
	for _, route := range routes {
		if route.Dst == nil { // Default route
			link, err := netlink.LinkByIndex(route.LinkIndex)
			if err == nil {
				defaultIface = link.Attrs().Name
				break
			}
		}
	}
	
	if defaultIface == "" {
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
	
	for _, rule := range rules {
		cmd := exec.Command("iptables", rule...)
		if err := cmd.Run(); err != nil {
			// Try to clean up on failure
			m.cleanupIPTables()
			return fmt.Errorf("failed to add iptables rule %v: %w", rule, err)
		}
		m.iptablesRules = append(m.iptablesRules, strings.Join(rule, " "))
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
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if network already exists
	if existing, exists := m.vmNetworks[vmID]; exists {
		m.logger.Warn("VM network already exists",
			slog.String("vm_id", vmID),
			slog.String("ip", existing.IPAddress.String()),
		)
		return existing, nil
	}
	
	// Allocate IP address
	ip, err := m.allocator.AllocateIP()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate IP: %w", err)
	}
	
	// Generate MAC address
	mac := m.generateMAC(vmID)
	
	// Create network namespace if it doesn't exist
	// It might have been pre-created by the jailer
	if err := m.createNamespace(nsName); err != nil {
		m.allocator.ReleaseIP(ip)
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}
	
	// Create TAP device and configure networking
	tapName := fmt.Sprintf("tap%s", vmID[:8])
	if err := m.setupVMNetworking(nsName, tapName, ip, mac); err != nil {
		m.allocator.ReleaseIP(ip)
		m.deleteNamespace(nsName)
		return nil, fmt.Errorf("failed to setup VM networking: %w", err)
	}
	
	// Create VM network info
	_, subnet, _ := net.ParseCIDR(m.config.VMSubnet)
	gateway := make(net.IP, len(subnet.IP))
	copy(gateway, subnet.IP)
	gateway[len(gateway)-1] = 1
	
	vmNet := &VMNetwork{
		VMID:       vmID,
		Namespace:  nsName,
		TapDevice:  tapName,
		IPAddress:  ip,
		Netmask:    net.IPv4Mask(255, 255, 255, 0),
		Gateway:    gateway,
		MacAddress: mac,
		DNSServers: m.config.DNSServers,
		CreatedAt:  time.Now(),
	}
	
	m.vmNetworks[vmID] = vmNet
	
	m.logger.Info("created VM network",
		slog.String("vm_id", vmID),
		slog.String("ip", ip.String()),
		slog.String("mac", mac),
		slog.String("tap", tapName),
		slog.String("namespace", nsName),
	)
	
	return vmNet, nil
}

// setupVMNetworking configures the network namespace and TAP device
func (m *Manager) setupVMNetworking(nsName, tapName string, ip net.IP, mac string) error {
	// Try to execute network operations using nsenter to access root namespace
	// This works around systemd's network isolation
	useNsenter := false
	if _, err := exec.LookPath("nsenter"); err == nil {
		// Check if we're in a restricted network view
		if _, err := netlink.LinkByName(m.config.BridgeName); err != nil {
			m.logger.Info("using nsenter to access root network namespace",
				slog.String("bridge", m.config.BridgeName),
			)
			useNsenter = true
		}
	}
	
	// Create veth pair - ensure names are under 15 chars
	// tapName is like "tap-ud-XXXXX", extract the suffix
	suffix := strings.TrimPrefix(tapName, "tap-")
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	vethHost := fmt.Sprintf("vh%s", suffix)
	vethNS := fmt.Sprintf("vn%s", suffix)
	
	// Create veth pair
	var cmd *exec.Cmd
	if useNsenter {
		// Use nsenter to create veth pair in root namespace
		cmd = exec.Command("nsenter", "-t", "1", "-n", "ip", "link", "add", vethHost, "type", "veth", "peer", "name", vethNS)
	} else {
		cmd = exec.Command("ip", "link", "add", vethHost, "type", "veth", "peer", "name", vethNS)
	}
	
	if output, err := cmd.CombinedOutput(); err != nil {
		m.logger.Warn("ip command failed to create veth pair, trying netlink",
			slog.String("output", string(output)),
			slog.String("error", err.Error()),
			slog.Bool("use_nsenter", useNsenter),
		)
		
		// Try with netlink as fallback
		veth := &netlink.Veth{
			LinkAttrs: netlink.LinkAttrs{Name: vethHost},
			PeerName:  vethNS,
		}
		
		if err := netlink.LinkAdd(veth); err != nil {
			return fmt.Errorf("failed to create veth pair: %w", err)
		}
	}
	
	// Get the namespace
	ns, err := netns.GetFromName(nsName)
	if err != nil {
		// Clean up veth pair
		if vethLink, err2 := netlink.LinkByName(vethHost); err2 == nil {
			netlink.LinkDel(vethLink)
		}
		return fmt.Errorf("failed to get namespace: %w", err)
	}
	defer ns.Close()
	
	// Move veth peer to namespace
	// Use nsenter if needed
	if useNsenter {
		// Use ip command via nsenter to move veth to namespace
		cmd := exec.Command("nsenter", "-t", "1", "-n", "ip", "link", "set", vethNS, "netns", nsName)
		if output, err := cmd.CombinedOutput(); err != nil {
			// Clean up veth pair
			exec.Command("nsenter", "-t", "1", "-n", "ip", "link", "del", vethHost).Run()
			return fmt.Errorf("failed to move veth to namespace: %s, %w", string(output), err)
		}
	} else {
		// Sometimes the link takes a moment to appear, retry a few times
		var vethNSLink netlink.Link
		for i := 0; i < 3; i++ {
			vethNSLink, err = netlink.LinkByName(vethNS)
			if err == nil {
				break
			}
			if i < 2 {
				time.Sleep(100 * time.Millisecond)
			}
		}
		if err != nil {
			// Clean up veth pair
			if vethLink, err2 := netlink.LinkByName(vethHost); err2 == nil {
				netlink.LinkDel(vethLink)
			}
			return fmt.Errorf("failed to get veth peer %s: %w", vethNS, err)
		}
		
		if err := netlink.LinkSetNsFd(vethNSLink, int(ns)); err != nil {
			// Clean up veth pair
			if vethLink, err2 := netlink.LinkByName(vethHost); err2 == nil {
				netlink.LinkDel(vethLink)
			}
			return fmt.Errorf("failed to move veth to namespace: %w", err)
		}
	}
	
	// Attach host end to bridge using ip command
	// Using ip command as a fallback when netlink can't see the bridge
	var cmd2 *exec.Cmd
	if useNsenter {
		cmd2 = exec.Command("nsenter", "-t", "1", "-n", "ip", "link", "set", vethHost, "master", m.config.BridgeName)
	} else {
		cmd2 = exec.Command("ip", "link", "set", vethHost, "master", m.config.BridgeName)
	}
	
	if output, err := cmd2.CombinedOutput(); err != nil {
		m.logger.Error("ip command failed to attach veth to bridge",
			slog.String("command", fmt.Sprintf("ip link set %s master %s", vethHost, m.config.BridgeName)),
			slog.String("output", string(output)),
			slog.String("error", err.Error()),
		)
		// Try with netlink as fallback
		vethHostLink, err2 := netlink.LinkByName(vethHost)
		if err2 != nil {
			return fmt.Errorf("failed to get veth host: %w", err2)
		}
		
		bridge, err2 := netlink.LinkByName(m.config.BridgeName)
		if err2 != nil {
			return fmt.Errorf("failed to attach veth to bridge (ip command failed: %v, netlink failed: %v)", err, err2)
		}
		
		if err2 := netlink.LinkSetMaster(vethHostLink, bridge); err2 != nil {
			return fmt.Errorf("failed to attach veth to bridge: %w", err2)
		}
	}
	
	// Bring up the veth host interface
	var cmd3 *exec.Cmd
	if useNsenter {
		cmd3 = exec.Command("nsenter", "-t", "1", "-n", "ip", "link", "set", vethHost, "up")
	} else {
		cmd3 = exec.Command("ip", "link", "set", vethHost, "up")
	}
	
	if err := cmd3.Run(); err != nil {
		// Try netlink as fallback
		if vethHostLink, err2 := netlink.LinkByName(vethHost); err2 == nil {
			if err2 := netlink.LinkSetUp(vethHostLink); err2 != nil {
				return fmt.Errorf("failed to bring up veth host: %w", err2)
			}
		} else {
			return fmt.Errorf("failed to bring up veth host: %w", err)
		}
	}
	
	// Configure inside namespace
	return m.configureNamespace(ns, vethNS, tapName, ip, mac)
}

// configureNamespace sets up networking inside the namespace
func (m *Manager) configureNamespace(ns netns.NsHandle, vethName, tapName string, ip net.IP, mac string) error {
	// Save current namespace
	origNS, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed to get current namespace: %w", err)
	}
	defer origNS.Close()
	
	// Switch to target namespace
	if err := netns.Set(ns); err != nil {
		return fmt.Errorf("failed to set namespace: %w", err)
	}
	defer netns.Set(origNS)
	
	// Create TAP device
	tap := &netlink.Tuntap{
		LinkAttrs: netlink.LinkAttrs{
			Name: tapName,
		},
		Mode: netlink.TUNTAP_MODE_TAP,
	}
	
	if err := netlink.LinkAdd(tap); err != nil {
		return fmt.Errorf("failed to create tap device: %w", err)
	}
	
	// Set MAC address on TAP
	tapLink, err := netlink.LinkByName(tapName)
	if err != nil {
		return fmt.Errorf("failed to get tap link: %w", err)
	}
	
	hwAddr, _ := net.ParseMAC(mac)
	if err := netlink.LinkSetHardwareAddr(tapLink, hwAddr); err != nil {
		return fmt.Errorf("failed to set tap MAC: %w", err)
	}
	
	// Create bridge inside namespace
	br := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: "br0",
		},
	}
	
	if err := netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("failed to create namespace bridge: %w", err)
	}
	
	// Get all links
	brLink, _ := netlink.LinkByName("br0")
	vethLink, _ := netlink.LinkByName(vethName)
	
	// Add interfaces to bridge
	netlink.LinkSetMaster(vethLink, brLink)
	netlink.LinkSetMaster(tapLink, brLink)
	
	// Bring everything up
	netlink.LinkSetUp(vethLink)
	netlink.LinkSetUp(tapLink)
	netlink.LinkSetUp(brLink)
	
	// Add IP to bridge
	addr := &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(24, 32),
		},
	}
	netlink.AddrAdd(brLink, addr)
	
	// Add default route
	_, subnet, _ := net.ParseCIDR(m.config.VMSubnet)
	gateway := make(net.IP, len(subnet.IP))
	copy(gateway, subnet.IP)
	gateway[len(gateway)-1] = 1
	
	route := &netlink.Route{
		Dst: nil, // default route
		Gw:  gateway,
	}
	netlink.RouteAdd(route)
	
	// Setup rate limiting if enabled
	if m.config.EnableRateLimit {
		m.applyRateLimit(tapLink, m.config.RateLimitMbps)
	}
	
	return nil
}

// applyRateLimit applies traffic shaping to the interface
func (m *Manager) applyRateLimit(link netlink.Link, mbps int) error {
	// Use tc (traffic control) to limit bandwidth
	// This is a simplified example - production would use netlink directly
	
	// Delete any existing qdisc
	exec.Command("tc", "qdisc", "del", "dev", link.Attrs().Name, "root").Run()
	
	// Add HTB qdisc
	cmd := exec.Command("tc", "qdisc", "add", "dev", link.Attrs().Name, "root", "handle", "1:", "htb")
	if err := cmd.Run(); err != nil {
		m.logger.Warn("failed to add HTB qdisc",
			slog.String("interface", link.Attrs().Name),
			slog.String("error", err.Error()),
		)
		return nil // Non-fatal
	}
	
	// Add rate limit class
	rate := fmt.Sprintf("%dmbit", mbps)
	cmd = exec.Command("tc", "class", "add", "dev", link.Attrs().Name,
		"parent", "1:", "classid", "1:1", "htb", "rate", rate)
	if err := cmd.Run(); err != nil {
		m.logger.Warn("failed to add rate limit",
			slog.String("interface", link.Attrs().Name),
			slog.String("error", err.Error()),
		)
	}
	
	return nil
}

// DeleteVMNetwork removes networking for a VM
func (m *Manager) DeleteVMNetwork(ctx context.Context, vmID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	vmNet, exists := m.vmNetworks[vmID]
	if !exists {
		return nil // Already deleted
	}
	
	// Release IP
	m.allocator.ReleaseIP(vmNet.IPAddress)
	
	// Delete veth pair (if it exists)
	vethName := fmt.Sprintf("veth%s", strings.TrimPrefix(vmNet.TapDevice, "tap"))
	if link, err := netlink.LinkByName(vethName); err == nil {
		netlink.LinkDel(link)
	}
	
	// Delete network namespace
	m.deleteNamespace(vmNet.Namespace)
	
	delete(m.vmNetworks, vmID)
	
	m.logger.Info("deleted VM network",
		slog.String("vm_id", vmID),
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
	m.logger.Info("shutting down network manager")
	
	// Delete all VM networks
	for vmID := range m.vmNetworks {
		if err := m.DeleteVMNetwork(ctx, vmID); err != nil {
			m.logger.Error("failed to delete VM network during shutdown",
				slog.String("vm_id", vmID),
				slog.String("error", err.Error()),
			)
		}
	}
	
	// Clean up iptables rules
	m.cleanupIPTables()
	
	// Optionally delete bridge (usually we keep it)
	// if m.bridgeCreated {
	//     if link, err := netlink.LinkByName(m.config.BridgeName); err == nil {
	//         netlink.LinkDel(link)
	//     }
	// }
	
	return nil
}

// Helper functions

func (m *Manager) createNamespace(name string) error {
	// Check if namespace already exists
	if _, err := netns.GetFromName(name); err == nil {
		return nil // Already exists
	}
	
	// Create new namespace
	if _, err := netns.NewNamed(name); err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", name, err)
	}
	
	// Create namespace directory for runtime data
	nsDir := filepath.Join("/var/run/netns", name)
	if err := os.MkdirAll(filepath.Dir(nsDir), 0755); err != nil {
		return fmt.Errorf("failed to create namespace directory: %w", err)
	}
	
	return nil
}

func (m *Manager) deleteNamespace(name string) {
	netns.DeleteNamed(name)
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
	// Remove our iptables rules in reverse order
	for i := len(m.iptablesRules) - 1; i >= 0; i-- {
		rule := m.iptablesRules[i]
		// Convert -A to -D to delete the rule
		deleteRule := strings.Replace(rule, "-A", "-D", 1)
		args := strings.Fields(deleteRule)
		
		cmd := exec.Command("iptables", args...)
		if err := cmd.Run(); err != nil {
			m.logger.Warn("failed to remove iptables rule",
				slog.String("rule", rule),
				slog.String("error", err.Error()),
			)
		}
	}
	m.iptablesRules = nil
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
	
	if err := netns.Set(ns); err != nil {
		return nil, fmt.Errorf("failed to set namespace: %w", err)
	}
	defer netns.Set(origNS)
	
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