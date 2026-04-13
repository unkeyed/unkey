package router

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var (
	// sentinelDeploymentLookupTotal counts deployment lookup outcomes.
	//
	// Labels:
	//   outcome: "not_found", "error"
	sentinelDeploymentLookupTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "sentinel",
			Name:      "routing_deployment_lookup_total",
			Help:      "Total number of deployment lookup attempts by outcome.",
		},
		[]string{"outcome"},
	)

	// sentinelInstanceSelectionTotal counts instance selection outcomes.
	//
	// Labels:
	//   outcome: "success", "no_instances", "no_running_instances", "error"
	sentinelInstanceSelectionTotal = lazy.NewCounterVec(
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
	sentinelRoutingDuration = lazy.NewHistogramVec(
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
