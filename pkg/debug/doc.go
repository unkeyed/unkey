// Package debug provides debugging and observability utilities for the Unkey system.
//
// This package contains tools for real-time debugging and performance monitoring
// of system components, with particular focus on cache behavior analysis. The
// package is designed to provide detailed insights into system operations without
// impacting performance when debugging features are disabled.
//
// The debug package integrates closely with the zen HTTP framework to provide
// request-scoped debugging information that can be exposed through HTTP headers
// or logged for analysis.
//
// # Cache Debug Headers
//
// The primary functionality is cache operation debugging through HTTP headers.
// Cache operations throughout the system can record their results, latencies,
// and status information which gets written to HTTP response headers for
// real-time observability.
//
// The cache debug system uses zen's session-in-context pattern to enable cache
// operations to write debug headers without requiring explicit session passing.
// This solves timing issues where traditional middleware approaches fail due to
// HTTP header write deadlines.
//
// Usage example:
//
//	// In a cache operation:
//	start := time.Now()
//	value, found := cache.Get(ctx, key)
//	latency := time.Since(start)
//
//	status := "MISS"
//	if found {
//	    status = "FRESH"
//	}
//
//	debug.RecordCacheHit(ctx, "ApiByID", status, latency)
//
//	// Results in HTTP header:
//	// X-Unkey-Debug-Cache: ApiByID:1.25ms:FRESH
//
// # Header Format
//
// Cache debug headers use the name "X-Unkey-Debug-Cache" with values in the
// format "cache_name:latency:status". Multiple cache operations in a single
// request result in multiple headers with the same name, following standard
// HTTP conventions.
//
// Common status values include:
//   - FRESH: Cache hit with valid, unexpired data
//   - STALE: Cache hit with expired data that may still be usable
//   - MISS: Cache miss requiring data source access
//   - ERROR: Cache operation failed due to system issues
//
// Latency values are automatically formatted for readability, showing milliseconds
// for durations >= 1ms and microseconds for shorter durations.
//
// # Performance Characteristics
//
// The debug package is designed for minimal performance impact:
//   - Zero overhead when debugging is disabled (context lookup returns early)
//   - Minimal overhead when enabled (single header write per cache operation)
//   - Efficient duration formatting suitable for hot paths
//   - No memory allocations in the common disabled case
//
// # Security Considerations
//
// Cache debug headers may expose information about internal system performance,
// cache hit rates, and operation timing. This functionality should typically
// only be enabled in development environments or for specific debugging scenarios
// in production with appropriate access controls.
//
// # Integration with Zen Framework
//
// The debug package relies on zen's context management for session access.
// Cache debug functionality is automatically enabled when the zen framework
// stores a session in the request context, and disabled otherwise.
//
// The package uses [zen.SessionFromContext] to retrieve the HTTP session and
// write headers directly, bypassing traditional middleware timing constraints
// that can prevent headers from being written after response bodies are sent.
package debug
