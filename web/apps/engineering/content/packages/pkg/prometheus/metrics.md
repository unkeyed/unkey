---
title: metrics
description: "provides Prometheus metric collectors for monitoring application performance"
---

Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains batch-related metrics for tracking batch processing performance.

Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains buffer-related metrics for tracking buffer usage.

Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains cache-related metrics for tracking cache efficiency, performance, and resource usage.

Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains ClickHouse proxy-related metrics for tracking event ingestion.

Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains database-related metrics for tracking database operation performance, usage patterns, and latency across primary and read-only replicas.

Package metrics provides Prometheus metric collectors for monitoring Unkey services.

All metrics are registered automatically via \[promauto] and use the "unkey" namespace. Metrics include constant labels for region and version to support multi-region deployments.

### Metric Organization

Metrics are organized by subsystem:

  - Batch processing: \[BatchSizeDistribution], \[BatchOperationsTotal], \[BatchItemsProcessedTotal]
  - Buffer management: \[BufferState], \[BufferSize], \[BufferErrorsTotal]
  - Caching: \[CacheReads], \[CacheWrites], \[CacheSize], \[CacheCapacity]
  - Circuit breaker: \[CircuitBreakerRequests], \[CircuitBreakerErrorsTotal]
  - ClickHouse proxy: \[ChproxyRequestsTotal], \[ChproxyRowsTotal], \[ChproxyErrorsTotal]
  - Database operations: \[DatabaseOperationsLatency], \[DatabaseOperationsTotal]
  - HTTP requests: \[HTTPRequestLatency], \[HTTPRequestTotal], \[HTTPRequestBodySize]
  - Key verification: \[KeyVerificationsTotal], \[KeyVerificationErrorsTotal]
  - Krane orchestration: \[KraneControlPlaneReconnectsTotal], \[KraneReconcileOperationsTotal], \[KraneSecretsRequestsTotal]
  - Rate limiting: \[RatelimitDecision], \[RatelimitBuckets], \[RatelimitWindows]
  - Usage limiting: \[UsagelimiterDecisions], \[UsagelimiterReplayOperations]
  - Internal: \[PanicsTotal]

### Usage

Import the package and use the metric collectors directly:

	import "github.com/unkeyed/unkey/pkg/prometheus/metrics"

	// Increment a counter
	metrics.HTTPRequestTotal.WithLabelValues("GET", "/v1/keys", "200").Inc()

	// Observe a latency
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	    metrics.HTTPRequestLatency.WithLabelValues("GET", "/v1/keys", "200").Observe(v)
	}))
	defer timer.ObserveDuration()

	// Set a gauge
	metrics.CacheSize.WithLabelValues("api_keys").Set(float64(cacheSize))

### Label Conventions

Common label patterns used across metrics:

  - "status": Operation outcome, typically "success" or "error"
  - "resource": Resource type being operated on (e.g., "user\_profile", "api\_key")
  - "replica": Database replica type, "rw" for primary, "ro" for read-only
  - "method", "path", "status": HTTP request attributes

Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains HTTP-related metrics for tracking request latencies and counts.

Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains Key-Verification-related metrics for tracking what keys do.

Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains krane-specific metrics for tracking the distributed container orchestration agent's health, control plane connectivity, Kubernetes API interactions, reconciliation operations, and secrets decryption.

Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains a metric for tracking panics across http handlers.

Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains rate-limiting metrics for tracking performance and behavior of the rate-limiting system.

Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains usage limiter metrics for tracking performance and behavior of the credit-based usage limiting system.

## Variables

```go
var (
	// BatchSizeDistribution tracks the size of batches when they're flushed as a histogram,
	// labeled by batch name and trigger type.
	// This helps understand batching efficiency and whether batches are being flushed
	// due to size limits or time intervals.
	//
	// Possible trigger values are:
	// - "size_limit": The batch was flushed because it reached the maximum size.
	// - "time_interval": The batch was flushed because the flush interval elapsed.
	// - "close": The batch was flushed because the processor was closed.
	//
	// Example usage:
	//   metrics.BatchSizeDistribution.WithLabelValues("database_writes", "size_limit").Observe(float64(len(batch)))
	BatchSizeDistribution = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "batch",
			Name:        "size_distribution",
			Help:        "Distribution of batch sizes when flushed",
			Buckets:     batchSizeBuckets,
			ConstLabels: constLabels,
		},
		[]string{"name", "trigger"},
	)

	// BatchOperationsTotal tracks the total number of batch flush operations,
	// labeled by batch name, trigger type, and success status.
	// Use this counter to monitor batch processing patterns and error rates.
	//
	// Possible trigger values are:
	// - "size_limit": The batch was flushed because it reached the maximum size.
	// - "time_interval": The batch was flushed because the flush interval elapsed.
	// - "close": The batch was flushed because the processor was closed.
	//
	// Possible status values are:
	// - "success": The batch flush operation completed successfully.
	// - "error": The batch flush operation encountered an error.
	//
	// Example usage:
	//   metrics.BatchOperationsTotal.WithLabelValues("database_writes", "size_limit", "success").Inc()
	//   metrics.BatchOperationsTotal.WithLabelValues("log_entries", "time_interval", "error").Inc()
	BatchOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "batch",
			Name:        "operations_total",
			Help:        "Total number of batch flush operations processed",
			ConstLabels: constLabels,
		},
		[]string{"name", "trigger", "status"},
	)

	// BatchItemsProcessedTotal tracks the total number of items processed through all batches,
	// labeled by batch name.
	// This is the core throughput metric for understanding how many individual items
	// are being processed by the batch system.
	//
	// Example usage:
	//   metrics.BatchItemsProcessedTotal.WithLabelValues("database_writes").Add(float64(len(batch)))
	BatchItemsProcessedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "batch",
			Name:        "items_processed_total",
			Help:        "Total number of items processed through batches",
			ConstLabels: constLabels,
		},
		[]string{"name"},
	)

	// BatchItemsProcessedErrorsTotal tracks the total number of items that resulted in errors
	// during batch processing, labeled by batch name.
	// Use this counter to monitor error rates in batch processing and identify problematic batches.
	//
	// Example usage:
	//   metrics.BatchItemsProcessedErrorsTotal.WithLabelValues("database_writes").Add(float64(errorCount))
	BatchItemsProcessedErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "batch",
			Name:        "items_processed_errors_total",
			Help:        "Total number of items processed through batches that resulted in an error",
			ConstLabels: constLabels,
		},
		[]string{"name"},
	)
)
```

```go
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
			Subsystem:   "buffer",
			Name:        "state_total",
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
			Subsystem:   "buffer",
			Name:        "size_percentage",
			Help:        "Percentage of buffered fill capacity between 0.0 and 1.0",
			ConstLabels: constLabels,
		},
		[]string{"name", "drop"},
	)

	// BufferErrorsTotal tracks the total number of buffer operation errors,
	// labeled by buffer name and error type. Use this counter to monitor buffer error rates.
	//
	// Example usage:
	//   metrics.BufferErrorsTotal.WithLabelValues("batch_writer", "write_failed").Inc()
	BufferErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "buffer",
			Name:        "errors_total",
			Help:        "Total number of buffer operation errors by name and state.",
			ConstLabels: constLabels,
		},
		[]string{"name", "state"},
	)
)
```

```go
var (
	// CacheHits tracks the number of cache read operations that found the requested item,
	// labeled by resource type. Use this to monitor cache hit rates and effectiveness.
	//
	// Example usage:
	//   metrics.CacheHits.WithLabelValues("user_profile").Inc()
	//   metrics.CacheHits.WithLabelValues("user_profile")
	CacheReads = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
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
			Namespace:   "unkey",
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
			Namespace:   "unkey",
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
			Namespace:   "unkey",
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
			Namespace:   "unkey",
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
			Namespace:   "unkey",
			Subsystem:   "cache",
			Name:        "revalidations_total",
			Help:        "Total number of cache revalidations by resource type.",
			ConstLabels: constLabels,
		},
		[]string{"resource"},
	)

	// CacheReadsErrorsTotal tracks the total number of cache read errors,
	// labeled by resource type. Use this counter to monitor cache read error rates.
	//
	// Example usage:
	//   metrics.CacheReadsErrorsTotal.WithLabelValues("user_profile").Inc()
	CacheReadsErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "cache",
			Name:        "reads_errors_total",
			Help:        "Total number of cache read errors by resource type.",
			ConstLabels: constLabels,
		},
		[]string{"resource"},
	)

	// CacheRevalidationsErrorsTotal tracks the total number of cache revalidation errors,
	// labeled by resource type. Use this counter to monitor cache revalidation error rates.
	//
	// Example usage:
	//   metrics.CacheRevalidationsErrorsTotal.WithLabelValues("user_profile").Inc()
	CacheRevalidationsErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "cache",
			Name:        "revalidations_errors_total",
			Help:        "Total number of cache revalidation errors by resource type.",
			ConstLabels: constLabels,
		},
		[]string{"resource"},
	)
)
```

```go
var (
	// ChproxyRequestsTotal tracks the total number of chproxy requests received, labeled by endpoint.
	// Use this counter to monitor ingestion traffic patterns.
	//
	// Example usage:
	//   metrics.ChproxyRequestsTotal.WithLabelValues("verifications").Inc()
	ChproxyRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "chproxy",
			Name:        "requests_total",
			Help:        "Total number of ClickHouse proxy requests processed.",
			ConstLabels: constLabels,
		},
		[]string{"endpoint"},
	)

	// ChproxyErrorsTotal tracks the total number of errors encountered by ClickHouse proxy,
	// labeled by endpoint. Use this counter to monitor error rates and identify problematic endpoints.
	//
	// Example usage:
	//   metrics.ChproxyErrorsTotal.WithLabelValues("verifications").Inc()
	ChproxyErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "chproxy",
			Name:        "errors_total",
			Help:        "Total number of errors encountered by ClickHouse proxy.",
			ConstLabels: constLabels,
		},
		[]string{"endpoint"},
	)

	// ChproxyRowsTotal tracks the total number of rows/events received in chproxy requests.
	// Use this counter to monitor data volume and ingestion patterns.
	//
	// Example usage:
	//   metrics.ChproxyRowsTotal.WithLabelValues("verifications").Add(float64(len(events)))
	ChproxyRowsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "chproxy",
			Name:        "rows_total",
			Help:        "Total number of rows/events processed by ClickHouse proxy.",
			ConstLabels: constLabels,
		},
		[]string{"endpoint"},
	)

	// ChproxyRowsErrorsTotal tracks the total number of row processing errors in ClickHouse proxy,
	// labeled by endpoint. Use this counter to monitor row processing error rates.
	//
	// Example usage:
	//   metrics.ChproxyRowsErrorsTotal.WithLabelValues("verifications").Inc()
	ChproxyRowsErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "chproxy",
			Name:        "rows_errors_total",
			Help:        "Total number of row processing errors in ClickHouse proxy.",
			ConstLabels: constLabels,
		},
		[]string{"endpoint"},
	)
)
```

```go
var (
	// CircuitBreakerRequests tracks the number of requests made to the circuit breaker.
	// This counter helps monitor the usage and performance of the circuit breaker.
	//
	// Example usage:
	//   metrics.CircuitBreakerRequests.WithLabelValues("my_circuit_breaker", "open").Inc()
	CircuitBreakerRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "unkey",
		Subsystem:   "circuitbreaker",
		Name:        "requests_total",
		Help:        "Tracks the number of requests made to the circuitbreaker by state.",
		ConstLabels: constLabels,
	}, []string{"service", "action"})

	// CircuitBreakerErrorsTotal tracks the total number of circuit breaker errors,
	// labeled by service and action. Use this counter to monitor circuit breaker error rates.
	//
	// Example usage:
	//   metrics.CircuitBreakerErrorsTotal.WithLabelValues("database", "timeout").Inc()
	CircuitBreakerErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "unkey",
		Subsystem:   "circuitbreaker",
		Name:        "errors_total",
		Help:        "Total number of circuit breaker errors by service and action.",
		ConstLabels: constLabels,
	}, []string{"service", "action"})
)
```

```go
var (
	// DatabaseOperationsLatency tracks database operation latencies as a histogram,
	// labeled by replica type (rw/ro), operation type, and success status.
	// This collector uses predefined buckets optimized for typical database operation latencies.
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.DatabaseOperationsLatency.WithLabelValues("rw", "exec", "success").Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	DatabaseOperationsLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "database",
			Name:        "operations_latency_seconds",
			Help:        "Histogram of database operation latencies in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
		[]string{"replica", "operation", "status"},
	)

	// DatabaseOperationTotal tracks the total number of database operations,
	// labeled by replica type (rw/ro), operation type, and success status.
	// Use this counter to monitor database traffic patterns and error rates.
	//
	// Example usage:
	//   metrics.DatabaseOperationTotal.WithLabelValues("rw", "exec", "success").Inc()
	//   metrics.DatabaseOperationTotal.WithLabelValues("ro", "query", "error").Inc()
	DatabaseOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "database",
			Name:        "operations_total",
			Help:        "Total number of database operations processed.",
			ConstLabels: constLabels,
		},
		[]string{"replica", "operation", "status"},
	)

	// DatabaseOperationsErrorsTotal tracks the total number of database operation errors,
	// labeled by replica type (rw/ro), and operation type.
	// Use this counter to monitor database error rates and identify problematic operations.
	//
	// Example usage:
	//   metrics.DatabaseOperationsErrorsTotal.WithLabelValues("rw", "exec").Inc()
	DatabaseOperationsErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "unkey",
		Subsystem:   "database",
		Name:        "operations_errors_total",
		Help:        "Total number of database operation errors.",
		ConstLabels: constLabels,
	}, []string{"replica", "operation"})
)
```

```go
var (
	// HTTPRequestLatency tracks HTTP request latencies as a histogram, labeled by method, path, and status.
	// This collector uses predefined buckets optimized for typical web service latencies,
	// ranging from 1ms to 10s.
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.HTTPRequestLatency.WithLabelValues("GET", "/users", "200").Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	HTTPRequestLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "http",
			Name:        "request_latency_seconds",
			Help:        "Histogram of HTTP request latencies in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestTotal tracks the number of HTTP requests handled, labeled by method, path, and status.
	// Use this counter to monitor API traffic patterns and error rates.
	//
	// Example usage:
	//   metrics.HTTPRequestTotal.WithLabelValues("GET", "/users", "200").Inc()
	HTTPRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "http",
			Name:        "requests_total",
			Help:        "Total number of HTTP requests processed.",
			ConstLabels: constLabels,
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestErrorTotal tracks the total number of HTTP request errors,
	// labeled by method, path, and status. Use this counter to monitor error rates by endpoint.
	//
	// Example usage:
	//   metrics.HTTPRequestErrorTotal.WithLabelValues("POST", "/api/keys", "500").Inc()
	HTTPRequestErrorTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "http",
			Name:        "requests_errors_total",
			Help:        "Total number of HTTP request errors.",
			ConstLabels: constLabels,
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestBodySize tracks the distribution of HTTP request body sizes as a histogram,
	// labeled by method, path, and status. This helps monitor payload sizes and identify potentially
	// problematic large requests.
	//
	// Example usage:
	//   metrics.HTTPRequestBodySize.WithLabelValues("POST", "/api/upload", "200").Observe(float64(bodySize))
	HTTPRequestBodySize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "http",
			Name:        "request_body_size_bytes",
			Help:        "Histogram of HTTP request body sizes in bytes.",
			Buckets:     bodySizeBuckets,
			ConstLabels: constLabels,
		},
		[]string{"method", "path", "status"},
	)
)
```

```go
var (
	// KeyVerificationsTotal tracks the number of key verifications handled, labeled by type and outcome.
	// The type should be either "root_key" or "key"
	// Use this counter to monitor API traffic patterns.
	//
	// Example usage:
	//   metrics.KeyVerificationsTotal.WithLabelValues("root_key", "VALID").Inc()
	KeyVerificationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "key",
			Name:        "verifications_total",
			Help:        "Total number of Key verifications processed.",
			ConstLabels: constLabels,
		},
		[]string{"type", "code"},
	)

	// KeyVerificationErrorsTotal tracks the number of errors in key verifications.
	// These are not errors in the keys themselves like "FORBIDDEN", or "RATE_LIMITED" but errors in
	// program functionality. Use this with the unkey_key_verifications_total metric to calculate
	// the error rate.
	//
	// Example usage:
	//   metrics.KeyVerificationErrorsTotal.WithLabelValues("root_key").Inc()
	KeyVerificationErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "key",
			Name:        "verification_errors_total",
			Help:        "Total number of key verification errors",
			ConstLabels: constLabels,
		},
		[]string{"type"},
	)
)
```

```go
var (

	// KraneControlPlaneReconnectsTotal counts stream reconnection attempts to the control plane.
	// An elevated reconnect rate indicates network instability or control plane issues.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//
	// Example usage:
	//   metrics.KraneControlPlaneReconnectsTotal.WithLabelValues("deployments").Inc()
	KraneControlPlaneReconnectsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "controlplane_reconnects_total",
			Help:        "Total number of control plane stream reconnection attempts.",
			ConstLabels: constLabels,
		},
		[]string{"controller"},
	)

	// KraneControlPlaneRPCRequestsTotal counts outbound RPC requests to the control plane.
	// Use this to monitor control plane communication health and error rates.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//   - "method": RPC method name (e.g., "WatchDeployments", "ReportDeploymentStatus")
	//   - "result": Outcome ("success" or "error")
	//
	// Example usage:
	//   metrics.KraneControlPlaneRPCRequestsTotal.WithLabelValues("deployments", "ReportDeploymentStatus", "success").Inc()
	KraneControlPlaneRPCRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "controlplane_rpc_requests_total",
			Help:        "Total number of outbound RPC requests to the control plane.",
			ConstLabels: constLabels,
		},
		[]string{"controller", "method", "result"},
	)

	// KraneControlPlaneRPCDurationSeconds tracks latency of outbound RPCs to the control plane.
	// Use this to detect control plane slowdowns before they cause failures.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//   - "method": RPC method name
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.KraneControlPlaneRPCDurationSeconds.WithLabelValues("deployments", "ReportDeploymentStatus").Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	KraneControlPlaneRPCDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "controlplane_rpc_duration_seconds",
			Help:        "Histogram of outbound RPC latencies to the control plane in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
		[]string{"controller", "method"},
	)

	// KraneK8sRequestsTotal counts Kubernetes API requests made by krane.
	// Use this to monitor K8s API health and distinguish K8s failures from control plane failures.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//   - "verb": K8s API verb ("get", "list", "patch", "delete", "watch")
	//   - "resource": K8s resource type ("replicaset", "deployment", "service", "namespace", "pod")
	//   - "result": Outcome ("success" or "error")
	//
	// Example usage:
	//   metrics.KraneK8sRequestsTotal.WithLabelValues("deployments", "patch", "replicaset", "success").Inc()
	KraneK8sRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "k8s_requests_total",
			Help:        "Total number of Kubernetes API requests.",
			ConstLabels: constLabels,
		},
		[]string{"controller", "verb", "resource", "result"},
	)

	// KraneK8sDurationSeconds tracks latency of Kubernetes API requests.
	// Use this to detect K8s API slowdowns that could affect reconciliation.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//   - "verb": K8s API verb
	//   - "resource": K8s resource type
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.KraneK8sDurationSeconds.WithLabelValues("deployments", "patch", "replicaset").Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	KraneK8sDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "k8s_duration_seconds",
			Help:        "Histogram of Kubernetes API request latencies in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
		[]string{"controller", "verb", "resource"},
	)

	// KraneReconcileOperationsTotal counts reconciliation operations (apply/delete).
	// Use this to monitor reconciliation throughput and error rates per workspace.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//   - "operation": Operation type ("apply" or "delete")
	//   - "result": Outcome ("success" or "error")
	//   - "workspace_id": The workspace ID (may be removed if cardinality becomes problematic)
	//
	// Example usage:
	//   metrics.KraneReconcileOperationsTotal.WithLabelValues("deployments", "apply", "success", "ws_123").Inc()
	KraneReconcileOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "reconcile_operations_total",
			Help:        "Total number of reconciliation operations (apply/delete).",
			ConstLabels: constLabels,
		},
		[]string{"controller", "operation", "result", "workspace_id"},
	)

	// KraneReconcileDurationSeconds tracks latency of reconciliation operations.
	// Use this to detect slow reconciliations that could delay deployments.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//   - "operation": Operation type ("apply" or "delete")
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.KraneReconcileDurationSeconds.WithLabelValues("deployments", "apply").Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	KraneReconcileDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "reconcile_duration_seconds",
			Help:        "Histogram of reconciliation operation latencies in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
		[]string{"controller", "operation"},
	)

	// KraneResyncCorrectionsTotal counts corrections made by the periodic resync loop.
	// High values indicate the streaming path is missing events and resync is compensating.
	// This is a proxy metric for streaming reliability.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//
	// Example usage:
	//   metrics.KraneResyncCorrectionsTotal.WithLabelValues("deployments").Inc()
	KraneResyncCorrectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "resync_corrections_total",
			Help:        "Total number of corrections made by the resync loop (indicates missed streaming events).",
			ConstLabels: constLabels,
		},
		[]string{"controller"},
	)

	// KraneResyncDurationSeconds tracks how long each resync loop iteration takes.
	// Use this to detect resync slowdowns that could indicate cluster growth or API issues.
	//
	// Labels:
	//   - "controller": The controller type ("deployments" or "sentinels")
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.KraneResyncDurationSeconds.WithLabelValues("deployments").Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	KraneResyncDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "resync_duration_seconds",
			Help:        "Histogram of resync loop iteration durations in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
		[]string{"controller"},
	)

	// KraneSecretsRequestsTotal counts secrets decryption requests.
	// Use this to monitor secrets service usage and success rates.
	//
	// Labels:
	//   - "result": Outcome ("success" or "error")
	//
	// Example usage:
	//   metrics.KraneSecretsRequestsTotal.WithLabelValues("success").Inc()
	KraneSecretsRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "secrets_requests_total",
			Help:        "Total number of secrets decryption requests.",
			ConstLabels: constLabels,
		},
		[]string{"result"},
	)

	// KraneSecretsErrorsTotal counts secrets service errors by type.
	// Use this to distinguish authentication failures from decryption failures.
	//
	// Labels:
	//   - "type": Error type ("unauthenticated" or "decrypt_failed")
	//
	// Example usage:
	//   metrics.KraneSecretsErrorsTotal.WithLabelValues("unauthenticated").Inc()
	KraneSecretsErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "secrets_errors_total",
			Help:        "Total number of secrets service errors by type.",
			ConstLabels: constLabels,
		},
		[]string{"type"},
	)

	// KraneSecretsDurationSeconds tracks latency of secrets decryption requests.
	// Use this to detect vault or storage slowdowns affecting pod startup.
	//
	// Example usage:
	//   timer := prometheus.NewTimer(metrics.KraneSecretsDurationSeconds)
	//   defer timer.ObserveDuration()
	KraneSecretsDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "secrets_duration_seconds",
			Help:        "Histogram of secrets decryption request latencies in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
	)

	// KraneRPCServerRequestsTotal counts inbound RPC requests to krane's gRPC server.
	// Use this to monitor server load and error rates by method.
	//
	// Labels:
	//   - "method": RPC method name (e.g., "DecryptSecretsBlob")
	//   - "code": Connect/gRPC status code (e.g., "ok", "unauthenticated", "internal")
	//
	// Example usage:
	//   metrics.KraneRPCServerRequestsTotal.WithLabelValues("DecryptSecretsBlob", "ok").Inc()
	KraneRPCServerRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "rpc_server_requests_total",
			Help:        "Total number of inbound RPC requests to krane server.",
			ConstLabels: constLabels,
		},
		[]string{"method", "code"},
	)

	// KraneRPCServerDurationSeconds tracks latency of inbound RPC requests.
	// Use this to monitor server-side latency by method.
	//
	// Labels:
	//   - "method": RPC method name
	//
	// Example usage:
	//   timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
	//       metrics.KraneRPCServerDurationSeconds.WithLabelValues("DecryptSecretsBlob").Observe(v)
	//   }))
	//   defer timer.ObserveDuration()
	KraneRPCServerDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "krane",
			Name:        "rpc_server_duration_seconds",
			Help:        "Histogram of inbound RPC request latencies in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
		[]string{"method"},
	)
)
```

```go
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
	//   metrics.RatelimitDecisions.WithLabelValues("local", "passed").Inc()
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
```

```go
var (
	// UsagelimiterDecisions counts usage limiter decisions by outcome (allowed/denied) and source (redis/db)
	// This counter helps understand the distribution of decisions and fallback patterns.
	//
	// Example usage:
	//   metrics.UsagelimiterDecisions.WithLabelValues("redis", "allowed").Inc()
	//   metrics.UsagelimiterDecisions.WithLabelValues("db", "denied").Inc()
	UsagelimiterDecisions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "usagelimiter",
			Name:        "decisions_total",
			Help:        "Total number of usage limiter decisions.",
			ConstLabels: constLabels,
		},
		[]string{"source", "outcome"},
	)

	// UsagelimiterReplayOperations counts replay operations to the database by status
	// This counter helps monitor the replay queue health and success rate.
	//
	// Example usage:
	//   metrics.UsagelimiterReplayOperations.WithLabelValues("success").Inc()
	//   metrics.UsagelimiterReplayOperations.WithLabelValues("error").Inc()
	UsagelimiterReplayOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "usagelimiter",
			Name:        "replay_operations_total",
			Help:        "Total number of credit replay operations to database.",
			ConstLabels: constLabels,
		},
		[]string{"status"},
	)

	// UsagelimiterReplayLatency measures the latency of replay operations to the database
	// This histogram helps track the performance of async database updates.
	//
	// Example usage:
	//   defer func(start time.Time) {
	//       metrics.UsagelimiterReplayLatency.Observe(time.Since(start).Seconds())
	//   }(time.Now())
	UsagelimiterReplayLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace:   "unkey",
			Subsystem:   "usagelimiter",
			Name:        "replay_latency_seconds",
			Help:        "Histogram of replay operation latencies in seconds.",
			Buckets:     latencyBuckets,
			ConstLabels: constLabels,
		},
	)
)
```

```go
var (
	// PanicsTotal tracks panics recovered by HTTP handler middleware.
	// Use this counter to monitor application stability and identify handlers
	// that are prone to panicking.
	//
	// Labels:
	//   - "caller": The function or handler that panicked
	//   - "path": The HTTP request path that triggered the panic
	//
	// Example usage:
	//   metrics.PanicsTotal.WithLabelValues("handleVerifyKey", "/v1/keys.verifyKey").Inc()
	PanicsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "unkey",
		Subsystem:   "internal",
		Name:        "panics_total",
		Help:        "Total number of panics recovered in HTTP handlers.",
		ConstLabels: constLabels,
	}, []string{"caller", "path"})
)
```

Standard histogram buckets for batch size metrics
```go
var batchSizeBuckets = []float64{
	1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000,
}
```

Standard histogram buckets for request body size metrics in bytes
```go
var bodySizeBuckets = []float64{
	100,
	1024,
	4096,
	16384,
	65536,
	262144,
	1048576,
	4194304,
	16777216,
	67108864,
}
```

We're using const labels as workaround for the prometheus->otel adapter The adapter does not seem to export the resource lavels correctly and because it's temporary, we take the pragmatic approach here.

Remove these after we've moved to pull based prometheus metrics.
```go
var constLabels = prometheus.Labels{
	"region":  os.Getenv("UNKEY_REGION"),
	"version": version.Version,
}
```

Standard histogram buckets for latency metrics in seconds
```go
var latencyBuckets = []float64{
	0.001,
	0.002,
	0.005,
	0.01,
	0.02,
	0.05,
	0.1,
	0.2,
	0.3,
	0.4,
	0.5,
	0.75,
	1.0,
	2.0,
	3.0,
	5.0,
	10.0,
}
```

