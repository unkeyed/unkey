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
	//   - "resource_type": "deployment" or "cilium_network_policy"
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
	//   - "resource_type": "deployment" or "cilium_network_policy"
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

	// RolloutGateWorkloads reports workloads by rollout gate state.
	//
	// Labels:
	//   - "state": "rolling" (holds a slot), "stalled" (exceeded its
	//     progress deadline, paused until a new template or a retry), or
	//     "waiting" (template changed, no slot yet)
	RolloutGateWorkloads = lazy.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "rollout_gate_workloads",
			Help:      "Workloads by rollout gate state.",
		},
		[]string{"state"},
	)

	// LastSuccessfulCheckpointUnixSeconds records when Krane last saved a token
	// after applying all earlier events.
	LastSuccessfulCheckpointUnixSeconds = lazy.NewGauge(
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
	//   - "source": "stream" or "full_sync"
	//   - "resource_type": "deployment" or "cilium_network_policy"
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

	// StreamErrorsTotal counts incremental streams that connected and then
	// ended with an error. Failed connection attempts are counted on
	// [StreamConnectionsTotal] instead.
	StreamErrorsTotal = lazy.NewCounter(
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
	FullSyncErrorsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "full_sync_errors_total",
			Help:      "Total number of full syncs that failed to open or ended with an error.",
		},
		[]string{"stage"},
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
	// the deployment controller.
	//
	// Labels:
	//   - "component": "deployment"
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
	//   - "component": "deployment"
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

	// ReportStatusDurationSeconds measures krane-side latency of the
	// ReportDeploymentStatus RPC, including circuit breaker overhead.
	//
	// Labels:
	//   - "component": "deployment"
	//   - "result": "success", "error"
	ReportStatusDurationSeconds = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "report_status_duration_seconds",
			Help:      "Duration of ReportDeploymentStatus calls from krane.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		},
		[]string{"component", "result"},
	)

	// ReportDedupedTotal counts reportIfChanged invocations that were skipped
	// because the fingerprint matched the last successful report.
	//
	// Labels:
	//   - "component": "deployment"
	ReportDedupedTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "report_deduped_total",
			Help:      "Status reports skipped by fingerprint dedup.",
		},
		[]string{"component"},
	)

	// InstanceEventsEmittedTotal counts container lifecycle events
	// successfully reported to the control plane.
	//
	// Labels:
	//   - "kind": "running", "terminated" or "waiting"
	//   - "reason": kubelet-supplied reason ("OOMKilled", "Error",
	//     "ContainerCannotRun", "CrashLoopBackOff", ...)
	InstanceEventsEmittedTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "instance_events_emitted_total",
			Help:      "Container lifecycle events successfully reported via ReportInstanceEvents.",
		},
		[]string{"kind", "reason"},
	)

	// InstanceEventsDedupDroppedTotal counts events dropped by the
	// in-memory LRU because the same (pod_uid, container_name,
	// restart_count, event_kind) tuple was already reported. Healthy
	// counter — every pod-watch tick after the first will hit dedup.
	//
	// Labels:
	//   - "kind": "running", "terminated" or "waiting"
	InstanceEventsDedupDroppedTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "instance_events_dedup_dropped_total",
			Help:      "Container lifecycle events skipped because the LRU already saw them.",
		},
		[]string{"kind"},
	)

	// Leader is 1 on the replica that holds the krane leader lease and 0 on
	// every other replica.
	Leader = lazy.NewGauge(
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
	InstanceEventsReportFailuresTotal = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "instance_events_report_failures_total",
			Help:      "ReportInstanceEvents RPC transport failures.",
		},
	)
)

// Init registers every krane metric and creates the label combinations krane
// is known to emit, so each series is exported at 0 from startup. Call it after
// lazy.SetRegistry. instance_events_emitted_total and pod_watch_events_total
// are left lazy: the first has kubelet-supplied reasons, the second has too
// many combinations for nothing that alerts on it.
func Init() {
	WatcherFullSyncsTotal.Register()
	StreamEventsReceivedTotal.Register()
	StreamErrorsTotal.Register()
	FullSyncEventsReceivedTotal.Register()
	FullSyncDurationSeconds.Register()
	InstanceEventsReportFailuresTotal.Register()
	LastSuccessfulCheckpointUnixSeconds.Register()
	Leader.Register()

	for _, result := range []string{"success", "error"} {
		StreamConnectionsTotal.WithLabelValues(result)
		ReportStatusDurationSeconds.WithLabelValues("deployment", result)
		for _, source := range []string{"stream", "full_sync"} {
			DispatchTotal.WithLabelValues(source, "deployment", result)
		}
		for _, operation := range []string{"apply", "delete"} {
			ReconcileOperationsTotal.WithLabelValues("deployment", operation, result)
		}
	}
	for _, stage := range []string{"open", "receive"} {
		FullSyncErrorsTotal.WithLabelValues(stage)
	}
	for _, state := range []string{"rolling", "stalled", "waiting"} {
		RolloutGateWorkloads.WithLabelValues(state)
	}
	for _, reason := range []string{"channel_closed", "error"} {
		PodWatchReconnectsTotal.WithLabelValues("deployment", reason)
	}
	for _, kind := range []string{"running", "terminated", "waiting"} {
		InstanceEventsDedupDroppedTotal.WithLabelValues(kind)
	}
	ResyncCorrectionsTotal.WithLabelValues("deployment")
	ReportDedupedTotal.WithLabelValues("deployment")
	PodWatchDeliveryLagSeconds.WithLabelValues("deployment", "watch")
}

// ResetLeaderGauges zeroes the gauges only the leader sets, so a replica that
// lost leadership stops exporting its last values.
func ResetLeaderGauges() {
	for _, state := range []string{"rolling", "stalled", "waiting"} {
		RolloutGateWorkloads.WithLabelValues(state).Set(0)
	}
	LastSuccessfulCheckpointUnixSeconds.Set(0)
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
