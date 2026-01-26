// Package metrics provides Prometheus metric collectors for monitoring Unkey services.
//
// All metrics are registered automatically via [promauto] and use the "unkey" namespace.
// Metrics include constant labels for region and version to support multi-region deployments.
//
// # Metric Organization
//
// Metrics are organized by subsystem:
//   - Batch processing: [BatchSizeDistribution], [BatchOperationsTotal], [BatchItemsProcessedTotal]
//   - Buffer management: [BufferState], [BufferSize], [BufferErrorsTotal]
//   - Caching: [CacheReads], [CacheWrites], [CacheSize], [CacheCapacity]
//   - Circuit breaker: [CircuitBreakerRequests], [CircuitBreakerErrorsTotal]
//   - ClickHouse proxy: [ChproxyRequestsTotal], [ChproxyRowsTotal], [ChproxyErrorsTotal]
//   - Database operations: [DatabaseOperationsLatency], [DatabaseOperationsTotal]
//   - HTTP requests: [HTTPRequestLatency], [HTTPRequestTotal], [HTTPRequestBodySize]
//   - Key verification: [KeyVerificationsTotal], [KeyVerificationErrorsTotal]
//   - Krane orchestration: [KraneControlPlaneReconnectsTotal], [KraneReconcileOperationsTotal], [KraneSecretsRequestsTotal]
//   - Rate limiting: [RatelimitDecision], [RatelimitBuckets], [RatelimitWindows]
//   - Usage limiting: [UsagelimiterDecisions], [UsagelimiterReplayOperations]
//   - Internal: [PanicsTotal]
//
// # Usage
//
// Import the package and use the metric collectors directly:
//
//	import "github.com/unkeyed/unkey/pkg/prometheus/metrics"
//
//	// Increment a counter
//	metrics.HTTPRequestTotal.WithLabelValues("GET", "/v1/keys", "200").Inc()
//
//	// Observe a latency
//	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
//	    metrics.HTTPRequestLatency.WithLabelValues("GET", "/v1/keys", "200").Observe(v)
//	}))
//	defer timer.ObserveDuration()
//
//	// Set a gauge
//	metrics.CacheSize.WithLabelValues("api_keys").Set(float64(cacheSize))
//
// # Label Conventions
//
// Common label patterns used across metrics:
//   - "status": Operation outcome, typically "success" or "error"
//   - "resource": Resource type being operated on (e.g., "user_profile", "api_key")
//   - "replica": Database replica type, "rw" for primary, "ro" for read-only
//   - "method", "path", "status": HTTP request attributes
package metrics
