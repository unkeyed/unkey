package network

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// NetworkMetrics handles all network-related metrics for metald
type NetworkMetrics struct {
	logger *slog.Logger
	meter  metric.Meter

	// Bridge capacity metrics
	bridgeVMCount       metric.Int64UpDownCounter
	bridgeCapacityRatio metric.Float64Gauge
	bridgeUtilization   metric.Int64Histogram

	// VM network metrics
	vmNetworkCreateTotal metric.Int64Counter
	vmNetworkDeleteTotal metric.Int64Counter
	vmNetworkErrors      metric.Int64Counter

	// Resource leak metrics
	orphanedTAPDevices  metric.Int64UpDownCounter
	orphanedVethDevices metric.Int64UpDownCounter
	orphanedNamespaces  metric.Int64UpDownCounter

	// Host protection metrics
	routeHijackDetected   metric.Int64Counter
	routeRecoveryAttempts metric.Int64Counter
	hostProtectionStatus  metric.Int64UpDownCounter

	// Performance metrics
	networkSetupDuration   metric.Float64Histogram
	networkCleanupDuration metric.Float64Histogram

	mutex       sync.RWMutex
	bridgeStats map[string]*BridgeStats
}

// BridgeStats tracks statistics for a specific bridge
type BridgeStats struct {
	BridgeName   string
	VMCount      int64
	MaxVMs       int64
	CreatedAt    time.Time
	LastActivity time.Time
	IsHealthy    bool
	ErrorCount   int64
}

// NewNetworkMetrics creates a new network metrics collector
func NewNetworkMetrics(logger *slog.Logger) (*NetworkMetrics, error) {
	meter := otel.Meter("metald.network")

	// Initialize all metrics
	bridgeVMCount, err := meter.Int64UpDownCounter(
		"metald_bridge_vm_count",
		metric.WithDescription("Current number of VMs per bridge"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	bridgeCapacityRatio, err := meter.Float64Gauge(
		"metald_bridge_capacity_ratio",
		metric.WithDescription("Ratio of current VMs to maximum VMs per bridge (0.0-1.0)"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	bridgeUtilization, err := meter.Int64Histogram(
		"metald_bridge_utilization_percent",
		metric.WithDescription("Bridge utilization percentage"),
		metric.WithUnit("%"),
		metric.WithExplicitBucketBoundaries(10, 25, 50, 75, 90, 95, 99),
	)
	if err != nil {
		return nil, err
	}

	vmNetworkCreateTotal, err := meter.Int64Counter(
		"metald_vm_network_create_total",
		metric.WithDescription("Total number of VM network creations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	vmNetworkDeleteTotal, err := meter.Int64Counter(
		"metald_vm_network_delete_total",
		metric.WithDescription("Total number of VM network deletions"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	vmNetworkErrors, err := meter.Int64Counter(
		"metald_vm_network_errors_total",
		metric.WithDescription("Total number of VM network errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	orphanedTAPDevices, err := meter.Int64UpDownCounter(
		"metald_orphaned_tap_devices",
		metric.WithDescription("Number of orphaned TAP devices"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	orphanedVethDevices, err := meter.Int64UpDownCounter(
		"metald_orphaned_veth_devices",
		metric.WithDescription("Number of orphaned veth devices"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	orphanedNamespaces, err := meter.Int64UpDownCounter(
		"metald_orphaned_namespaces",
		metric.WithDescription("Number of orphaned network namespaces"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	routeHijackDetected, err := meter.Int64Counter(
		"metald_route_hijack_detected_total",
		metric.WithDescription("Total number of route hijacking attempts detected"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	routeRecoveryAttempts, err := meter.Int64Counter(
		"metald_route_recovery_attempts_total",
		metric.WithDescription("Total number of route recovery attempts"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	hostProtectionStatus, err := meter.Int64UpDownCounter(
		"metald_host_protection_status",
		metric.WithDescription("Host protection status (1=active, 0=inactive)"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	networkSetupDuration, err := meter.Float64Histogram(
		"metald_network_setup_duration_seconds",
		metric.WithDescription("Time taken to set up VM networking"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0),
	)
	if err != nil {
		return nil, err
	}

	networkCleanupDuration, err := meter.Float64Histogram(
		"metald_network_cleanup_duration_seconds",
		metric.WithDescription("Time taken to clean up VM networking"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0),
	)
	if err != nil {
		return nil, err
	}

	return &NetworkMetrics{
		logger:                 logger.With("component", "network-metrics"),
		meter:                  meter,
		bridgeVMCount:          bridgeVMCount,
		bridgeCapacityRatio:    bridgeCapacityRatio,
		bridgeUtilization:      bridgeUtilization,
		vmNetworkCreateTotal:   vmNetworkCreateTotal,
		vmNetworkDeleteTotal:   vmNetworkDeleteTotal,
		vmNetworkErrors:        vmNetworkErrors,
		orphanedTAPDevices:     orphanedTAPDevices,
		orphanedVethDevices:    orphanedVethDevices,
		orphanedNamespaces:     orphanedNamespaces,
		routeHijackDetected:    routeHijackDetected,
		routeRecoveryAttempts:  routeRecoveryAttempts,
		hostProtectionStatus:   hostProtectionStatus,
		networkSetupDuration:   networkSetupDuration,
		networkCleanupDuration: networkCleanupDuration,
		bridgeStats:            make(map[string]*BridgeStats),
	}, nil
}

// RecordVMNetworkCreate records a VM network creation
func (m *NetworkMetrics) RecordVMNetworkCreate(ctx context.Context, bridgeName string, success bool) {
	m.vmNetworkCreateTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("bridge", bridgeName),
		attribute.Bool("success", success),
	))

	if success {
		m.updateBridgeStats(bridgeName, 1)
	} else {
		m.vmNetworkErrors.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "create"),
			attribute.String("bridge", bridgeName),
		))
	}
}

// RecordVMNetworkDelete records a VM network deletion
func (m *NetworkMetrics) RecordVMNetworkDelete(ctx context.Context, bridgeName string, success bool) {
	m.vmNetworkDeleteTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("bridge", bridgeName),
		attribute.Bool("success", success),
	))

	if success {
		m.updateBridgeStats(bridgeName, -1)
	} else {
		m.vmNetworkErrors.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "delete"),
			attribute.String("bridge", bridgeName),
		))
	}
}

// RecordNetworkSetupDuration records the time taken for network setup
func (m *NetworkMetrics) RecordNetworkSetupDuration(ctx context.Context, duration time.Duration, bridgeName string, success bool) {
	m.networkSetupDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("bridge", bridgeName),
		attribute.Bool("success", success),
	))
}

// RecordNetworkCleanupDuration records the time taken for network cleanup
func (m *NetworkMetrics) RecordNetworkCleanupDuration(ctx context.Context, duration time.Duration, bridgeName string, success bool) {
	m.networkCleanupDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("bridge", bridgeName),
		attribute.Bool("success", success),
	))
}

// RecordOrphanedResources records counts of orphaned network resources
func (m *NetworkMetrics) RecordOrphanedResources(ctx context.Context, taps, veths, namespaces int64) {
	m.orphanedTAPDevices.Add(ctx, taps)
	m.orphanedVethDevices.Add(ctx, veths)
	m.orphanedNamespaces.Add(ctx, namespaces)
}

// RecordRouteHijackDetected records a route hijacking detection
func (m *NetworkMetrics) RecordRouteHijackDetected(ctx context.Context, hijackedInterface, expectedInterface string) {
	m.routeHijackDetected.Add(ctx, 1, metric.WithAttributes(
		attribute.String("hijacked_interface", hijackedInterface),
		attribute.String("expected_interface", expectedInterface),
	))
}

// RecordRouteRecoveryAttempt records a route recovery attempt
func (m *NetworkMetrics) RecordRouteRecoveryAttempt(ctx context.Context, success bool) {
	m.routeRecoveryAttempts.Add(ctx, 1, metric.WithAttributes(
		attribute.Bool("success", success),
	))
}

// RecordNamespaceDeletionFailure records a namespace deletion failure
func (m *NetworkMetrics) RecordNamespaceDeletionFailure(ctx context.Context, namespace, errorMsg string) {
	m.vmNetworkErrors.Add(ctx, 1, metric.WithAttributes(
		attribute.String("error_type", "namespace_deletion_failed"),
		attribute.String("namespace", namespace),
		attribute.String("error", errorMsg),
	))

	// Also increment orphaned namespaces counter since it failed to delete
	m.orphanedNamespaces.Add(ctx, 1)
}

// SetHostProtectionStatus sets the host protection status
func (m *NetworkMetrics) SetHostProtectionStatus(ctx context.Context, active bool) {
	status := int64(0)
	if active {
		status = 1
	}
	m.hostProtectionStatus.Add(ctx, status)
}

// updateBridgeStats updates bridge statistics and capacity metrics
func (m *NetworkMetrics) updateBridgeStats(bridgeName string, vmCountDelta int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	stats, exists := m.bridgeStats[bridgeName]
	if !exists {
		stats = &BridgeStats{
			BridgeName:   bridgeName,
			VMCount:      0,
			MaxVMs:       1000, // Default max VMs per bridge
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
			IsHealthy:    true,
		}
		m.bridgeStats[bridgeName] = stats
	}

	stats.VMCount += vmCountDelta
	stats.LastActivity = time.Now()

	// Ensure VM count doesn't go negative
	if stats.VMCount < 0 {
		stats.VMCount = 0
	}

	// Update metrics
	ctx := context.Background()
	m.bridgeVMCount.Add(ctx, vmCountDelta, metric.WithAttributes(
		attribute.String("bridge", bridgeName),
	))

	// Calculate and record capacity ratio
	ratio := float64(stats.VMCount) / float64(stats.MaxVMs)
	m.bridgeCapacityRatio.Record(ctx, ratio, metric.WithAttributes(
		attribute.String("bridge", bridgeName),
	))

	// Calculate and record utilization percentage
	utilizationPercent := int64(ratio * 100)
	m.bridgeUtilization.Record(ctx, utilizationPercent, metric.WithAttributes(
		attribute.String("bridge", bridgeName),
	))

	// Log warnings for high utilization
	if ratio >= 0.9 {
		m.logger.Warn("bridge approaching capacity",
			slog.String("bridge", bridgeName),
			slog.Int64("current_vms", stats.VMCount),
			slog.Int64("max_vms", stats.MaxVMs),
			slog.Float64("utilization", ratio),
		)
	}
}

// SetBridgeMaxVMs sets the maximum VMs for a bridge
func (m *NetworkMetrics) SetBridgeMaxVMs(bridgeName string, maxVMs int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	stats, exists := m.bridgeStats[bridgeName]
	if !exists {
		stats = &BridgeStats{
			BridgeName:   bridgeName,
			VMCount:      0,
			MaxVMs:       maxVMs,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
			IsHealthy:    true,
		}
		m.bridgeStats[bridgeName] = stats
	} else {
		stats.MaxVMs = maxVMs
	}
}

// GetBridgeStats returns current bridge statistics
func (m *NetworkMetrics) GetBridgeStats() map[string]*BridgeStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to avoid concurrent access issues
	statsCopy := make(map[string]*BridgeStats)
	for name, stats := range m.bridgeStats {
		statsCopy[name] = &BridgeStats{
			BridgeName:   stats.BridgeName,
			VMCount:      stats.VMCount,
			MaxVMs:       stats.MaxVMs,
			CreatedAt:    stats.CreatedAt,
			LastActivity: stats.LastActivity,
			IsHealthy:    stats.IsHealthy,
			ErrorCount:   stats.ErrorCount,
		}
	}

	return statsCopy
}

// GetBridgeCapacityAlerts returns bridges that are approaching capacity
func (m *NetworkMetrics) GetBridgeCapacityAlerts() []BridgeCapacityAlert {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var alerts []BridgeCapacityAlert

	for _, stats := range m.bridgeStats {
		ratio := float64(stats.VMCount) / float64(stats.MaxVMs)

		var severity AlertSeverity
		var threshold float64

		switch {
		case ratio >= 0.95:
			severity = AlertCritical
			threshold = 0.95
		case ratio >= 0.90:
			severity = AlertWarning
			threshold = 0.90
		case ratio >= 0.80:
			severity = AlertInfo
			threshold = 0.80
		default:
			continue // No alert needed
		}

		alerts = append(alerts, BridgeCapacityAlert{
			BridgeName:       stats.BridgeName,
			CurrentVMs:       stats.VMCount,
			MaxVMs:           stats.MaxVMs,
			UtilizationRatio: ratio,
			Severity:         severity,
			Threshold:        threshold,
			Message:          m.formatCapacityAlertMessage(stats, ratio, severity),
		})
	}

	return alerts
}

// formatCapacityAlertMessage creates a human-readable alert message
func (m *NetworkMetrics) formatCapacityAlertMessage(stats *BridgeStats, ratio float64, severity AlertSeverity) string {
	utilizationPercent := int(ratio * 100)

	switch severity {
	case AlertCritical:
		return fmt.Sprintf("CRITICAL: Bridge %s is at %d%% capacity (%d/%d VMs). Immediate action required!",
			stats.BridgeName, utilizationPercent, stats.VMCount, stats.MaxVMs)
	case AlertWarning:
		return fmt.Sprintf("WARNING: Bridge %s is at %d%% capacity (%d/%d VMs). Consider load balancing or scaling.",
			stats.BridgeName, utilizationPercent, stats.VMCount, stats.MaxVMs)
	case AlertInfo:
		return fmt.Sprintf("INFO: Bridge %s utilization is %d%% (%d/%d VMs). Monitor for continued growth.",
			stats.BridgeName, utilizationPercent, stats.VMCount, stats.MaxVMs)
	default:
		return fmt.Sprintf("Bridge %s utilization: %d%% (%d/%d VMs)",
			stats.BridgeName, utilizationPercent, stats.VMCount, stats.MaxVMs)
	}
}

// BridgeCapacityAlert represents a bridge capacity alert
type BridgeCapacityAlert struct {
	BridgeName       string        `json:"bridge_name"`
	CurrentVMs       int64         `json:"current_vms"`
	MaxVMs           int64         `json:"max_vms"`
	UtilizationRatio float64       `json:"utilization_ratio"`
	Severity         AlertSeverity `json:"severity"`
	Threshold        float64       `json:"threshold"`
	Message          string        `json:"message"`
}

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertInfo     AlertSeverity = "info"
	AlertWarning  AlertSeverity = "warning"
	AlertCritical AlertSeverity = "critical"
)
