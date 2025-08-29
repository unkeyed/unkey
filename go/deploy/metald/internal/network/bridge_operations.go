package network

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"
	"github.com/vishvananda/netlink"
)

// VerifyBridge verifies bridge infrastructure exists and is properly configured
// Bridge is managed by metald-bridge.service
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

	bridge := &netlink.Bridge{ //nolint:exhaustruct // Only setting Name field, other bridge fields use appropriate defaults
		LinkAttrs: netlink.LinkAttrs{ //nolint:exhaustruct // Only setting Name field, other link attributes use appropriate defaults
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
