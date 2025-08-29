package network

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"runtime"
	"strings"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// configureNamespace configures networking inside the VM namespace
func (m *Manager) configureNamespace(ns netns.NsHandle, vethName string, tapName string, ip net.IP, mac string, workspaceSubnet string) error {
	// Save current namespace
	origNs, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed to get current namespace: %w", err)
	}
	defer origNs.Close()

	// Switch to target namespace
	if err := netns.Set(ns); err != nil {
		return fmt.Errorf("failed to set namespace: %w", err)
	}

	// Ensure we return to original namespace
	defer func() {
		if setErr := netns.Set(origNs); setErr != nil {
			m.logger.Error("failed to restore original namespace", "error", setErr)
		}
	}()

	m.logger.Info("configuring namespace networking",
		slog.String("veth", vethName),
		slog.String("tap", tapName),
		slog.String("ip", ip.String()),
		slog.String("mac", mac),
	)

	// Get veth device in namespace
	vethLink, err := netlink.LinkByName(vethName)
	if err != nil {
		return fmt.Errorf("failed to find veth device %s in namespace: %w", vethName, err)
	}

	// Calculate /29 subnet for this VM
	vmSubnet := calculateVMSubnet(ip)
	vmAddr, err := netlink.ParseAddr(fmt.Sprintf("%s/29", ip.String()))
	if err != nil {
		return fmt.Errorf("failed to parse VM IP address: %w", err)
	}

	// Add IP to veth interface
	if err := netlink.AddrAdd(vethLink, vmAddr); err != nil {
		return fmt.Errorf("failed to add IP to veth device: %w", err)
	}

	// Set MAC address
	if mac != "" {
		hwAddr, err := net.ParseMAC(mac)
		if err != nil {
			return fmt.Errorf("failed to parse MAC address %s: %w", mac, err)
		}
		if err := netlink.LinkSetHardwareAddr(vethLink, hwAddr); err != nil {
			return fmt.Errorf("failed to set MAC address: %w", err)
		}
	}

	// Bring up veth interface
	if err := netlink.LinkSetUp(vethLink); err != nil {
		return fmt.Errorf("failed to bring up veth device: %w", err)
	}

	// Set up loopback interface
	lo, err := netlink.LinkByName("lo")
	if err != nil {
		return fmt.Errorf("failed to find loopback device: %w", err)
	}
	if err := netlink.LinkSetUp(lo); err != nil {
		return fmt.Errorf("failed to bring up loopback: %w", err)
	}

	// AIDEV-NOTE: CRITICAL FIX - Add default route through the /29 subnet gateway
	// The gateway IP is the first IP in the VM's /29 subnet
	gatewayIP := calculateVethHostIP(ip)
	if gatewayIP != "" {
		gateway := net.ParseIP(gatewayIP)
		if gateway != nil {
			defaultRoute := &netlink.Route{
				Dst: nil, // Default route (0.0.0.0/0)
				Gw:  gateway,
			}

			if err := netlink.RouteAdd(defaultRoute); err != nil {
				// Route might already exist, log warning but don't fail
				m.logger.Warn("failed to add default route (may already exist)",
					slog.String("gateway", gatewayIP),
					slog.String("error", err.Error()),
				)
			} else {
				m.logger.Info("added default route via /29 gateway",
					slog.String("gateway", gatewayIP),
					slog.String("vm_ip", ip.String()),
					slog.String("vm_subnet", vmSubnet),
				)
			}
		}
	}

	// Add route to workspace subnet via the veth interface
	if workspaceSubnet != "" {
		_, workspaceNet, err := net.ParseCIDR(workspaceSubnet)
		if err == nil {
			workspaceRoute := &netlink.Route{
				Dst:       workspaceNet,
				LinkIndex: vethLink.Attrs().Index,
			}

			if err := netlink.RouteAdd(workspaceRoute); err != nil {
				m.logger.Warn("failed to add workspace subnet route (may already exist)",
					slog.String("subnet", workspaceSubnet),
					slog.String("error", err.Error()),
				)
			} else {
				m.logger.Info("added route to workspace subnet",
					slog.String("subnet", workspaceSubnet),
					slog.String("interface", vethName),
				)
			}
		}
	}

	// Bridge TAP device to veth in namespace using tc-mirred
	// This allows Firecracker to communicate through the veth
	m.logger.Info("bridging TAP to veth using tc-mirred",
		slog.String("tap", tapName),
		slog.String("veth", vethName),
	)

	// Get TAP device in namespace
	tapLink, err := netlink.LinkByName(tapName)
	if err != nil {
		return fmt.Errorf("failed to find TAP device %s in namespace: %w", tapName, err)
	}

	// Setup ingress qdisc on both interfaces (required for tc-mirred)
	for _, link := range []netlink.Link{tapLink, vethLink} {
		qdisc := &netlink.Ingress{
			QdiscAttrs: netlink.QdiscAttrs{
				LinkIndex: link.Attrs().Index,
				Handle:    netlink.MakeHandle(0xffff, 0),
				Parent:    netlink.HANDLE_INGRESS,
			},
		}
		if err := netlink.QdiscAdd(qdisc); err != nil && !strings.Contains(err.Error(), "file exists") {
			return fmt.Errorf("failed to add ingress qdisc: %w", err)
		}
	}

	// Mirror traffic from TAP to veth
	// This command redirects all ingress traffic from TAP to veth egress
	tcCmd := []string{
		"tc", "filter", "add",
		"dev", tapName,
		"parent", "ffff:",
		"protocol", "all",
		"u32", "match", "u8", "0", "0",
		"action", "mirred", "egress", "redirect", "dev", vethName,
	}
	if output, err := exec.Command(tcCmd[0], tcCmd[1:]...).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to setup TAP->veth mirroring: %s: %w", output, err)
	}

	// Mirror traffic from veth to TAP
	tcCmd = []string{
		"tc", "filter", "add",
		"dev", vethName,
		"parent", "ffff:",
		"protocol", "all",
		"u32", "match", "u8", "0", "0",
		"action", "mirred", "egress", "redirect", "dev", tapName,
	}
	if output, err := exec.Command(tcCmd[0], tcCmd[1:]...).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to setup veth->TAP mirroring: %s: %w", output, err)
	}

	m.logger.Info("successfully bridged TAP and veth devices")

	return nil
}

// createNamespace creates a new network namespace if it doesn't exist
func (m *Manager) createNamespace(name string) error {
	// AIDEV-NOTE: CRITICAL FIX - Lock OS thread when creating namespace
	// This prevents race conditions with namespace operations
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Check if namespace already exists
	if _, err := netns.GetFromName(name); err == nil {
		m.logger.Info("namespace already exists", slog.String("namespace", name))
		return nil // Namespace already exists
	}

	// Create new namespace
	m.logger.Info("creating new namespace", slog.String("namespace", name))
	newNs, err := netns.NewNamed(name)
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", name, err)
	}
	newNs.Close()

	m.logger.Info("namespace created successfully", slog.String("namespace", name))
	return nil
}

// deleteNamespace removes a network namespace
func (m *Manager) deleteNamespace(name string) {
	// AIDEV-NOTE: CRITICAL FIX - Lock OS thread when deleting namespace
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	m.logger.Info("deleting namespace", slog.String("namespace", name))

	if err := netns.DeleteNamed(name); err != nil {
		if !strings.Contains(err.Error(), "not found") &&
			!strings.Contains(err.Error(), "no such file") {
			m.logger.Warn("failed to delete namespace",
				slog.String("namespace", name),
				slog.String("error", err.Error()),
			)
		}
	} else {
		m.logger.Info("namespace deleted successfully", slog.String("namespace", name))
	}
}

// namespaceExists checks if a namespace exists
func (m *Manager) namespaceExists(namespace string) bool {
	// AIDEV-NOTE: CRITICAL FIX - Lock OS thread for namespace operations
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if ns, err := netns.GetFromName(namespace); err == nil {
		ns.Close()
		return true
	}
	return false
}

// verifyNetworkCleanup verifies that all network resources for a VM have been cleaned up
func (m *Manager) verifyNetworkCleanup(ctx context.Context, vmID string, deviceNames *NetworkDeviceNames) error {
	var issues []string

	// Check if TAP device still exists
	if _, err := netlink.LinkByName(deviceNames.TAP); err == nil {
		issues = append(issues, fmt.Sprintf("TAP device %s still exists", deviceNames.TAP))
	}

	// Check if host veth still exists
	if _, err := netlink.LinkByName(deviceNames.VethHost); err == nil {
		issues = append(issues, fmt.Sprintf("veth device %s still exists", deviceNames.VethHost))
	}

	// Check if namespace still exists
	expectedNS := fmt.Sprintf("vn_%s", deviceNames.Namespace[3:]) // Extract network ID
	if m.namespaceExists(expectedNS) {
		issues = append(issues, fmt.Sprintf("namespace %s still exists", expectedNS))
	}

	if len(issues) > 0 {
		return fmt.Errorf("cleanup verification failed for VM %s: %s", vmID, strings.Join(issues, ", "))
	}

	m.logger.InfoContext(ctx, "network cleanup verified successfully",
		slog.String("vm_id", vmID),
		slog.String("tap", deviceNames.TAP),
		slog.String("veth", deviceNames.VethHost),
		slog.String("namespace", expectedNS),
	)

	return nil
}
