package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ReconcileOperationsTotal counts reconciliation operations (apply/delete).
	// Use this to monitor deployment throughput and error rates.
	//
	// Labels:
	//   - "resource_type": "deployment", "sentinel", or "cilium_network_policy"
	//   - "operation": "apply" or "delete"
	//   - "result": "success" or "error"
	ReconcileOperationsTotal = promauto.NewCounterVec(
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
	ResyncCorrectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "resync_corrections_total",
			Help:      "Total number of corrections made by the resync loop (indicates missed streaming events).",
		},
		[]string{"resource_type"},
	)

	// WatcherFullSyncsTotal counts how often the watcher resets its cursor to 0
	// and performs a full sync. Under normal operation this fires once per
	// fullSyncInterval (5 minutes). Elevated values indicate repeated stream failures.
	WatcherFullSyncsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "watcher_full_syncs_total",
			Help:      "Total number of full syncs triggered by the watcher.",
		},
	)

	// WatcherVersionLastSeen is the current cursor position in the deployment_changes
	// stream. If this flatlines, the stream is stuck or the watcher is not receiving events.
	WatcherVersionLastSeen = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "krane",
			Name:      "watcher_version_last_seen",
			Help:      "Current cursor position in the deployment_changes stream.",
		},
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
