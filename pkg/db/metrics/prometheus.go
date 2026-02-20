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
	// DatabaseOperationsLatency tracks database operation latencies as a histogram,
	// labeled by replica type (rw/ro), operation type, and success status.
	// This collector uses predefined buckets optimized for typical database operation latencies.
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.DatabaseOperationsLatency.WithLabelValues("rw", "exec", "success").Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	DatabaseOperationsLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "database",
			Name:      "operations_latency_seconds",
			Help:      "Histogram of database operation latencies in seconds.",
			Buckets:   latencyBuckets,
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
	DatabaseOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "database",
			Name:      "operations_total",
			Help:      "Total number of database operations processed.",
		},
		[]string{"replica", "operation", "status"},
	)

	// DatabaseOperationsErrorsTotal tracks the total number of database operation errors,
	// labeled by replica type (rw/ro), and operation type.
	// Use this counter to monitor database error rates and identify problematic operations.
	//
	// Example usage:
	//   metrics.DatabaseOperationsErrorsTotal.WithLabelValues("rw", "exec").Inc()
	DatabaseOperationsErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "database",
		Name:      "operations_errors_total",
		Help:      "Total number of database operation errors.",
	}, []string{"replica", "operation"})
)
