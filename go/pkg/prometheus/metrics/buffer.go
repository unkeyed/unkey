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
	// BufferInserts are a counter to track the number of times a buffer is inserted into.
	// Use this with BufferDrops to establish a % rate of drop
	//
	// Example usage:
	//   metrics.BufferInserts.WithLabelValues(b.String()).Inc()
	BufferInserts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "api",
			Name:        "buffer_inserts_total",
			Help:        "Number of buffer inserts by type",
			ConstLabels: constLabels,
		},
		[]string{"type"},
	)

	// BufferDrops is a counter to track number of times an item has been dropped from a buffer
	//
	// Example usage:
	//   metrics.BufferDrops.WithLabelValues(b.String()).Inc()
	BufferDrops = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "api",
			Name:        "buffer_drops_total",
			Help:        "Number of buffer drops by type",
			ConstLabels: constLabels,
		},
		[]string{"type"},
	)

	// BufferConsumed
	//
	// Example usage:
	//   metrics.BufferConsumed.WithLabelValues(b.String()).Inc()
	BufferConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "api",
			Name:        "buffer_consumed_total",
			Help:        "Number of times the buffer has been consumed from by type",
			ConstLabels: constLabels,
		},
		[]string{"type"},
	)
)
