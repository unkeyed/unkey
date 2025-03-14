// Package metrics provides OpenTelemetry metric instrumentation for the Unkey system.
//
// This package offers a set of pre-defined metrics.
//
// It's designed to be initialized once at application startup and then used throughout
// the codebase for consistent metric collection.
//
// By default, the package initializes with no-op metrics that don't record any data,
// ensuring that importing the package is safe even without explicit initialization.
// For production use, call Init() with a proper meter provider.
//
// Example usage:
//
//	// Initialize metrics with a real provider
//	provider := prometheus.NewMeterProvider()
//	meter := provider.Meter("my-service")
//	if err := metrics.Init(meter); err != nil {
//	    log.Fatal("failed to initialize metrics:", err)
//	}
//
//	// Record HTTP request metrics
//	metrics.Http.Requests.Add(ctx, 1, metric.WithAttributes(
//	    attribute.String("path", "/api/v1/keys"),
//	    attribute.Int("status", 200),
//	))
//
//	// Register a callback for observable metrics
//	metrics.Cache.Size.RegisterCallback(func(_ context.Context, o metric.Int64Observer) error {
//	    o.Observe(currentSize, metric.WithAttributes(
//	        attribute.String("resource", "api_keys"),
//	    ))
//	    return nil
//	})
package metrics
