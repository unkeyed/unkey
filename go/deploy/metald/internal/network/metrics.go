package network

import (
	"context"
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
	meter := otel.Meter("metald/network",
		metric.WithInstrumentationVersion("1.0.0"),
		metric.WithSchemaURL("https://github.com/unkeyed/unkey/go/deploy/metald"),
	)

	// Initialize bridge capacity metrics
	bridgeVMCount, err := meter.Int64UpDownCounter(
		"metald_network_bridge_vm_count",
		metric.WithDescription("Number of VMs currently attached to each bridge"),
		metric.WithUnit("vm"),
	)
	if err != nil {
		return nil, err
	}

	bridgeCapacityRatio, err := meter.Float64Gauge(
		"metald_network_bridge_capacity_ratio",
		metric.WithDescription("Current VM count as ratio of maximum capacity for each bridge"),
		metric.WithUnit("ratio"),
	)
	if err != nil {
		return nil, err
	}

	bridgeUtilization, err := meter.Int64Histogram(
		"metald_network_bridge_utilization",
		metric.WithDescription("Distribution of bridge utilization percentages"),
		metric.WithUnit("percent"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize VM network operation metrics
	vmNetworkCreateTotal, err := meter.Int64Counter(
		"metald_network_vm_create_total",
		metric.WithDescription("Total number of VM network creation attempts"),
		metric.WithUnit("operations"),
	)
	if err != nil {
		return nil, err
	}

	vmNetworkDeleteTotal, err := meter.Int64Counter(
		"metald_network_vm_delete_total",
		metric.WithDescription("Total number of VM network deletion attempts"),
		metric.WithUnit("operations"),
	)
	if err != nil {
		return nil, err
	}

	vmNetworkErrors, err := meter.Int64Counter(
		"metald_network_vm_errors_total",
		metric.WithDescription("Total number of VM network operation errors"),
		metric.WithUnit("errors"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize resource leak metrics
	orphanedTAPDevices, err := meter.Int64UpDownCounter(
		"metald_network_orphaned_tap_devices",
		metric.WithDescription("Number of orphaned TAP devices detected during cleanup"),
		metric.WithUnit("devices"),
	)
	if err != nil {
		return nil, err
	}

	orphanedVethDevices, err := meter.Int64UpDownCounter(
		"metald_network_orphaned_veth_devices",
		metric.WithDescription("Number of orphaned veth devices detected during cleanup"),
		metric.WithUnit("devices"),
	)
	if err != nil {
		return nil, err
	}

	orphanedNamespaces, err := meter.Int64UpDownCounter(
		"metald_network_orphaned_namespaces",
		metric.WithDescription("Number of orphaned network namespaces detected during cleanup"),
		metric.WithUnit("namespaces"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize host protection metrics
	routeHijackDetected, err := meter.Int64Counter(
		"metald_network_route_hijack_detected_total",
		metric.WithDescription("Total number of route hijack attempts detected"),
		metric.WithUnit("detections"),
	)
	if err != nil {
		return nil, err
	}

	routeRecoveryAttempts, err := meter.Int64Counter(
		"metald_network_route_recovery_attempts_total",
		metric.WithDescription("Total number of route recovery attempts"),
		metric.WithUnit("attempts"),
	)
	if err != nil {
		return nil, err
	}

	hostProtectionStatus, err := meter.Int64UpDownCounter(
		"metald_network_host_protection_status",
		metric.WithDescription("Host protection system status (1=active, 0=inactive)"),
		metric.WithUnit("status"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize performance metrics
	networkSetupDuration, err := meter.Float64Histogram(
		"metald_network_setup_duration_seconds",
		metric.WithDescription("Duration of VM network setup operations"),
		metric.WithUnit("seconds"),
	)
	if err != nil {
		return nil, err
	}

	networkCleanupDuration, err := meter.Float64Histogram(
		"metald_network_cleanup_duration_seconds",
		metric.WithDescription("Duration of VM network cleanup operations"),
		metric.WithUnit("seconds"),
	)
	if err != nil {
		return nil, err
	}

	return &NetworkMetrics{
		logger:                 logger,
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

// RecordVMNetworkCreate records a VM network creation event
func (m *NetworkMetrics) RecordVMNetworkCreate(ctx context.Context, bridgeName string, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("bridge", bridgeName),
		attribute.Bool("success", success),
	}

	m.vmNetworkCreateTotal.Add(ctx, 1, metric.WithAttributes(attrs...))

	if success {
		m.updateBridgeStats(bridgeName, 1) // Increment VM count
	} else {
		m.vmNetworkErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordVMNetworkDelete records a VM network deletion event
func (m *NetworkMetrics) RecordVMNetworkDelete(ctx context.Context, bridgeName string, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("bridge", bridgeName),
		attribute.Bool("success", success),
	}

	m.vmNetworkDeleteTotal.Add(ctx, 1, metric.WithAttributes(attrs...))

	if success {
		m.updateBridgeStats(bridgeName, -1) // Decrement VM count
	} else {
		m.vmNetworkErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordNetworkSetupDuration records the time taken for network setup
func (m *NetworkMetrics) RecordNetworkSetupDuration(ctx context.Context, duration time.Duration, bridgeName string, success bool) {
	m.networkSetupDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(
			attribute.String("bridge", bridgeName),
			attribute.Bool("success", success),
		))
}

// RecordNetworkCleanupDuration records the time taken for network cleanup
func (m *NetworkMetrics) RecordNetworkCleanupDuration(ctx context.Context, duration time.Duration, bridgeName string, success bool) {
	m.networkCleanupDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(
			attribute.String("bridge", bridgeName),
			attribute.Bool("success", success),
		))
}

// RecordOrphanedResources records orphaned network resources found during cleanup
func (m *NetworkMetrics) RecordOrphanedResources(ctx context.Context, taps, veths, namespaces int64) {
	m.orphanedTAPDevices.Add(ctx, taps)
	m.orphanedVethDevices.Add(ctx, veths)
	m.orphanedNamespaces.Add(ctx, namespaces)
}

// RecordRouteHijackDetected records detection of route hijacking attempt
func (m *NetworkMetrics) RecordRouteHijackDetected(ctx context.Context, hijackedInterface, expectedInterface string) {
	m.routeHijackDetected.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("hijacked_interface", hijackedInterface),
			attribute.String("expected_interface", expectedInterface),
		))
}

// RecordRouteRecoveryAttempt records a route recovery attempt
func (m *NetworkMetrics) RecordRouteRecoveryAttempt(ctx context.Context, success bool) {
	m.routeRecoveryAttempts.Add(ctx, 1,
		metric.WithAttributes(attribute.Bool("success", success)))
}

// SetHostProtectionStatus updates the host protection system status
func (m *NetworkMetrics) SetHostProtectionStatus(ctx context.Context, active bool) {
	var status int64
	if active {
		status = 1
	}
	m.hostProtectionStatus.Add(ctx, status)
}
