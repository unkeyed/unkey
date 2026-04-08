package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all ctrl Prometheus metrics. Create with [NewMetrics].
type Metrics struct {
	// FullSyncDurationSeconds tracks how long full syncs take in the
	// SyncDesiredState RPC. Growth indicates increasing resource count in the region.
	FullSyncDurationSeconds prometheus.Histogram

	// SyncDesiredStateTotal counts SyncDesiredState RPC calls by outcome.
	//
	// Labels:
	//   - "status": "success" or "error"
	SyncDesiredStateTotal *prometheus.CounterVec

	// SyncDesiredStateEventsSentTotal counts events sent during SyncDesiredState by resource type.
	//
	// Labels:
	//   - "resource_type": "deployment", "sentinel", or "cilium_network_policy"
	SyncDesiredStateEventsSentTotal *prometheus.CounterVec

	// DeploymentChangesProcessedTotal counts incremental deployment change processing outcomes.
	//
	// Labels:
	//   - "resource_type": "deployment_topology", "sentinel", "cilium_network_policy", or "unknown"
	//   - "status": "success", "not_found", or "error"
	DeploymentChangesProcessedTotal *prometheus.CounterVec
}

// NewMetrics creates a [Metrics] instance and registers all collectors with reg.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	f := promauto.With(reg)

	return &Metrics{
		FullSyncDurationSeconds: f.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: "unkey",
				Subsystem: "control",
				Name:      "deployment_changes_full_sync_duration_seconds",
				Help:      "Duration of full sync operations in seconds.",
				Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
			},
		),
		SyncDesiredStateTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "control",
				Name:      "sync_desired_state_total",
				Help:      "Total number of SyncDesiredState RPC calls.",
			},
			[]string{"status"},
		),
		SyncDesiredStateEventsSentTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "control",
				Name:      "sync_desired_state_events_sent_total",
				Help:      "Total number of events sent during SyncDesiredState.",
			},
			[]string{"resource_type"},
		),
		DeploymentChangesProcessedTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "control",
				Name:      "deployment_changes_processed_total",
				Help:      "Total number of deployment changes processed from the outbox.",
			},
			[]string{"resource_type", "status"},
		),
	}
}
