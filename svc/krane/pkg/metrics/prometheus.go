package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var (
	// ReconcileOperationsTotal counts reconciliation operations (apply/delete).
	// Use this to monitor deployment throughput and error rates.
	//
	// Labels:
	//   - "resource_type": "deployment", "sentinel", or "cilium_network_policy"
	//   - "operation": "apply" or "delete"
	//   - "result": "success" or "error"
	ReconcileOperationsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "reconcile_operations_total",
			Help:      "Total number of reconciliation operations (apply/delete).",
		},
		[]string{"resource_type", "operation", "result"},
	)

	// ResyncCorrectionsTotal counts corrections made by the periodic resync loop.
	// High values indicate the streaming path is missing events and resync is compensating.
	//
	// Labels:
	//   - "resource_type": "deployment", "sentinel", or "cilium_network_policy"
	ResyncCorrectionsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "resync_corrections_total",
			Help:      "Total number of corrections made by the resync loop (indicates missed streaming events).",
		},
		[]string{"resource_type"},
	)

	// WatcherFullSyncsTotal counts how often the watcher performs a full sync
	// via SyncDesiredState. Under normal operation this fires once per
	// fullSyncInterval (10 minutes) plus once on startup.
	WatcherFullSyncsTotal = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "watcher_full_syncs_total",
			Help:      "Total number of full syncs triggered by the watcher.",
		},
	)

	// WatcherVersionLastSeen is the current cursor position in the deployment_changes
	// stream. If this flatlines, the stream is stuck or the watcher is not receiving events.
	WatcherVersionLastSeen = lazy.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "watcher_version_last_seen",
			Help:      "Current cursor position in the deployment_changes stream.",
		},
	)

	// StreamConnectionsTotal counts incremental stream connection attempts by outcome.
	//
	// Labels:
	//   - "status": "success" or "error"
	StreamConnectionsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "stream_connections_total",
			Help:      "Total number of incremental stream connection attempts.",
		},
		[]string{"status"},
	)

	// StreamEventsReceivedTotal counts events received on the incremental stream.
	StreamEventsReceivedTotal = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "stream_events_received_total",
			Help:      "Total number of events received on the incremental stream.",
		},
	)

	// DispatchTotal counts event dispatches to controllers by source, resource type, and outcome.
	//
	// Labels:
	//   - "source": "stream" or "full_sync"
	//   - "resource_type": "deployment", "sentinel", or "cilium_network_policy"
	//   - "status": "success" or "error"
	DispatchTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "dispatch_total",
			Help:      "Total number of event dispatches to controllers.",
		},
		[]string{"source", "resource_type", "status"},
	)

	// FullSyncEventsReceivedTotal counts events received during a full sync.
	FullSyncEventsReceivedTotal = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "full_sync_events_received_total",
			Help:      "Total number of events received during full syncs.",
		},
	)

	// FullSyncDurationSeconds tracks how long the krane-side full sync takes.
	FullSyncDurationSeconds = lazy.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "full_sync_duration_seconds",
			Help:      "Duration of full sync operations from the krane side.",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
	)

	// PodWatchEventsTotal counts pod watch events and their disposition for
	// both the deployment and sentinel controllers.
	//
	// Labels:
	//   - "component": "deployment" or "sentinel"
	//   - "event_type": k8s watch event type ("added", "modified", "deleted")
	//   - "outcome": "reported", "deduped", "skipped_no_rs", "skipped_rs_gone",
	//                "skipped_no_deployment", "error"
	PodWatchEventsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "pod_watch_events_total",
			Help:      "Pod watch events received by krane controllers.",
		},
		[]string{"component", "event_type", "outcome"},
	)

	// PodWatchDeliveryLagSeconds measures time between the kubelet publishing
	// ContainersReady=True on a pod and krane first observing it via the
	// real-time watch.
	//
	// Only emitted from the watch path (see observePodReadyLagOnTransition):
	// observing lag from the resync or list-sync paths would re-sample the
	// same static LastTransitionTime on every pass and report pod age as
	// "lag". The drift signal ("resync had to correct something the watch
	// missed") lives on [ResyncCorrectionsTotal].
	//
	// Labels:
	//   - "component": "deployment" or "sentinel"
	//   - "source": "watch" (real-time) — kept as a label for future
	//               resync-sourced samples.
	PodWatchDeliveryLagSeconds = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "pod_watch_delivery_lag_seconds",
			Help:      "Lag between pod ContainersReady=True and krane observing it via the watch.",
			Buckets:   []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"component", "source"},
	)

	// PodWatchReconnectsTotal counts pod watch reconnect attempts by reason.
	// Events arriving during the 1-5s reconnect backoff are dropped and fall
	// to the resync loop, so a high rate here correlates with lag spikes
	// on the "resync" label of PodWatchDeliveryLagSeconds.
	//
	// Labels:
	//   - "component": "deployment" or "sentinel"
	//   - "reason": "channel_closed", "error"
	PodWatchReconnectsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "pod_watch_reconnects_total",
			Help:      "Pod watch reconnect attempts by reason.",
		},
		[]string{"component", "reason"},
	)

	// ReportStatusDurationSeconds measures krane-side latency of the
	// ReportDeploymentStatus and ReportSentinelStatus RPCs, including
	// circuit breaker overhead.
	//
	// Labels:
	//   - "component": "deployment" or "sentinel"
	//   - "result": "success", "error"
	ReportStatusDurationSeconds = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "report_status_duration_seconds",
			Help:      "Duration of ReportDeploymentStatus / ReportSentinelStatus calls from krane.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		},
		[]string{"component", "result"},
	)

	// ReportDedupedTotal counts reportIfChanged invocations that were skipped
	// because the fingerprint matched the last successful report.
	//
	// Labels:
	//   - "component": "deployment" or "sentinel"
	ReportDedupedTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "report_deduped_total",
			Help:      "Status reports skipped by fingerprint dedup.",
		},
		[]string{"component"},
	)
)

// RecordReconcile records a reconciliation operation result. Intended for use
// with defer and named returns:
//
//	func (c *Controller) ApplyDeployment(...) (retErr error) {
//	    defer func() { metrics.RecordReconcile("deployment", "apply", retErr) }()
func RecordReconcile(resourceType, operation string, err error) {
	result := "success"
	if err != nil {
		result = "error"
	}
	ReconcileOperationsTotal.WithLabelValues(resourceType, operation, result).Inc()
}
