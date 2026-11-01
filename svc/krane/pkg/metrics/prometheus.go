package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/unkeyed/unkey/pkg/prometheus/vec"
)

var (
	// ReconcileOperationsTotal counts reconciliation operations (apply/delete).
	// Use this to monitor deployment throughput and error rates.
	//
	// Labels:
	//   - "resource_type": "deployment"
	//   - "operation": "apply" or "delete"
	//   - "result": "success" or "error"
	ReconcileOperationsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "reconcile_operations_total",
			Help:      "Total number of reconciliation operations (apply/delete).",
		},
		vec.Values("resource_type", "deployment"),
		vec.Values("operation", "apply", "delete"),
		vec.Values("result", "success", "error"),
	)

	// ResyncCorrectionsTotal counts corrections made by the periodic resync loop.
	// High values indicate the streaming path is missing events and resync is compensating.
	//
	// Labels:
	//   - "resource_type": "deployment"
	ResyncCorrectionsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "resync_corrections_total",
			Help:      "Total number of corrections made by the resync loop (indicates missed streaming events).",
		},
		vec.Values("resource_type", "deployment"),
	)

	// WatcherFullSyncsTotal counts how often the watcher performs a full sync
	// via SyncDesiredState. Under normal operation this fires once per
	// fullSyncInterval (10 minutes) plus once on startup.
	WatcherFullSyncsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "watcher_full_syncs_total",
			Help:      "Total number of full syncs triggered by the watcher.",
		},
	)

	// StreamConnectionsTotal counts incremental stream connection attempts by outcome.
	//
	// Labels:
	//   - "status": "success" or "error"
	StreamConnectionsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "stream_connections_total",
			Help:      "Total number of incremental stream connection attempts.",
		},
		vec.Values("status", "success", "error"),
	)

	// StreamEventsReceivedTotal counts events received on the incremental stream.
	StreamEventsReceivedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "stream_events_received_total",
			Help:      "Total number of events received on the incremental stream.",
		},
	)

	// RolloutGateWorkloads reports workloads by rollout gate state.
	//
	// Labels:
	//   - "state": "rolling" (holds a slot), "stalled" (exceeded its
	//     progress deadline, paused until a new template or a retry), or
	//     "waiting" (template changed, no slot yet)
	RolloutGateWorkloads = vec.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "rollout_gate_workloads",
			Help:      "Workloads by rollout gate state.",
		},
		vec.Values("state", "rolling", "stalled", "waiting"),
	)

	// LastSuccessfulCheckpointUnixSeconds records when Krane last saved a token
	// after applying all earlier events.
	LastSuccessfulCheckpointUnixSeconds = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "last_successful_checkpoint_unix_seconds",
			Help:      "Unix time when the incremental stream last accepted a checkpoint.",
		},
	)

	// DispatchTotal counts event dispatches to controllers by source, resource type, and outcome.
	//
	// Labels:
	//   - "source": "stream", "retry" (a failed stream event applied again), or "full_sync"
	//   - "resource_type": "deployment", or "unknown" for an unrecognized event
	//   - "status": "success" or "error"
	DispatchTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "dispatch_total",
			Help:      "Total number of event dispatches to controllers.",
		},
		vec.Values("source", "stream", "retry", "full_sync"),
		vec.Values("resource_type", "deployment"),
		vec.Values("status", "success", "error"),
	)

	// StreamRetriesPending counts workloads whose latest stream event failed
	// and waits for another attempt. A value above zero that does not fall
	// means a workload's desired state cannot be applied.
	StreamRetriesPending = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "stream_retries_pending",
			Help:      "Workloads whose latest stream event failed and waits for another attempt.",
		},
	)

	// StreamErrorsTotal counts incremental streams that connected and then
	// ended with an error. Failed connection attempts are counted on
	// [StreamConnectionsTotal] instead.
	StreamErrorsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "stream_errors_total",
			Help:      "Total number of incremental streams that ended with an error after connecting.",
		},
	)

	// FullSyncErrorsTotal counts full syncs that failed before reading every
	// event. Errors applying individual events are counted on [DispatchTotal].
	//
	// Labels:
	//   - "stage": "open" (the stream could not be opened) or "receive" (the
	//     stream ended with an error partway through)
	FullSyncErrorsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "full_sync_errors_total",
			Help:      "Total number of full syncs that failed to open or ended with an error.",
		},
		vec.Values("stage", "open", "receive"),
	)

	// FullSyncEventsReceivedTotal counts events received during a full sync.
	FullSyncEventsReceivedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "full_sync_events_received_total",
			Help:      "Total number of events received during full syncs.",
		},
	)

	// FullSyncDurationSeconds tracks how long the krane-side full sync takes.
	FullSyncDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "full_sync_duration_seconds",
			Help:      "Duration of full sync operations from the krane side.",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
	)

	// PodWatchEventsTotal counts pod watch events and their disposition for
	// the deployment controller.
	//
	// Labels:
	//   - "component": "deployment"
	//   - "event_type": k8s watch event type ("added", "modified", "deleted")
	//   - "outcome": "reported", "deduped", "skipped_no_rs", "skipped_rs_gone",
	//                "error"
	PodWatchEventsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "pod_watch_events_total",
			Help:      "Pod watch events received by krane controllers.",
		},
		vec.Values("component", "deployment"),
		vec.Dynamic("event_type"),
		vec.Values("outcome", "reported", "deduped", "skipped_no_rs", "skipped_rs_gone", "error"),
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
	//   - "component": "deployment"
	//   - "source": "watch" (real-time) — kept as a label for future
	//               resync-sourced samples.
	PodWatchDeliveryLagSeconds = vec.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "pod_watch_delivery_lag_seconds",
			Help:      "Lag between pod ContainersReady=True and krane observing it via the watch.",
			Buckets:   []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60},
		},
		vec.Values("component", "deployment"),
		vec.Values("source", "watch"),
	)

	// PodWatchReconnectsTotal counts pod watch reconnect attempts by reason.
	// Events arriving during the 1-5s reconnect backoff are dropped and fall
	// to the resync loop, so a high rate here correlates with lag spikes
	// on the "resync" label of PodWatchDeliveryLagSeconds.
	//
	// Labels:
	//   - "component": "deployment"
	//   - "reason": "channel_closed", "error"
	PodWatchReconnectsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "pod_watch_reconnects_total",
			Help:      "Pod watch reconnect attempts by reason.",
		},
		vec.Values("component", "deployment"),
		vec.Values("reason", "channel_closed", "error"),
	)

	// ReportStatusDurationSeconds measures krane-side latency of the
	// ReportDeploymentStatus RPC, including circuit breaker overhead.
	//
	// Labels:
	//   - "component": "deployment"
	//   - "result": "success", "error"
	ReportStatusDurationSeconds = vec.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "report_status_duration_seconds",
			Help:      "Duration of ReportDeploymentStatus calls from krane.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		},
		vec.Values("component", "deployment"),
		vec.Values("result", "success", "error"),
	)

	// ReportDedupedTotal counts reportIfChanged invocations that were skipped
	// because the fingerprint matched the last successful report.
	//
	// Labels:
	//   - "component": "deployment"
	ReportDedupedTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "report_deduped_total",
			Help:      "Status reports skipped by fingerprint dedup.",
		},
		vec.Values("component", "deployment"),
	)

	// InstanceEventsEmittedTotal counts container lifecycle events
	// successfully reported to the control plane.
	//
	// Labels:
	//   - "kind": "running", "terminated" or "waiting"
	//   - "reason": kubelet-supplied reason ("OOMKilled", "Error",
	//     "ContainerCannotRun", "CrashLoopBackOff", ...)
	InstanceEventsEmittedTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "instance_events_emitted_total",
			Help:      "Container lifecycle events successfully reported via ReportInstanceEvents.",
		},
		vec.Values("kind", "running", "terminated", "waiting"),
		vec.Dynamic("reason"),
	)

	// InstanceEventsDedupDroppedTotal counts events dropped by the
	// in-memory LRU because the same (pod_uid, container_name,
	// restart_count, event_kind) tuple was already reported. Healthy
	// counter — every pod-watch tick after the first will hit dedup.
	//
	// Labels:
	//   - "kind": "running", "terminated" or "waiting"
	InstanceEventsDedupDroppedTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "instance_events_dedup_dropped_total",
			Help:      "Container lifecycle events skipped because the LRU already saw them.",
		},
		vec.Values("kind", "running", "terminated", "waiting"),
	)

	// Leader is 1 on the replica that holds the krane leader lease and 0 on
	// every other replica.
	Leader = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "leader",
			Help:      "1 if this replica holds the krane leader lease, 0 otherwise.",
		},
	)

	// InstanceEventsReportFailuresTotal counts ReportInstanceEvents RPC
	// transport failures. Should track the existing krane→ctrl RPC
	// failure rate; a divergence means something specific to this RPC.
	InstanceEventsReportFailuresTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "instance_events_report_failures_total",
			Help:      "ReportInstanceEvents RPC transport failures.",
		},
	)
)

// ResetLeaderGauges zeroes the gauges only the leader sets, so a replica that
// lost leadership stops exporting its last values.
func ResetLeaderGauges() {
	for _, state := range []string{"rolling", "stalled", "waiting"} {
		RolloutGateWorkloads.WithLabelValues(state).Set(0)
	}
	LastSuccessfulCheckpointUnixSeconds.Set(0)
	StreamRetriesPending.Set(0)
	Leader.Set(0)
}

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
