/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains cache-related metrics for tracking cache efficiency, performance, and resource usage.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CacheHits tracks the number of cache read operations that found the requested item,
	// labeled by resource type. Use this to monitor cache hit rates and effectiveness.
	//
	// Example usage:
	//   metrics.CacheHits.WithLabelValues("user_profile").Inc()
	//   metrics.CacheHits.WithLabelValues("user_profile")
	CacheReads = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem:   "cache",
			Name:        "reads_total",
			Help:        "Number of cache reads by resource type and hit status.",
			ConstLabels: constLabels,
		},
		[]string{"resource", "hit"},
	)

	// CacheWrites tracks the number of cache write operations, labeled by resource type.
	// Use this to monitor write pressure on the cache.
	//
	// Example usage:
	//   metrics.CacheWrites.WithLabelValues("user_profile").Inc()
	//   metrics.CacheWrites.WithLabelValues("user_profile").Set(float64(writeCount))
	CacheWrites = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem:   "cache",
			Name:        "writes",
			Help:        "Number of cache writes by resource type.",
			ConstLabels: constLabels,
		},
		[]string{"resource"},
	)

	// CacheDeleted tracks the number of items removed from the cache due to space constraints
	// or explicit deletion, labeled by resource type and reason.
	// Use this to monitor cache churn and capacity issues.
	//
	// Example usage:
	//   metrics.CacheDeleted.WithLabelValues("user_profile", "ttl").Inc()
	//   metrics.CacheDeleted.WithLabelValues("user_profile", "capacity").Set(float64(evictionCount))
	CacheDeleted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem:   "cache",
			Name:        "deleted_total",
			Help:        "Number of cache entries deleted by resource type and reason.",
			ConstLabels: constLabels,
		},
		[]string{"resource", "reason"},
	)

	// CacheSize tracks the current number of items in the cache, labeled by resource type.
	// Use this to monitor cache utilization and growth patterns.
	//
	// Example usage:
	//   metrics.CacheSize.WithLabelValues("user_profile").Set(float64(cacheSize))
	CacheSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem:   "cache",
			Name:        "size",
			Help:        "Current number of entries in the cache by resource type.",
			ConstLabels: constLabels,
		},
		[]string{"resource"},
	)

	// CacheCapacity tracks the maximum number of items the cache can hold, labeled by resource type.
	// Use this to monitor cache utilization relative to its configured capacity.
	//
	// Example usage:
	//   metrics.CacheCapacity.WithLabelValues("user_profile").Set(float64(cacheCapacity))
	CacheCapacity = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem:   "cache",
			Name:        "capacity",
			Help:        "Maximum capacity of the cache by resource type.",
			ConstLabels: constLabels,
		},
		[]string{"resource"},
	)

	// CacheRevalidations counts the number of times the cache has been revalidated,
	// labeled by resource type. Use this to monitor cache refresh frequency and performance.
	//
	// Example usage:
	//   metrics.CacheRevalidations.WithLabelValues("user_profile").Inc()
	CacheRevalidations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem:   "cache",
			Name:        "revalidations_total",
			Help:        "Total number of cache revalidations by resource type.",
			ConstLabels: constLabels,
		},
		[]string{"resource"},
	)
)
