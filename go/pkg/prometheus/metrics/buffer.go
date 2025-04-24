/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains buffer-related metrics for tracking buffer usage.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// BufferState is a counter to track the number of times a buffer is used and
	// what state is triggered.
	//
	// Possible states are:
	// - "buffered": The item was added to the buffer.
	// - "dropped": The buffer was removed from the system.
	// - "closed": The buffer was closed.
	//
	// Example usage:
	//   metrics.BufferInserts.WithLabelValues(b.String(), "buffered").Inc()
	BufferState = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "api",
			Name:        "buffer_state_total",
			Help:        "Number of buffer inserts by name and state",
			ConstLabels: constLabels,
		},
		[]string{"name", "state"},
	)

	// BufferSize is a gauge to track the fill percentage of buffers and whether or not they
	// are configured to drop on overflow.
	//
	// Example usage:
	// 	 metrics.BufferSize.WithLabelValues(b.String(), "true").Set(float64(capacity)/float64(maxCapacity))
	BufferSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "unkey",
			Subsystem:   "api",
			Name:        "buffer_size_percentage",
			Help:        "Percentage of buffered fill capacity between 0.0 and 1.0",
			ConstLabels: constLabels,
		},
		[]string{"name", "drop"},
	)
)
