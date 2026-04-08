package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all krane Prometheus metrics. Create with [NewMetrics].
type Metrics struct {
	// ReconcileOperationsTotal counts reconciliation operations (apply/delete).
	// Use this to monitor deployment throughput and error rates.
	//
	// Labels:
	//   - "resource_type": "deployment", "sentinel", or "cilium_network_policy"
	//   - "operation": "apply" or "delete"
	//   - "result": "success" or "error"
	ReconcileOperationsTotal *prometheus.CounterVec

	// ResyncCorrectionsTotal counts corrections made by the periodic resync loop.
	// High values indicate the streaming path is missing events and resync is compensating.
	//
	// Labels:
	//   - "resource_type": "deployment", "sentinel", or "cilium_network_policy"
	ResyncCorrectionsTotal *prometheus.CounterVec

	// WatcherFullSyncsTotal counts how often the watcher performs a full sync
	// via SyncDesiredState. Under normal operation this fires once per
	// fullSyncInterval (10 minutes) plus once on startup.
	WatcherFullSyncsTotal prometheus.Counter

	// WatcherVersionLastSeen is the current cursor position in the deployment_changes
	// stream. If this flatlines, the stream is stuck or the watcher is not receiving events.
	WatcherVersionLastSeen prometheus.Gauge

	// StreamConnectionsTotal counts incremental stream connection attempts by outcome.
	//
	// Labels:
	//   - "status": "success" or "error"
	StreamConnectionsTotal *prometheus.CounterVec

	// StreamEventsReceivedTotal counts events received on the incremental stream.
	StreamEventsReceivedTotal prometheus.Counter

	// DispatchTotal counts event dispatches to controllers by source, resource type, and outcome.
	//
	// Labels:
	//   - "source": "stream" or "full_sync"
	//   - "resource_type": "deployment", "sentinel", or "cilium_network_policy"
	//   - "status": "success" or "error"
	DispatchTotal *prometheus.CounterVec

	// FullSyncEventsReceivedTotal counts events received during a full sync.
	FullSyncEventsReceivedTotal prometheus.Counter

	// FullSyncDurationSeconds tracks how long the krane-side full sync takes.
	FullSyncDurationSeconds prometheus.Histogram
}

// NewMetrics creates a [Metrics] instance and registers all collectors with reg.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	f := promauto.With(reg)

	return &Metrics{
		ReconcileOperationsTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "krane",
				Name:      "reconcile_operations_total",
				Help:      "Total number of reconciliation operations (apply/delete).",
			},
			[]string{"resource_type", "operation", "result"},
		),
		ResyncCorrectionsTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "krane",
				Name:      "resync_corrections_total",
				Help:      "Total number of corrections made by the resync loop (indicates missed streaming events).",
			},
			[]string{"resource_type"},
		),
		WatcherFullSyncsTotal: f.NewCounter(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "krane",
				Name:      "watcher_full_syncs_total",
				Help:      "Total number of full syncs triggered by the watcher.",
			},
		),
		WatcherVersionLastSeen: f.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "unkey",
				Subsystem: "krane",
				Name:      "watcher_version_last_seen",
				Help:      "Current cursor position in the deployment_changes stream.",
			},
		),
		StreamConnectionsTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "krane",
				Name:      "stream_connections_total",
				Help:      "Total number of incremental stream connection attempts.",
			},
			[]string{"status"},
		),
		StreamEventsReceivedTotal: f.NewCounter(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "krane",
				Name:      "stream_events_received_total",
				Help:      "Total number of events received on the incremental stream.",
			},
		),
		DispatchTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "krane",
				Name:      "dispatch_total",
				Help:      "Total number of event dispatches to controllers.",
			},
			[]string{"source", "resource_type", "status"},
		),
		FullSyncEventsReceivedTotal: f.NewCounter(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "krane",
				Name:      "full_sync_events_received_total",
				Help:      "Total number of events received during full syncs.",
			},
		),
		FullSyncDurationSeconds: f.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: "unkey",
				Subsystem: "krane",
				Name:      "full_sync_duration_seconds",
				Help:      "Duration of full sync operations from the krane side.",
				Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
			},
		),
	}
}

// RecordReconcile records a reconciliation operation result. Intended for use
// with defer and named returns:
//
//	func (c *Controller) ApplyDeployment(...) (retErr error) {
//	    defer func() { m.RecordReconcile("deployment", "apply", retErr) }()
func (m *Metrics) RecordReconcile(resourceType, operation string, err error) {
	result := "success"
	if err != nil {
		result = "error"
	}
	m.ReconcileOperationsTotal.WithLabelValues(resourceType, operation, result).Inc()
}
