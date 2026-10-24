package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
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
	// DatabaseOperationsLatency tracks latency in seconds for mysql operations
	// by replica mode, operation type, and status.
	DatabaseOperationsLatency = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "database",
			Name:      "operations_latency_seconds",
			Help:      "Histogram of database operation latencies in seconds.",
			Buckets:   latencyBuckets,
		},
		lazy.Dynamic("replica"),
		lazy.Values("operation", "exec", "prepare", "query", "query_row", "begin", "commit", "rollback", "batch"),
		lazy.Values("status", "success", "error"),
	)

	// DatabaseOperationsTotal tracks the total number of mysql operations
	// by replica mode, operation type, and status.
	DatabaseOperationsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "database",
			Name:      "operations_total",
			Help:      "Total number of database operations processed.",
		},
		lazy.Dynamic("replica"),
		lazy.Values("operation", "exec", "prepare", "query", "query_row", "begin", "commit", "rollback", "batch"),
		lazy.Values("status", "success", "error"),
	)
)
