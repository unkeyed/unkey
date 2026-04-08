/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains rate-limiting metrics for tracking performance and behavior of the rate-limiting system.
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

// Metrics holds all Prometheus metrics for the rate-limiting system.
type Metrics struct {
	// RatelimitBuckets tracks how many rate-limit buckets are currently active.
	RatelimitBuckets prometheus.Gauge

	// RatelimitWindows tracks how many rate-limit time windows are currently active.
	RatelimitWindows prometheus.Gauge

	// RatelimitBucketsCreated counts how many rate-limit buckets were created.
	RatelimitBucketsCreated prometheus.Counter

	// RatelimitBucketsEvicted counts how many rate-limit buckets were evicted.
	RatelimitBucketsEvicted prometheus.Counter

	// RatelimitWindowsCreated counts how many rate-limit time windows were created.
	RatelimitWindowsCreated prometheus.Counter

	// RatelimitWindowsEvicted counts how many rate-limit time windows were evicted.
	RatelimitWindowsEvicted prometheus.Counter

	// RatelimitDecision counts how often rate-limit decisions were resolved locally or remotely
	// and whether they were allowed or denied.
	RatelimitDecision *prometheus.CounterVec

	// RatelimitRefreshFromOrigin counts how often rate-limits were refreshed from an origin.
	RatelimitRefreshFromOrigin prometheus.Counter

	// RatelimitOriginSyncLatency measures the latency of origin sync operations.
	RatelimitOriginSyncLatency prometheus.Histogram

	// RatelimitRefreshFromOriginErrorsTotal tracks the total number of errors when refreshing
	// rate-limits from an origin.
	RatelimitRefreshFromOriginErrorsTotal prometheus.Counter
}

// NewMetrics creates a new Metrics instance, registering all collectors with the given registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	factory := promauto.With(reg)

	return &Metrics{
		RatelimitBuckets: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "unkey",
				Subsystem: "ratelimit",
				Name:      "buckets",
				Help:      "Current number of active rate-limit buckets.",
			},
		),
		RatelimitWindows: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "unkey",
				Subsystem: "ratelimit",
				Name:      "windows",
				Help:      "Current number of rate-limit windows.",
			},
		),
		RatelimitBucketsCreated: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "ratelimit",
				Name:      "buckets_created_total",
				Help:      "Total number of rate-limit buckets created.",
			},
		),
		RatelimitBucketsEvicted: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "ratelimit",
				Name:      "buckets_evicted_total",
				Help:      "Total number of rate-limit buckets evicted.",
			},
		),
		RatelimitWindowsCreated: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "ratelimit",
				Name:      "windows_created_total",
				Help:      "Total number of rate-limit time windows created.",
			},
		),
		RatelimitWindowsEvicted: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "ratelimit",
				Name:      "windows_evicted_total",
				Help:      "Total number of rate-limit time windows evicted.",
			},
		),
		RatelimitDecision: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "ratelimit",
				Name:      "decisions_total",
				Help:      "Total number of rate-limit decisions.",
			},
			[]string{"source", "outcome"},
		),
		RatelimitRefreshFromOrigin: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "ratelimit",
				Name:      "refresh_from_origin_total",
				Help:      "Total number of refreshes from an origin.",
			},
		),
		RatelimitOriginSyncLatency: factory.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: "unkey",
				Subsystem: "ratelimit",
				Name:      "origin_sync_latency_seconds",
				Help:      "Histogram of origin sync latencies in seconds.",
				Buckets:   latencyBuckets,
			},
		),
		RatelimitRefreshFromOriginErrorsTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "ratelimit",
				Name:      "refresh_from_origin_errors_total",
				Help:      "Total number of errors when refreshing from an origin.",
			},
		),
	}
}
