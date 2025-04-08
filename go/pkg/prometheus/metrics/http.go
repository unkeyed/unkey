/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains HTTP-related metrics for tracking request latencies and counts.
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
	// HTTPRequestLatency tracks HTTP request latencies as a histogram, labeled by method, path, and status.
	// This collector uses predefined buckets optimized for typical web service latencies,
	// ranging from 1ms to 10s.
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.HTTPRequestLatency.WithLabelValues("GET", "/users", "200").Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	HTTPRequestLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: "http",
			Name:      "request_latency_seconds",
			Help:      "Histogram of HTTP request latencies in seconds.",
			Buckets:   latencyBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestTotal tracks the number of HTTP requests handled, labeled by method, path, and status.
	// Use this counter to monitor API traffic patterns and error rates.
	//
	// Example usage:
	//   metrics.HTTPRequestTotal.WithLabelValues("GET", "/users", "200").Inc()
	HTTPRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests processed.",
		},
		[]string{"method", "path", "status"},
	)
)
