package router

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// routingDecisionsTotal tracks where frontline routes each request.
	// Use this to understand traffic distribution across regions and detect
	// unexpected shifts (e.g., all traffic going remote means local instances are down).
	//
	// Labels:
	//   decision: "local_sentinel" or "remote_region"
	//   target_region: the region being routed to (local region name or remote region.platform)
	routingDecisionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline_routing",
			Name:      "decisions_total",
			Help:      "Total routing decisions by type and target region.",
		},
		[]string{"decision", "target_region"},
	)

	// routingErrorsTotal tracks routing failures by reason.
	// Spike in "no_running_instances" = deployments are down.
	// Spike in "no_sentinels_for_instances" = sentinels unhealthy but instances exist.
	// Spike in "config_not_found" = bad DNS / unconfigured domains hitting frontline.
	routingErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline_routing",
			Name:      "errors_total",
			Help:      "Total routing errors by reason.",
		},
		[]string{"reason"},
	)

	// routingDuration tracks how long the full Route() call takes (cache lookups + DB fallback).
	// High p99 here means cache misses or DB latency is hurting request latency.
	routingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline_routing",
			Name:      "duration_seconds",
			Help:      "Time spent making routing decisions (hostname lookup + sentinel selection + instance check).",
			Buckets:   []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
	)


)
