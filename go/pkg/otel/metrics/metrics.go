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
		Requests Int64Counter
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
		//   metrics.Cache.Hits.RegisterCallback(func(_ context.Context, o metric.Int64Observer) error {
		//     o.Observe(hitCount, metric.WithAttributes(
		//       attribute.String("resource", "user_profile"),
		//     ))
		//     return nil
		//   })
		Hits Int64Observable

		// Misses tracks the number of cache read operations that did not find the requested item.
		// Use this to monitor cache efficiency and identify opportunities for improvement.
		//
		// Attributes:
		//   - resource (string): The type of resource being cached (e.g., "user_profile")
		//
		// Example:
		//   metrics.Cache.Misses.RegisterCallback(func(_ context.Context, o metric.Int64Observer) error {
		//     o.Observe(missCount, metric.WithAttributes(
		//       attribute.String("resource", "user_profile"),
		//     ))
		//     return nil
		//   })
		Misses Int64Observable

		// Writes tracks the number of cache write operations.
		// Use this to monitor write pressure on the cache.
		//
		// Attributes:
		//   - resource (string): The type of resource being cached (e.g., "user_profile")
		//
		// Example:
		//   metrics.Cache.Writes.RegisterCallback(func(_ context.Context, o metric.Int64Observer) error {
		//     o.Observe(writeCount, metric.WithAttributes(
		//       attribute.String("resource", "user_profile"),
		//     ))
		//     return nil
		//   })
		Writes Int64Observable

		// Evicted tracks the number of items removed from the cache due to space constraints
		// or explicit deletion. Use this to monitor cache churn and capacity issues.
		//
		// Attributes:
		//   - resource (string): The type of resource being cached (e.g., "user_profile")
		//   - reason (string): The reason for eviction (e.g., "capacity", "ttl", "manual")
		//
		// Example:
		//   metrics.Cache.Evicted.RegisterCallback(func(_ context.Context, o metric.Int64Observer) error {
		//     o.Observe(evictionCount, metric.WithAttributes(
		//       attribute.String("resource", "user_profile"),
		//       attribute.String("reason", "ttl"),
		//     ))
		//     return nil
		//   })
		Evicted Int64Observable

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
		//   metrics.Cache.Size.RegisterCallback(func(_ context.Context, o metric.Int64Observer) error {
		//     o.Observe(currentSize, metric.WithAttributes(
		//       attribute.String("resource", "user_profile"),
		//     ))
		//     return nil
		//   })
		Size Int64Observable

		// Capacity tracks the current maximum number of items in the cache
		// Use this to monitor cache utilization and growth patterns.
		//
		// Attributes:
		//   - resource (string): The type of resource being cached (e.g., "user_profile")
		//
		// Example:
		//   metrics.Cache.Capacity.RegisterCallback(func(_ context.Context, o metric.Int64Observer) error {
		//     o.Observe(currentCapacity, metric.WithAttributes(
		//       attribute.String("resource", "user_profile"),
		//     ))
		//     return nil
		//   })
		Capacity Int64Observable
	}

	// Cluster contains metrics related to cluster operations and status
	Cluster struct {
		// Size tracks the current number of nodes in the cluster.
		// Use this to monitor cluster health, scaling events, and load distribution.
		//
		// Attributes:
		//   - nodeID (string): The unique identifier of the node (e.g., "node-1", "node-abc123")
		//
		// Example:
		//   metrics.Cluster.Size.RegisterCallback(func(_ context.Context, o metric.Int64Observer) error {
		//     o.Observe(nodeCount, metric.WithAttributes(
		//       attribute.String("nodeID", "node-abc123"),
		//     ))
		//     return nil
		//   })
		Size Int64Observable
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
	Cache.Hits = &int64ObservableCounter{
		m:    m,
		name: "cache_hit",
		opts: []metric.Int64ObservableCounterOption{
			metric.WithDescription("How many cache hits we encountered."),
		},
	}

	Cache.Misses = &int64ObservableCounter{
		m:    m,
		name: "cache_miss",
		opts: []metric.Int64ObservableCounterOption{
			metric.WithDescription("How many cache misses we encountered."),
		},
	}

	Cache.Writes = &int64ObservableCounter{
		m:    m,
		name: "cache_writes",
		opts: []metric.Int64ObservableCounterOption{
			metric.WithDescription("How many cache writes we did."),
		},
	}

	Cache.Evicted = &int64ObservableCounter{
		m:    m,
		name: "cache_evicted",
		opts: []metric.Int64ObservableCounterOption{
			metric.WithDescription("How many cache evictions we did."),
		},
	}

	Cache.ReadLatency, err = m.Int64Histogram("cache_read_latency",
		metric.WithDescription("The latency of read operations"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}

	Cache.Size = &int64ObservableGauge{
		m:    m,
		name: "cache_size",
		opts: []metric.Int64ObservableGaugeOption{
			metric.WithDescription("How many entries are stored in the cache."),
		},
	}
	Cache.Capacity = &int64ObservableGauge{
		m:    m,
		name: "cache_capacity",
		opts: []metric.Int64ObservableGaugeOption{
			metric.WithDescription("Maximum number of items the cache can hold."),
		},
	}

	Cluster.Size = &int64ObservableGauge{
		m:    m,
		name: "cluster_size",
		opts: []metric.Int64ObservableGaugeOption{
			metric.WithDescription("How many nodes are in the cluster."),
		},
	}
	return nil
}
