package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// NoopMetrics returns a Metrics instance registered to a discarded registry.
// Use this when metrics are not needed (tests, optional components).
func NoopMetrics() *Metrics {
	return NewMetrics(prometheus.NewRegistry())
}

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

// Metrics holds all Prometheus metrics for the mysql package.
type Metrics struct {
	// DatabaseOperationsLatency tracks latency in seconds for mysql operations
	// by replica mode, operation type, and status.
	DatabaseOperationsLatency *prometheus.HistogramVec

	// DatabaseOperationsTotal tracks the total number of mysql operations
	// by replica mode, operation type, and status.
	DatabaseOperationsTotal *prometheus.CounterVec

	// DatabaseOperationsErrorsTotal tracks mysql operation errors by replica
	// mode and operation type.
	DatabaseOperationsErrorsTotal *prometheus.CounterVec
}

// NewMetrics creates and registers all database metrics with the given registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	f := promauto.With(reg)

	return &Metrics{
		DatabaseOperationsLatency: f.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "unkey",
				Subsystem: "database",
				Name:      "operations_latency_seconds",
				Help:      "Histogram of database operation latencies in seconds.",
				Buckets:   latencyBuckets,
			},
			[]string{"replica", "operation", "status"},
		),

		DatabaseOperationsTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "database",
				Name:      "operations_total",
				Help:      "Total number of database operations processed.",
			},
			[]string{"replica", "operation", "status"},
		),

		DatabaseOperationsErrorsTotal: f.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "database",
				Name:      "operations_errors_total",
				Help:      "Total number of database operation errors.",
			},
			[]string{"replica", "operation"},
		),
	}
}
