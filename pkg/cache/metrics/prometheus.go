package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/vec"
)

var (
	// CacheHits tracks the number of cache read operations that found the requested item,
	// labeled by resource type. Use this to monitor cache hit rates and effectiveness.
	//
	// Example usage:
	//   metrics.CacheHits.WithLabelValues("user_profile").Inc()
	//   metrics.CacheHits.WithLabelValues("user_profile")
	CacheReads = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "reads_total",
			Help:      "Number of cache reads by resource type and hit status.",
		},
		vec.Dynamic("resource"),
		vec.Values("hit", "true", "false"),
	)

	// CacheDeleted tracks the number of items removed from the cache due to space constraints
	// or explicit deletion, labeled by resource type and reason.
	// Use this to monitor cache churn and capacity issues.
	//
	// Example usage:
	//   metrics.CacheDeleted.WithLabelValues("user_profile", "ttl").Inc()
	//   metrics.CacheDeleted.WithLabelValues("user_profile", "capacity").Set(float64(evictionCount))
	CacheDeleted = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "deleted_total",
			Help:      "Number of cache entries deleted by resource type and reason.",
		},
		vec.Dynamic("resource"),
		vec.Values("reason", "Replaced", "Size", "Expired", "Explicit"),
	)

	// CacheSize tracks the current number of items in the cache, labeled by resource type.
	// Use this to monitor cache utilization and growth patterns.
	//
	// Example usage:
	//   metrics.CacheSize.WithLabelValues("user_profile").Set(float64(cacheSize))
	CacheSize = vec.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "size",
			Help:      "Current number of entries in the cache by resource type.",
		},
		vec.Dynamic("resource"),
	)

	// CacheCapacity tracks the maximum number of items the cache can hold, labeled by resource type.
	// Use this to monitor cache utilization relative to its configured capacity.
	//
	// Example usage:
	//   metrics.CacheCapacity.WithLabelValues("user_profile").Set(float64(cacheCapacity))
	CacheCapacity = vec.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "capacity",
			Help:      "Maximum capacity of the cache by resource type.",
		},
		vec.Dynamic("resource"),
	)

	// CacheRevalidations counts the number of times the cache has been revalidated,
	// labeled by resource type. Use this to monitor cache refresh frequency and performance.
	//
	// Example usage:
	//   metrics.CacheRevalidations.WithLabelValues("user_profile").Inc()
	CacheRevalidations = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "revalidations_total",
			Help:      "Total number of cache revalidations by resource type.",
		},
		vec.Dynamic("resource"),
	)

	CacheRevalidationEnqueues = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "revalidation_enqueues_total",
			Help:      "Refresh enqueue attempts by outcome: enqueued, deduplicated (all keys already pending), queue_full, or closed. Each batch is one attempt.",
		},
		vec.Dynamic("resource"),
		vec.Values("outcome", "enqueued", "deduplicated", "queue_full", "closed"),
	)

	CacheRevalidationQueueDepth = vec.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "revalidation_queue_depth",
			Help:      "Queued jobs observed before enqueue attempts with new keys, excluding running jobs. Each batch occupies one slot.",
			Buckets:   []float64{0, 1, 5, 10, 50, 100, 250, 500, 750, 1000},
		},
		vec.Dynamic("resource"),
	)

	CacheRevalidationQueueWait = vec.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "revalidation_queue_wait_seconds",
			Help:      "Time from enqueue to execution for background refresh jobs. Each batch is one observation.",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
		},
		vec.Dynamic("resource"),
	)

	// CacheOperationDuration measures how long Get, SWR, and SWRWithFallback
	// take, from the call until the value is returned. A miss includes the
	// origin read the caller waits for.
	//
	// Labels:
	//   - resource: the cache's Config.Resource, for example "live_api_by_id".
	//   - operation: "get", "swr", or "swr_fallback".
	//   - status: "fresh" when the entry was within its fresh time, "stale"
	//     when it was past fresh but within stale, and "miss" when the cache
	//     had no usable entry.
	//
	// Buckets start at 100ns and each is 2 times the previous one, for 24
	// buckets: 100ns, 200ns, 400ns, 800ns, 1.6µs, 3.2µs, 6.4µs, and so on up
	// to about 0.84s. In-memory hits take a few hundred nanoseconds, so the
	// fine buckets below 10µs are the ones that show whether a change to the
	// cache makes hits faster or slower. The higher buckets cover misses that
	// wait for an origin read. Anything slower than 0.84s falls into +Inf.
	// Durations below about 100ns are not reliable, because reading the clock
	// and observing the histogram cost about that much.
	CacheOperationDuration = vec.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "operation_duration_seconds",
			Help:      "Duration of cache reads by resource type, operation (get, swr, swr_fallback), and status (fresh, stale, miss).",
			Buckets:   prometheus.ExponentialBuckets(100e-9, 2, 24),
		},
		vec.Dynamic("resource"),
		vec.Values("operation", "get", "swr", "swr_fallback"),
		vec.Values("status", "fresh", "stale", "miss"),
	)

	CacheRevalidationDuration = vec.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "revalidation_duration_seconds",
			Help:      "Execution time of background refresh jobs, including failed refreshes and excluding queue wait. Each batch is one observation.",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
		},
		vec.Dynamic("resource"),
	)

	// CacheRevalidationsErrorsTotal tracks the total number of cache revalidation errors,
	// labeled by resource type. Use this counter to monitor cache revalidation error rates.
	//
	// Example usage:
	//   metrics.CacheRevalidationsErrorsTotal.WithLabelValues("user_profile").Inc()
	CacheRevalidationsErrorsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "cache",
			Name:      "revalidations_errors_total",
			Help:      "Total number of cache revalidation errors by resource type.",
		},
		vec.Dynamic("resource"),
	)
)
