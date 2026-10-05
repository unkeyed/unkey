package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var (
	// CacheHits tracks the number of cache read operations that found the requested item,
	// labeled by resource type. Use this to monitor cache hit rates and effectiveness.
	//
	// Example usage:
	//   metrics.CacheHits.WithLabelValues("user_profile").Inc()
	//   metrics.CacheHits.WithLabelValues("user_profile")
	CacheReads = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "reads_total",
			Help:      "Number of cache reads by resource type and hit status.",
		},
		[]string{"resource", "hit"},
	)

	// CacheWrites tracks the number of cache write operations, labeled by resource type.
	// Use this to monitor write pressure on the cache.
	//
	// Example usage:
	//   metrics.CacheWrites.WithLabelValues("user_profile").Inc()
	//   metrics.CacheWrites.WithLabelValues("user_profile").Set(float64(writeCount))
	CacheWrites = lazy.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "writes",
			Help:      "Number of cache writes by resource type.",
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
	CacheDeleted = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "deleted_total",
			Help:      "Number of cache entries deleted by resource type and reason.",
		},
		[]string{"resource", "reason"},
	)

	// CacheSize tracks the current number of items in the cache, labeled by resource type.
	// Use this to monitor cache utilization and growth patterns.
	//
	// Example usage:
	//   metrics.CacheSize.WithLabelValues("user_profile").Set(float64(cacheSize))
	CacheSize = lazy.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "size",
			Help:      "Current number of entries in the cache by resource type.",
		},
		[]string{"resource"},
	)

	// CacheCapacity tracks the maximum number of items the cache can hold, labeled by resource type.
	// Use this to monitor cache utilization relative to its configured capacity.
	//
	// Example usage:
	//   metrics.CacheCapacity.WithLabelValues("user_profile").Set(float64(cacheCapacity))
	CacheCapacity = lazy.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "capacity",
			Help:      "Maximum capacity of the cache by resource type.",
		},
		[]string{"resource"},
	)

	// CacheRevalidations counts the number of times the cache has been revalidated,
	// labeled by resource type. Use this to monitor cache refresh frequency and performance.
	//
	// Example usage:
	//   metrics.CacheRevalidations.WithLabelValues("user_profile").Inc()
	CacheRevalidations = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "revalidations_total",
			Help:      "Total number of cache revalidations by resource type.",
		},
		[]string{"resource"},
	)

	CacheRevalidationEnqueues = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "revalidation_enqueues_total",
			Help:      "Refresh enqueue attempts by outcome: enqueued, deduplicated (all keys already pending), queue_full, or closed. Each batch is one attempt.",
		},
		[]string{"resource", "outcome"},
	)

	CacheRevalidationQueueDepth = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "revalidation_queue_depth",
			Help:      "Queued jobs observed before enqueue attempts with new keys, excluding running jobs. Each batch occupies one slot.",
			Buckets:   []float64{0, 1, 5, 10, 50, 100, 250, 500, 750, 1000},
		},
		[]string{"resource"},
	)

	CacheRevalidationQueueWait = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "revalidation_queue_wait_seconds",
			Help:      "Time from enqueue to execution for background refresh jobs. Each batch is one observation.",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
		},
		[]string{"resource"},
	)

	CacheRevalidationDuration = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "revalidation_duration_seconds",
			Help:      "Execution time of background refresh jobs, including failed refreshes and excluding queue wait. Each batch is one observation.",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
		},
		[]string{"resource"},
	)

	// CacheReadsErrorsTotal tracks the total number of cache read errors,
	// labeled by resource type. Use this counter to monitor cache read error rates.
	//
	// Example usage:
	//   metrics.CacheReadsErrorsTotal.WithLabelValues("user_profile").Inc()
	CacheReadsErrorsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "reads_errors_total",
			Help:      "Total number of cache read errors by resource type.",
		},
		[]string{"resource"},
	)

	// CacheRevalidationsErrorsTotal tracks the total number of cache revalidation errors,
	// labeled by resource type. Use this counter to monitor cache revalidation error rates.
	//
	// Example usage:
	//   metrics.CacheRevalidationsErrorsTotal.WithLabelValues("user_profile").Inc()
	CacheRevalidationsErrorsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "revalidations_errors_total",
			Help:      "Total number of cache revalidation errors by resource type.",
		},
		[]string{"resource"},
	)
)
