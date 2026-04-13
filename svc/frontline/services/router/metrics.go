package router

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var (
	// routingDecisionsTotal tracks where frontline routes each request.
	//
	// Labels:
	//   decision: "local_sentinel" or "remote_region"
	//   target_region: the region being routed to (local region name or remote region.platform)
	routingDecisionsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "routing_decisions_total",
			Help:      "Total routing decisions by type and target region.",
		},
		[]string{"decision", "target_region"},
	)

	// routingErrorsTotal tracks routing failures by reason.
	//
	// Labels:
	//   reason: "no_running_instances", "no_sentinels_for_instances",
	//           "config_not_found", "config_load_failed", "sentinel_load_failed",
	//           "instance_load_failed", "no_reachable_region"
	routingErrorsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "routing_errors_total",
			Help:      "Total routing errors by reason.",
		},
		[]string{"reason"},
	)

	// routingDuration tracks how long the full Route() call takes (cache lookups + DB fallback).
	routingDuration = lazy.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "routing_duration_seconds",
			Help:      "Time spent making routing decisions.",
			Buckets:   []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
	)
)
