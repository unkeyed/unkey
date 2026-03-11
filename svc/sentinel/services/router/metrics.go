package router

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// sentinelInstanceSelectionTotal counts instance selection outcomes.
	//
	// Labels:
	//   - "outcome": "success", "no_instances", "no_running_instances", "deployment_not_found", "error"
	sentinelInstanceSelectionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinel_routing_instance_selection_total",
			Help: "Total number of instance selection attempts by outcome.",
		},
		[]string{"outcome"},
	)

	// sentinelRoutingDuration tracks how long routing operations take.
	//
	// Labels:
	//   - "operation": "get_deployment", "select_instance"
	sentinelRoutingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "sentinel_routing_duration_seconds",
			Help:    "Duration of routing operations in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
)
