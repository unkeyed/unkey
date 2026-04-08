/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains buffer-related metrics for tracking buffer usage.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus collectors for the buffer package.
type Metrics struct {
	// BufferState is a counter to track the number of times a buffer is used and
	// what state is triggered.
	//
	// Possible states are:
	// - "buffered": The item was added to the buffer.
	// - "dropped": The buffer was removed from the system.
	// - "closed": The buffer was closed.
	//
	// Example usage:
	//   m.BufferState.WithLabelValues(b.String(), "buffered").Inc()
	BufferState *prometheus.CounterVec

	// BufferSize is a gauge to track the fill percentage of buffers and whether or not they
	// are configured to drop on overflow.
	//
	// Example usage:
	// 	 m.BufferSize.WithLabelValues(b.String(), "true").Set(float64(capacity)/float64(maxCapacity))
	BufferSize *prometheus.GaugeVec

	// BufferErrorsTotal tracks the total number of buffer operation errors,
	// labeled by buffer name and error type. Use this counter to monitor buffer error rates.
	//
	// Example usage:
	//   m.BufferErrorsTotal.WithLabelValues("batch_writer", "write_failed").Inc()
	BufferErrorsTotal *prometheus.CounterVec
}

// NoopMetrics returns a Metrics instance registered to a discarded registry.
// Metrics are recorded but not exported, making this safe for tests and optional instrumentation.
func NoopMetrics() *Metrics {
	return NewMetrics(prometheus.NewRegistry())
}

// NewMetrics registers and returns a new Metrics instance with the given registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	factory := promauto.With(reg)

	return &Metrics{
		BufferState: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "buffer",
				Name:      "state_total",
				Help:      "Number of buffer inserts by name and state",
			},
			[]string{"name", "state"},
		),
		BufferSize: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "unkey",
				Subsystem: "buffer",
				Name:      "size_percentage",
				Help:      "Percentage of buffered fill capacity between 0.0 and 1.0",
			},
			[]string{"name", "drop"},
		),
		BufferErrorsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "buffer",
				Name:      "errors_total",
				Help:      "Total number of buffer operation errors by name and state.",
			},
			[]string{"name", "state"},
		),
	}
}
