package network

import (
	"fmt"
	"log/slog"
	"net"
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// setupVMNetworking creates and configures TAP and veth devices for a VM
func (m *Manager) setupVMNetworking(nsName string, deviceNames *NetworkDeviceNames, ip net.IP, mac string, workspaceSubnet string) error {
	// Lock the OS thread for namespace operations
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	m.logger.Info("setting up VM networking devices",
		slog.String("namespace", nsName),
		slog.String("tap", deviceNames.TAP),
		slog.String("veth_host", deviceNames.VethHost),
		slog.String("veth_ns", deviceNames.VethNS),
		slog.String("ip", ip.String()),
		slog.String("mac", mac),
		slog.String("workspace_subnet", workspaceSubnet),
	)

	// Get target namespace handle
	targetNs, err := netns.GetFromName(nsName)
	if err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", nsName, err)
	}
	defer targetNs.Close()

	// Create TAP device in host namespace for Firecracker
	m.logger.Info("creating TAP device", slog.String("device", deviceNames.TAP))

	tap := &netlink.Tuntap{
		LinkAttrs: netlink.LinkAttrs{
			Name: deviceNames.TAP,
		},
		Mode: netlink.TUNTAP_MODE_TAP,
	}

	if err := netlink.LinkAdd(tap); err != nil {
		return fmt.Errorf("failed to create TAP device %s: %w", deviceNames.TAP, err)
	}

	// Set TAP device up
	if err := netlink.LinkSetUp(tap); err != nil {
		return fmt.Errorf("failed to bring TAP device up: %w", err)
	}

	m.logger.Info("TAP device created and up", slog.String("tap", deviceNames.TAP))

	// Create veth pair
	m.logger.Info("creating veth pair",
		slog.String("host_side", deviceNames.VethHost),
		slog.String("ns_side", deviceNames.VethNS),
	)

	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: deviceNames.VethHost,
		},
		PeerName: deviceNames.VethNS,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return fmt.Errorf("failed to create veth pair: %w", err)
	}

	m.logger.Info("veth pair created successfully")

	// Get veth peer device
	vethPeer, err := netlink.LinkByName(deviceNames.VethNS)
	if err != nil {
		return fmt.Errorf("failed to find veth peer device %s: %w", deviceNames.VethNS, err)
	}

	// Move veth peer to target namespace
	if err := netlink.LinkSetNsFd(vethPeer, int(targetNs)); err != nil {
		return fmt.Errorf("failed to move veth peer to namespace: %w", err)
	}

	m.logger.Info("moved veth peer to namespace",
		slog.String("device", deviceNames.VethNS),
		slog.String("namespace", nsName),
	)

	// Configure the namespace side of the veth
	if err := m.configureNamespace(targetNs, deviceNames.VethNS, deviceNames.TAP, ip, mac, workspaceSubnet); err != nil {
		return fmt.Errorf("failed to configure namespace: %w", err)
	}

	// Bring up host side of veth
	hostVeth, err := netlink.LinkByName(deviceNames.VethHost)
	if err != nil {
		return fmt.Errorf("failed to get host veth device: %w", err)
	}

	if err := netlink.LinkSetUp(hostVeth); err != nil {
		return fmt.Errorf("failed to bring up host veth: %w", err)
	}

	// AIDEV-NOTE: CRITICAL FIX - Set up /29 subnet gateway IP on the host veth interface
	// This creates proper L3 routing for the VM's /29 subnet
	gatewayIP := calculateVethHostIP(ip)
	if gatewayIP != "" {
		gatewayAddr, err := netlink.ParseAddr(gatewayIP + "/29")
		if err != nil {
			return fmt.Errorf("failed to parse gateway IP %s: %w", gatewayIP, err)
		}

		// Add gateway IP to host veth interface
		if err := netlink.AddrAdd(hostVeth, gatewayAddr); err != nil {
			return fmt.Errorf("failed to add gateway IP to host veth: %w", err)
		}

		m.logger.Info("configured gateway IP on host veth",
			slog.String("veth", deviceNames.VethHost),
			slog.String("gateway_ip", gatewayIP),
		)
	}

	// Apply rate limiting if enabled
	if m.config.EnableRateLimit && m.config.RateLimitMbps > 0 {
		m.applyRateLimit(hostVeth, m.config.RateLimitMbps)
	}

	m.logger.Info("VM networking setup completed successfully",
		slog.String("tap", deviceNames.TAP),
		slog.String("veth_host", deviceNames.VethHost),
		slog.String("veth_ns", deviceNames.VethNS),
		slog.String("namespace", nsName),
		slog.String("ip", ip.String()),
		slog.String("mac", mac),
	)

	return nil
}

// calculateVMSubnet calculates the /29 subnet for a VM based on its IP
func calculateVMSubnet(ip net.IP) string {
	// Calculate the base of the /29 subnet (multiples of 8)
	lastOctet := ip[3]
	base := (lastOctet / 8) * 8
	return fmt.Sprintf("%d.%d.%d.%d/29", ip[0], ip[1], ip[2], base)
}

// calculateVethHostIP calculates the host-side veth IP for a VM subnet
// Returns the first IP in the /29 subnet as the gateway
func calculateVethHostIP(vmIP net.IP) string {
	// Calculate the base of the /29 subnet (multiples of 8)
	lastOctet := vmIP[3]
	base := (lastOctet / 8) * 8
	// First IP in the /29 is the gateway (base + 1)
	return fmt.Sprintf("%d.%d.%d.%d", vmIP[0], vmIP[1], vmIP[2], base+1)
}

// applyRateLimit applies traffic rate limiting to a network interface
func (m *Manager) applyRateLimit(link netlink.Link, mbps int) {
	// Calculate rate in bytes per second
	rate := uint32(mbps * 125000) // Convert Mbps to bytes/sec (1 Mbps = 125000 bytes/sec)
	burst := rate / 10            // Allow burst of 1/10th of the rate

	// Create TC qdisc for rate limiting
	qdisc := &netlink.Tbf{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: link.Attrs().Index,
			Handle:    netlink.MakeHandle(1, 0),
			Parent:    netlink.HANDLE_ROOT,
		},
		Rate:   uint64(rate),
		Limit:  32 * 1024, // 32KB buffer
		Buffer: burst,
	}

	// Add the qdisc (ignore errors if already exists)
	if err := netlink.QdiscAdd(qdisc); err != nil {
		m.logger.Warn("failed to add rate limit qdisc (may already exist)",
			slog.String("device", link.Attrs().Name),
			slog.Int("mbps", mbps),
			slog.String("error", err.Error()),
		)
	} else {
		m.logger.Info("applied rate limit to interface",
			slog.String("device", link.Attrs().Name),
			slog.Int("mbps", mbps),
			slog.Uint64("rate_bytes_per_sec", uint64(rate)),
		)
	}
}
