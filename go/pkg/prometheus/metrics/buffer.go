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
	// Example usage:
	//   metrics.BufferInserts.WithLabelValues(b.String(), "add").Inc()
	BufferState = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "api",
			Name:        "buffer_state_total",
			Help:        "Number of buffer inserts by type",
			ConstLabels: constLabels,
		},
		[]string{"type", "state"},
	)

	// BufferSize is a gauge to track the size of buffers and whether or not they
	// are configured to drop on overflow.
	//
	// Example usage:
	// 	 metrics.BufferSize.WithLabelValues(b.String(), "true").Set(float64(capacity))
	BufferSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "unkey",
			Subsystem:   "api",
			Name:        "buffer_size_total",
			Help:        "Size of buffers by type and if they're configured to drop",
			ConstLabels: constLabels,
		},
		[]string{"type", "drop"},
	)
)
