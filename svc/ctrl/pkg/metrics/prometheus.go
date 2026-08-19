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

	// NotifyInstancesReadyTotal counts the outcome of maybeNotifyInstancesReady
	// evaluations. Used to distinguish the healthy path (threshold met,
	// Restate notified) from the common early-return reasons. A spike in
	// "threshold_not_met" around deploy completion means krane is reporting
	// instances that the DB doesn't yet reflect as running.
	//
	// Labels:
	//   - "outcome": "notified", "inactive_status", "topology_missing",
	//                "threshold_not_met", "already_notified",
	//                "topology_error", "instances_error", "restate_error"
	NotifyInstancesReadyTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "control",
			Name:      "notify_instances_ready_total",
			Help:      "Outcome of maybeNotifyInstancesReady evaluations.",
		},
		[]string{"outcome"},
	)

	// ReportDeploymentStatusDurationSeconds measures control-plane handler
	// latency for ReportDeploymentStatus, including the TxRetry upsert and
	// the post-commit notify check.
	//
	// Labels:
	//   - "result": "success", "error"
	ReportDeploymentStatusDurationSeconds = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "control",
			Name:      "report_deployment_status_duration_seconds",
			Help:      "Handler latency of ReportDeploymentStatus.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		},
		[]string{"result"},
	)
)
