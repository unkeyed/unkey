package network

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"
	"github.com/vishvananda/netlink"
)

// HostProtection monitors and protects the host's primary network interface
// from being hijacked by metald bridges
type HostProtection struct {
	logger         *slog.Logger
	config         *config.NetworkConfig
	primaryIface   string
	originalRoutes []netlink.Route
	originalDNS    []string
	monitorActive  bool
	mutex          sync.RWMutex
	stopChan       chan struct{}
}

// NewHostProtection creates a new host protection system
func NewHostProtection(logger *slog.Logger, netConfig *config.NetworkConfig) *HostProtection {
	return &HostProtection{
		logger:   logger.With("component", "host-protection"),
		config:   netConfig,
		stopChan: make(chan struct{}),
	}
}

// Start initializes and starts the host protection system
func (p *HostProtection) Start(ctx context.Context) error {
	if !p.config.EnableHostProtection {
		p.logger.InfoContext(ctx, "host protection disabled")
		return nil
	}

	p.logger.InfoContext(ctx, "starting host network protection")

	// 1. Detect primary interface
	if err := p.detectPrimaryInterface(); err != nil {
		return fmt.Errorf("failed to detect primary interface: %w", err)
	}

	// 2. Snapshot current network state
	if err := p.snapshotNetworkState(); err != nil {
		return fmt.Errorf("failed to snapshot network state: %w", err)
	}

	// 3. Install protective iptables rules
	if err := p.installProtectiveRules(); err != nil {
		return fmt.Errorf("failed to install protective rules: %w", err)
	}

	// 4. Start monitoring
	go p.monitorNetworkChanges(ctx)

	p.logger.InfoContext(ctx, "host protection started successfully",
		slog.String("primary_interface", p.primaryIface),
		slog.Int("protected_routes", len(p.originalRoutes)),
	)

	return nil
}

// Stop shuts down the host protection system
func (p *HostProtection) Stop(ctx context.Context) error {
	if !p.config.EnableHostProtection {
		return nil
	}

	p.logger.InfoContext(ctx, "stopping host protection")

	p.mutex.Lock()
	p.monitorActive = false
	p.mutex.Unlock()

	close(p.stopChan)

	// Clean up protective iptables rules
	if err := p.removeProtectiveRules(); err != nil {
		p.logger.WarnContext(ctx, "failed to remove protective rules", "error", err)
	}

	p.logger.InfoContext(ctx, "host protection stopped")
	return nil
}

// detectPrimaryInterface finds the primary network interface
func (p *HostProtection) detectPrimaryInterface() error {
	if p.config.PrimaryInterface != "" {
		p.primaryIface = p.config.PrimaryInterface
		p.logger.Info("using configured primary interface",
			slog.String("interface", p.primaryIface))
		return nil
	}

	// Find default route interface
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to list routes: %w", err)
	}

	for _, route := range routes {
		if route.Dst == nil { // Default route
			link, err := netlink.LinkByIndex(route.LinkIndex)
			if err == nil {
				// Skip virtual interfaces
				ifaceName := link.Attrs().Name
				if !p.isVirtualInterface(ifaceName) {
					p.primaryIface = ifaceName
					p.logger.Info("detected primary interface",
						slog.String("interface", p.primaryIface),
						slog.String("type", link.Type()),
					)
					return nil
				}
			}
		}
	}

	return fmt.Errorf("could not detect primary interface")
}

// isVirtualInterface checks if an interface is virtual (should be ignored)
func (p *HostProtection) isVirtualInterface(name string) bool {
	virtualPrefixes := []string{
		"lo", "docker", "br-", "virbr", "veth", "tap_", "vh_", "vn_",
		"metald-", "tun", "bridge", "dummy", "bond", "team",
	}

	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

// snapshotNetworkState captures the current network configuration
func (p *HostProtection) snapshotNetworkState() error {
	// Capture routes
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to capture routes: %w", err)
	}

	// Filter routes for primary interface
	for _, route := range routes {
		if link, err := netlink.LinkByIndex(route.LinkIndex); err == nil {
			if link.Attrs().Name == p.primaryIface {
				p.originalRoutes = append(p.originalRoutes, route)
			}
		}
	}

	p.logger.Info("captured network state snapshot",
		slog.Int("routes", len(p.originalRoutes)),
		slog.String("primary_interface", p.primaryIface),
	)

	return nil
}

// installProtectiveRules installs iptables rules to prevent bridge hijacking
func (p *HostProtection) installProtectiveRules() error {
	rules := [][]string{
		// Mark traffic from metald bridges
		{"-t", "mangle", "-I", "POSTROUTING", "1", "-o", "br-vms", "-j", "MARK", "--set-mark", "0x100"},
		{"-t", "mangle", "-I", "POSTROUTING", "1", "-o", "metald-br+", "-j", "MARK", "--set-mark", "0x100"},

		// Ensure host traffic uses primary interface (higher priority)
		{"-t", "mangle", "-I", "OUTPUT", "1", "-o", p.primaryIface, "-j", "MARK", "--set-mark", "0x200"},

		// Protect against bridge route hijacking
		{"-t", "mangle", "-I", "PREROUTING", "1", "-i", "br-vms", "-j", "MARK", "--set-mark", "0x100"},
		{"-t", "mangle", "-I", "PREROUTING", "1", "-i", "metald-br+", "-j", "MARK", "--set-mark", "0x100"},
	}

	for _, rule := range rules {
		cmd := exec.Command("iptables", rule...)
		if err := cmd.Run(); err != nil {
			p.logger.Warn("failed to install protective rule",
				slog.Any("rule", rule),
				slog.String("error", err.Error()),
			)
			// Don't fail completely - some rules might work
		}
	}

	p.logger.Info("installed protective iptables rules")
	return nil
}

// removeProtectiveRules removes the protective iptables rules
func (p *HostProtection) removeProtectiveRules() error {
	rules := [][]string{
		// Remove in reverse order
		{"-t", "mangle", "-D", "PREROUTING", "-i", "metald-br+", "-j", "MARK", "--set-mark", "0x100"},
		{"-t", "mangle", "-D", "PREROUTING", "-i", "br-vms", "-j", "MARK", "--set-mark", "0x100"},
		{"-t", "mangle", "-D", "OUTPUT", "-o", p.primaryIface, "-j", "MARK", "--set-mark", "0x200"},
		{"-t", "mangle", "-D", "POSTROUTING", "-o", "metald-br+", "-j", "MARK", "--set-mark", "0x100"},
		{"-t", "mangle", "-D", "POSTROUTING", "-o", "br-vms", "-j", "MARK", "--set-mark", "0x100"},
	}

	for _, rule := range rules {
		cmd := exec.Command("iptables", rule...)
		_ = cmd.Run() // Ignore errors during cleanup
	}

	return nil
}

// monitorNetworkChanges monitors for network changes that could affect host connectivity
func (p *HostProtection) monitorNetworkChanges(ctx context.Context) {
	p.mutex.Lock()
	p.monitorActive = true
	p.mutex.Unlock()

	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.checkNetworkHealth(ctx)
		}
	}
}

// checkNetworkHealth verifies that host networking is still healthy
func (p *HostProtection) checkNetworkHealth(ctx context.Context) {
	p.mutex.RLock()
	if !p.monitorActive {
		p.mutex.RUnlock()
		return
	}
	p.mutex.RUnlock()

	// 1. Check if primary interface still exists and is up
	if err := p.checkPrimaryInterface(); err != nil {
		p.logger.WarnContext(ctx, "primary interface check failed", "error", err)
		return
	}

	// 2. Check for route hijacking
	if hijacked := p.detectRouteHijacking(); hijacked {
		p.logger.ErrorContext(ctx, "CRITICAL: route hijacking detected, attempting recovery")
		if err := p.recoverHostRoutes(); err != nil {
			p.logger.ErrorContext(ctx, "failed to recover host routes", "error", err)
		}
	}

	// 3. Check connectivity
	if err := p.checkConnectivity(); err != nil {
		p.logger.WarnContext(ctx, "connectivity check failed", "error", err)
	}
}

// checkPrimaryInterface verifies the primary interface is still healthy
func (p *HostProtection) checkPrimaryInterface() error {
	link, err := netlink.LinkByName(p.primaryIface)
	if err != nil {
		return fmt.Errorf("primary interface %s not found: %w", p.primaryIface, err)
	}

	if link.Attrs().OperState != netlink.OperUp {
		return fmt.Errorf("primary interface %s is not up: %s", p.primaryIface, link.Attrs().OperState)
	}

	return nil
}

// detectRouteHijacking checks if metald bridges have hijacked routing
func (p *HostProtection) detectRouteHijacking() bool {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		p.logger.Warn("failed to list routes for hijacking detection", "error", err)
		return false
	}

	// Look for default routes pointing to metald bridges
	for _, route := range routes {
		if route.Dst == nil { // Default route
			if link, err := netlink.LinkByIndex(route.LinkIndex); err == nil {
				name := link.Attrs().Name
				if strings.HasPrefix(name, "br-vms") || strings.HasPrefix(name, "metald-br") {
					p.logger.Error("route hijacking detected",
						slog.String("hijacked_interface", name),
						slog.String("expected_interface", p.primaryIface),
					)
					return true
				}
			}
		}
	}

	return false
}

// recoverHostRoutes attempts to restore proper host routing
func (p *HostProtection) recoverHostRoutes() error {
	p.logger.Info("attempting to recover host routes")

	// Get current routes
	currentRoutes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to list current routes: %w", err)
	}

	// Remove any default routes pointing to metald bridges
	for _, route := range currentRoutes {
		if route.Dst == nil { // Default route
			if link, err := netlink.LinkByIndex(route.LinkIndex); err == nil {
				name := link.Attrs().Name
				if strings.HasPrefix(name, "br-vms") || strings.HasPrefix(name, "metald-br") {
					if delErr := netlink.RouteDel(&route); delErr != nil {
						p.logger.Warn("failed to delete hijacked route",
							slog.String("interface", name),
							slog.String("error", delErr.Error()),
						)
					} else {
						p.logger.Info("removed hijacked route", slog.String("interface", name))
					}
				}
			}
		}
	}

	return nil
}

// checkConnectivity tests basic internet connectivity
func (p *HostProtection) checkConnectivity() error {
	// Try to resolve a DNS name
	_, err := net.LookupHost("google.com")
	if err != nil {
		return fmt.Errorf("DNS resolution failed: %w", err)
	}

	return nil
}

// GetStatus returns the current status of host protection
func (p *HostProtection) GetStatus() *HostProtectionStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return &HostProtectionStatus{
		Enabled:          p.config.EnableHostProtection,
		Active:           p.monitorActive,
		PrimaryInterface: p.primaryIface,
		ProtectedRoutes:  len(p.originalRoutes),
	}
}

// HostProtectionStatus represents the current status of host protection
type HostProtectionStatus struct {
	Enabled          bool   `json:"enabled"`
	Active           bool   `json:"active"`
	PrimaryInterface string `json:"primary_interface"`
	ProtectedRoutes  int    `json:"protected_routes"`
}
