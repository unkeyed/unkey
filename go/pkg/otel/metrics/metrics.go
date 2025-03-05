// unkey/go/pkg/otel/metrics/metrics.go
// Package metrics provides OpenTelemetry instrumentation for monitoring application behavior.
// It exposes global metric instances, initialized with no-op
// implementations by default that can be replaced with real implementations via Init().
package metrics

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

// init initializes the metrics package with a no-op meter provider.
// This ensures the package can be safely imported even if not explicitly initialized.
// For production use, call Init() with a proper meter provider.
func init() {
	p := noop.NewMeterProvider()
	m := p.Meter("noop")

	err := Init(m)
	if err != nil {
		panic(err)
	}
}

// Global metric instances provide easy access to commonly used metrics.
// These are initialized during package initialization and are safe for concurrent use.
var (
	// Http contains metrics related to HTTP operations
	Http struct {
		// Requests counts all incoming API requests.
		// Use this counter to monitor API traffic patterns.
		//
		// Attributes:
		//   - path (string): The HTTP path of the request (e.g., "/api/v1/users")
		//   - status (int): The HTTP status code of the response (e.g., 200, 404, 500)
		//
		// Example:
		//   metrics.Http.Requests.Add(ctx, 1, metric.WithAttributes(
		//     attribute.String("path", "/api/v1/users"),
		//     attribute.Int("status", 200),
		//   ))
		Requests metric.Int64Counter
	}

	// Cache contains metrics related to cache operations
	Cache struct {
		// Hits tracks the number of cache read operations that found the requested item.
		// Use this to monitor cache hit rates and effectiveness.
		//
		// Attributes:
		//   - resource (string): The type of resource being cached (e.g., "user_profile")
		//
		// Example:
		//   metrics.Cache.Hits.Record(ctx, 1, metric.WithAttributes(
		//     attribute.String("resource", "user_profile"),
		//   ))
		Hits metric.Int64Gauge

		// Misses tracks the number of cache read operations that did not find the requested item.
		// Use this to monitor cache efficiency and identify opportunities for improvement.
		//
		// Attributes:
		//   - resource (string): The type of resource being cached (e.g., "user_profile")
		//
		// Example:
		//   metrics.Cache.Misses.Record(ctx, 1, metric.WithAttributes(
		//     attribute.String("resource", "user_profile"),
		//   ))
		Misses metric.Int64Gauge

		// Writes tracks the number of cache write operations.
		// Use this to monitor write pressure on the cache.
		//
		// Attributes:
		//   - resource (string): The type of resource being cached (e.g., "user_profile")
		//
		// Example:
		//   metrics.Cache.Writes.Add(ctx, 1, metric.WithAttributes(
		//     attribute.String("resource", "user_profile"),
		//   ))
		Writes metric.Int64Counter

		// Evicted tracks the number of items removed from the cache due to space constraints
		// or explicit deletion. Use this to monitor cache churn and capacity issues.
		//
		// Attributes:
		//   - resource (string): The type of resource being cached (e.g., "user_profile")
		//   - reason (string): The reason for eviction (e.g., "capacity", "ttl", "manual")
		//
		// Example:
		//   metrics.Cache.Evicted.Record(ctx, 1, metric.WithAttributes(
		//     attribute.String("resource", "user_profile"),
		//     attribute.String("reason", "ttl"),
		//   ))
		Evicted metric.Int64Gauge

		// ReadLatency measures the duration of cache read operations in milliseconds.
		// This histogram helps track cache performance and identify slowdowns.
		//
		// Attributes:
		//   - resource (string): The type of resource being read (e.g., "user_profile")
		//   - result (string): The outcome of the read ("hit" or "miss")
		//
		// Example:
		//   metrics.Cache.ReadLatency.Record(ctx, 42, metric.WithAttributes(
		//     attribute.String("resource", "user_profile"),
		//     attribute.String("result", "hit"),
		//   ))
		ReadLatency metric.Int64Histogram

		// Size tracks the current number of items in the cache.
		// Use this to monitor cache utilization and growth patterns.
		//
		// Attributes:
		//   - resource (string): The type of resource being cached (e.g., "user_profile")
		//
		// Example:
		//   metrics.Cache.Size.Record(ctx, 1042, metric.WithAttributes(
		//     attribute.String("resource", "user_profile"),
		//   ))
		Size metric.Int64Gauge
	}
)

// Init initializes the metrics with the provided meter.
// This function must be called before using any metrics in production, typically
// during application startup. It replaces the default no-op implementations with
// real metrics that will be reported to your metrics backend.
//
// Parameters:
//   - m: The meter instance to use for creating metrics
//
// Returns:
//   - error: Any error encountered during metric initialization
//
// Example:
//
//	provider := prometheus.NewMeterProvider()
//	meter := provider.Meter("my-service")
//	if err := metrics.Init(meter); err != nil {
//	    log.Fatal("failed to initialize metrics:", err)
//	}
func Init(m metric.Meter) error {
	var err error

	// Initialize HTTP metrics
	Http.Requests, err = m.Int64Counter("http_request",
		metric.WithDescription("How many api requests we handle."),
	)
	if err != nil {
		return err
	}

	// Initialize Cache metrics
	Cache.Hits, err = m.Int64Gauge("cache_hit",
		metric.WithDescription("Cache hits"),
	)
	if err != nil {
		return err
	}

	Cache.Misses, err = m.Int64Gauge("cache_miss",
		metric.WithDescription("Cache misses"),
	)
	if err != nil {
		return err
	}

	Cache.Writes, err = m.Int64Counter("cache_write",
		metric.WithDescription("Cache writes"),
	)
	if err != nil {
		return err
	}

	Cache.Evicted, err = m.Int64Gauge("cache_evicted",
		metric.WithDescription("Evicted entries"),
	)
	if err != nil {
		return err
	}

	Cache.ReadLatency, err = m.Int64Histogram("cache_read_latency",
		metric.WithDescription("The latency of read operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}

	Cache.Size, err = m.Int64Gauge("cache_size",
		metric.WithDescription("How many entries are stored in the cache."),
	)
	if err != nil {
		return err
	}

	return nil
}
