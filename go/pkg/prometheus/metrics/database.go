/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains database-related metrics for tracking database operation performance,
usage patterns, and latency across primary and read-only replicas.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// DatabaseOperationLatency tracks database operation latencies as a histogram,
	// labeled by replica type (rw/ro), operation type, and success status.
	// This collector uses predefined buckets optimized for typical database operation latencies.
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.DatabaseOperationLatency.WithLabelValues("rw", "exec", "success").Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	DatabaseOperationLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem:   "database",
			Name:        "operation_latency_seconds",
			Help:        "Histogram of database operation latencies in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
		[]string{"replica", "operation", "status"},
	)

	// DatabaseOperationTotal tracks the total number of database operations,
	// labeled by replica type (rw/ro), operation type, and success status.
	// Use this counter to monitor database traffic patterns and error rates.
	//
	// Example usage:
	//   metrics.DatabaseOperationTotal.WithLabelValues("rw", "exec", "success").Inc()
	//   metrics.DatabaseOperationTotal.WithLabelValues("ro", "query", "error").Inc()
	DatabaseOperationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem:   "database",
			Name:        "operations_total",
			Help:        "Total number of database operations processed.",
			ConstLabels: constLabels,
		},
		[]string{"replica", "operation", "status"},
	)
)
