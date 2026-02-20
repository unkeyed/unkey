/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains usage limiter metrics for tracking performance and behavior of the credit-based usage limiting system.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Standard histogram buckets for latency metrics in seconds
var latencyBuckets = []float64{
	0.001, // 1ms
	0.002, // 2ms
	0.005, // 5ms
	0.01,  // 10ms
	0.02,  // 20ms
	0.05,  // 50ms
	0.1,   // 100ms
	0.2,   // 200ms
	0.3,   // 300ms
	0.4,   // 400ms
	0.5,   // 500ms
	0.75,  // 750ms
	1.0,   // 1s
	2.0,   // 2s
	3.0,   // 3s
	5.0,   // 5s
	10.0,  // 10s
}

var (
	// UsagelimiterDecisions counts usage limiter decisions by outcome (allowed/denied) and source (redis/db)
	// This counter helps understand the distribution of decisions and fallback patterns.
	//
	// Example usage:
	//   metrics.UsagelimiterDecisions.WithLabelValues("redis", "allowed").Inc()
	//   metrics.UsagelimiterDecisions.WithLabelValues("db", "denied").Inc()
	UsagelimiterDecisions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "usagelimiter",
			Name:      "decisions_total",
			Help:      "Total number of usage limiter decisions.",
		},
		[]string{"source", "outcome"},
	)

	// UsagelimiterReplayOperations counts replay operations to the database by status
	// This counter helps monitor the replay queue health and success rate.
	//
	// Example usage:
	//   metrics.UsagelimiterReplayOperations.WithLabelValues("success").Inc()
	//   metrics.UsagelimiterReplayOperations.WithLabelValues("error").Inc()
	UsagelimiterReplayOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "usagelimiter",
			Name:      "replay_operations_total",
			Help:      "Total number of credit replay operations to database.",
		},
		[]string{"status"},
	)

	// UsagelimiterReplayLatency measures the latency of replay operations to the database
	// This histogram helps track the performance of async database updates.
	//
	// Example usage:
	//   defer func(start time.Time) {
	//       metrics.UsagelimiterReplayLatency.Observe(time.Since(start).Seconds())
	//   }(time.Now())
	UsagelimiterReplayLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "usagelimiter",
			Name:      "replay_latency_seconds",
			Help:      "Histogram of replay operation latencies in seconds.",
			Buckets:   latencyBuckets,
		},
	)
)
