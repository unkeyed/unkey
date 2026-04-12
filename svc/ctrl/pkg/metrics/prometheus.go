package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var (
	// FullSyncDurationSeconds tracks how long full syncs take in the
	// SyncDesiredState RPC. Growth indicates increasing resource count in the region.
	FullSyncDurationSeconds = lazy.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "control",
			Name:      "deployment_changes_full_sync_duration_seconds",
			Help:      "Duration of full sync operations in seconds.",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
	)

	// SyncDesiredStateTotal counts SyncDesiredState RPC calls by outcome.
	//
	// Labels:
	//   - "status": "success" or "error"
	SyncDesiredStateTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "control",
			Name:      "sync_desired_state_total",
			Help:      "Total number of SyncDesiredState RPC calls.",
		},
		[]string{"status"},
	)

	// SyncDesiredStateEventsSentTotal counts events sent during SyncDesiredState by resource type.
	//
	// Labels:
	//   - "resource_type": "deployment", "sentinel", or "cilium_network_policy"
	SyncDesiredStateEventsSentTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "control",
			Name:      "sync_desired_state_events_sent_total",
			Help:      "Total number of events sent during SyncDesiredState.",
		},
		[]string{"resource_type"},
	)

	// DeploymentChangesProcessedTotal counts incremental deployment change processing outcomes.
	//
	// Labels:
	//   - "resource_type": "deployment_topology", "sentinel", "cilium_network_policy", or "unknown"
	//   - "status": "success", "not_found", or "error"
	DeploymentChangesProcessedTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "control",
			Name:      "deployment_changes_processed_total",
			Help:      "Total number of deployment changes processed from the outbox.",
		},
		[]string{"resource_type", "status"},
	)
)
