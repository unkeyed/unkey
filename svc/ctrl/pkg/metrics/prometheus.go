package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// FullSyncDurationSeconds tracks how long full syncs take in the
	// WatchDeploymentChanges RPC. Growth indicates increasing resource count in the region.
	FullSyncDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "control",
			Name:      "deployment_changes_full_sync_duration_seconds",
			Help:      "Duration of full sync operations in seconds.",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
	)
)
