/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains rate-limiting metrics for tracking performance and behavior of the rate-limiting system.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (

	// RatelimitBuckets tracks how many rate-limit buckets are currently active.
	// This gauge helps monitor the resource usage of the rate-limiting system.
	//
	// Example usage:
	//   metrics.RatelimitBuckets.Set(float64(activeBuckets))
	RatelimitBuckets = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   "unkey",
			Subsystem:   "ratelimit",
			Name:        "buckets",
			Help:        "Current number of active rate-limit buckets.",
			ConstLabels: constLabels,
		},
	)

	// RatelimitWindows tracks how many rate-limit time windows are currently active.
	// This gauge helps monitor the resource usage of the rate-limiting system.
	//
	// Example usage:
	//   metrics.RatelimitWindows.Set(float64(activeWindows))
	RatelimitWindows = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   "unkey",
			Subsystem:   "ratelimit",
			Name:        "windows",
			Help:        "Current number of rate-limit windows.",
			ConstLabels: constLabels,
		},
	)

	// RatelimitBucketsCreated counts how many rate-limit buckets were created.
	// This counter helps track the creation rate and churn of rate-limit buckets.
	//
	// Example usage:
	//   metrics.RatelimitBucketsCreated.Inc()
	RatelimitBucketsCreated = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "ratelimit",
			Name:        "buckets_created_total",
			Help:        "Total number of rate-limit buckets created.",
			ConstLabels: constLabels,
		},
	)

	// RatelimitBucketsEvicted counts how many rate-limit buckets were evicted.
	// This counter helps track the eviction rate and lifetime of rate-limit buckets.
	//
	// Example usage:
	//   metrics.RatelimitBucketsEvicted.Inc()
	RatelimitBucketsEvicted = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "ratelimit",
			Name:        "buckets_evicted_total",
			Help:        "Total number of rate-limit buckets evicted.",
			ConstLabels: constLabels,
		},
	)

	// RatelimitWindowsCreated counts how many rate-limit time windows were created.
	// This counter helps track the creation rate and churn of rate-limit windows.
	//
	// Example usage:
	//   metrics.RatelimitWindowsCreated.Inc()
	RatelimitWindowsCreated = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "ratelimit",
			Name:        "windows_created_total",
			Help:        "Total number of rate-limit time windows created.",
			ConstLabels: constLabels,
		},
	)

	// RatelimitWindowsEvicted counts how many rate-limit time windows were evicted.
	// This counter helps track the eviction rate and lifetime of rate-limit windows.
	//
	// Example usage:
	//   metrics.RatelimitWindowsEvicted.Inc()
	RatelimitWindowsEvicted = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "ratelimit",
			Name:        "windows_evicted_total",
			Help:        "Total number of rate-limit time windows evicted.",
			ConstLabels: constLabels,
		},
	)

	// RatelimitDecisions counts how often rate-limit decisions were resolved locally or remotely
	// and whether they were allowed or denied.
	// This counter helps understand the distribution of decision-making in the cluster.
	//
	// Example usage:
	//   metrics.RatelimitDecisions.WithLabelValues("local", "allowed").Inc()
	//   metrics.RatelimitDecisions.WithLabelValues("origin", "denied").Inc()
	RatelimitDecision = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "ratelimit",
			Name:        "decisions_total",
			Help:        "Total number of rate-limit decisions.",
			ConstLabels: constLabels,
		},
		[]string{"source", "outcome"},
	)

	// RatelimitRefreshFromOrigin counts how often rate-limits were refreshed from an origin.
	// This counter helps understand the centralization of decision-making in the cluster.
	//
	// Example usage:
	//   metrics.RatelimitRefreshFromOrigin.Inc()
	RatelimitRefreshFromOrigin = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "ratelimit",
			Name:        "refresh_from_origin_total",
			Help:        "Total number of refreshes from an origin.",
			ConstLabels: constLabels,
		},
	)

	// RatelimitOriginSyncLatency measures the latency of origin sync operations.
	// This histogram helps track the performance and reliability of origin synchronization.
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.RatelimitOriginSyncLatency.Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	RatelimitOriginSyncLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "ratelimit",
			Name:        "origin_sync_latency_seconds",
			Help:        "Histogram of origin sync latencies in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
	)

	// RatelimitRefreshFromOriginErrorsTotal tracks the total number of errors when refreshing
	// rate-limits from an origin. Use this counter to monitor origin sync reliability.
	//
	// Example usage:
	//   metrics.RatelimitRefreshFromOriginErrorsTotal.Inc()
	RatelimitRefreshFromOriginErrorsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "ratelimit",
			Name:        "refresh_from_origin_errors_total",
			Help:        "Total number of errors when refreshing from an origin.",
			ConstLabels: constLabels,
		},
	)
)
