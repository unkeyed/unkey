package router

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// sentinelInstanceSelectionTotal counts instance selection outcomes.
	//
	// Labels:
	//   outcome: "success", "no_instances", "no_running_instances", "deployment_not_found", "error"
	sentinelInstanceSelectionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "sentinel",
			Name:      "routing_instance_selection_total",
			Help:      "Total number of instance selection attempts by outcome.",
		},
		[]string{"outcome"},
	)

	// sentinelRoutingDuration tracks how long routing operations take.
	//
	// Labels:
	//   operation: "get_deployment", "select_instance"
	sentinelRoutingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "sentinel",
			Name:      "routing_duration_seconds",
			Help:      "Duration of routing operations in seconds.",
			Buckets:   []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
		},
		[]string{"operation"},
	)
)
