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
			Namespace:   "unkey",
			Subsystem:   "database",
			Name:        "operations_latency_seconds",
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
	DatabaseOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "database",
			Name:        "operations_total",
			Help:        "Total number of database operations processed.",
			ConstLabels: constLabels,
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
		Namespace:   "unkey",
		Subsystem:   "database",
		Name:        "operations_errors_total",
		Help:        "Total number of database operation errors.",
		ConstLabels: constLabels,
	}, []string{"replica", "operation"})
)
