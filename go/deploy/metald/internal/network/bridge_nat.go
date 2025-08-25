package network

import (
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
)

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
