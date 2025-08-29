package network

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

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
