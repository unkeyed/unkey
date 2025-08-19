/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains usage limiter metrics for tracking performance and behavior of the credit-based usage limiting system.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// UsagelimiterDecisions counts usage limiter decisions by outcome (allowed/denied) and source (redis/db)
	// This counter helps understand the distribution of decisions and fallback patterns.
	//
	// Example usage:
	//   metrics.UsagelimiterDecisions.WithLabelValues("redis", "allowed").Inc()
	//   metrics.UsagelimiterDecisions.WithLabelValues("db", "denied").Inc()
	UsagelimiterDecisions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "usagelimiter",
			Name:        "decisions_total",
			Help:        "Total number of usage limiter decisions.",
			ConstLabels: constLabels,
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
			Namespace:   "unkey",
			Subsystem:   "usagelimiter",
			Name:        "replay_operations_total",
			Help:        "Total number of credit replay operations to database.",
			ConstLabels: constLabels,
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
			Namespace:   "unkey",
			Subsystem:   "usagelimiter",
			Name:        "replay_latency_seconds",
			Help:        "Histogram of replay operation latencies in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
	)

	// UsagelimiterFallbackOperations counts fallback operations to direct DB access
	// This counter helps monitor Redis health and fallback frequency.
	//
	// Example usage:
	//   metrics.UsagelimiterFallbackOperations.Inc()
	UsagelimiterFallbackOperations = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "usagelimiter",
			Name:        "fallback_operations_total",
			Help:        "Total number of fallback operations to direct database access.",
			ConstLabels: constLabels,
		},
	)

	// UsagelimiterCreditsProcessed counts the total number of credits processed
	// This counter helps track the overall usage and throughput of the system.
	//
	// Example usage:
	//   metrics.UsagelimiterCreditsProcessed.Add(float64(creditsUsed))
	UsagelimiterCreditsProcessed = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "usagelimiter",
			Name:        "credits_processed_total",
			Help:        "Total number of credits processed by the usage limiter.",
			ConstLabels: constLabels,
		},
	)
)
