package debug

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/go/pkg/zen"
)

// cacheHeadersEnabled controls whether cache debug headers are written to HTTP responses.
// Use EnableCacheHeaders() and DisableCacheHeaders() to modify this value safely.
var cacheHeadersEnabled atomic.Bool

// EnableCacheHeaders enables writing cache debug headers (X-Unkey-Debug-Cache) to HTTP responses.
// When enabled, cache operations will add headers showing cache hit/miss status and latency.
//
// This should typically be called once during application startup based on configuration:
//
//	if config.DebugCacheHeaders {
//	    debug.EnableCacheHeaders()
//	}
//
// Thread-safe and can be called from multiple goroutines.
func EnableCacheHeaders() {
	cacheHeadersEnabled.Store(true)
}

// DisableCacheHeaders disables writing cache debug headers to HTTP responses.
// This is the default state - headers are disabled unless explicitly enabled.
//
// Thread-safe and can be called from multiple goroutines.
func DisableCacheHeaders() {
	cacheHeadersEnabled.Store(false)
}

// CacheHeadersEnabled returns whether cache debug headers are currently enabled.
// This can be used to check the current state without modifying it.
func CacheHeadersEnabled() bool {
	return cacheHeadersEnabled.Load()
}

// RecordCacheHit records a cache operation result and immediately writes it as
// an HTTP response header for debugging and observability purposes.
//
// This function enables real-time cache debugging by adding structured headers
// that show cache hit/miss patterns, operation latencies, and cache naming.
// It works with any HTTP framework by using zen's session-in-context pattern
// to write headers directly without requiring explicit session passing.
//
// The function checks for a zen session in the provided context and writes cache
// debug information directly to the HTTP response headers if a session is available.
// If no session is found (cache debug not enabled), the function returns silently
// without any side effects.
//
// Parameters:
//   - ctx: Request context that should contain a zen session stored via
//     zen.WithSession. If no session is present, the function is a no-op.
//   - cacheName: Identifies the specific cache being accessed. This should be
//     a stable identifier that helps distinguish between different caches in
//     the system (e.g., "ApiByID", "RootKeyByHash", "PermissionsByApiId").
//   - status: Describes the cache operation result. Common values include
//     "FRESH" (cache hit with valid data), "STALE" (cache hit with expired data),
//     "MISS" (cache miss requiring data source access), and "ERROR" (cache
//     operation failed).
//   - latency: Total time spent on the cache operation, including any database
//     roundtrips for cache misses. This should represent the complete operation
//     time from the caller's perspective.
//
// Header format:
// The function writes headers with the name "X-Unkey-Debug-Cache" and values
// in the format "cache_name:latency:status". Multiple cache operations in a
// single request result in multiple headers with the same name, which is
// standard HTTP behavior and allows observing all cache activity.
//
// Example headers:
//
//	X-Unkey-Debug-Cache: ApiByID:1.25ms:FRESH
//	X-Unkey-Debug-Cache: RootKeyByHash:150us:MISS
//	X-Unkey-Debug-Cache: PermissionsByApiId:2.1ms:STALE
//
// Latency formatting:
// Durations are formatted for human readability with appropriate units.
// Values >= 1ms are shown in milliseconds with 2 decimal places, while
// values < 1ms are shown in microseconds rounded to whole numbers.
//
// Usage pattern:
// This function is typically called from within cache operations after
// determining the cache result and measuring the operation time:
//
//	start := time.Now()
//	value, found := cache.Get(key)
//	latency := time.Since(start)
//
//	status := "MISS"
//	if found {
//	    status = "FRESH" // or "STALE" based on expiration
//	}
//
//	debug.RecordCacheHit(ctx, "ApiByID", status, latency)
//
// Performance considerations:
// The function has minimal overhead when cache debug is disabled (simple context
// lookup that returns early). When enabled, it performs a single header write
// operation which is very fast. The formatting operations use efficient string
// building and are suitable for hot paths.
//
// Security note:
// Cache debug headers may expose information about internal system performance
// and caching behavior. This functionality should typically only be enabled
// in development environments or for specific debugging scenarios in production.
func RecordCacheHit(ctx context.Context, cacheName, status string, latency time.Duration) {
	// Fast path: check if cache headers are enabled globally
	if !cacheHeadersEnabled.Load() {
		return
	}

	// Try to get a session from context
	session, ok := zen.SessionFromContext(ctx)
	if !ok {
		// No session in context, can't write headers
		return
	}

	// Create structured cache header and write it
	header := NewCacheHeader(cacheName, status, latency)
	session.AddHeader("X-Unkey-Debug-Cache", header.String())
}

// formatDuration formats a time duration for human-readable display in cache debug headers.
//
// The function automatically selects appropriate time units based on the duration
// magnitude to provide optimal readability. Longer durations are displayed in
// milliseconds with decimal precision, while shorter durations use microseconds
// to maintain precision for fast cache operations.
//
// Formatting rules:
//   - Durations >= 1ms: Displayed as milliseconds with 2 decimal places (e.g., "1.25ms")
//   - Durations < 1ms: Displayed as whole microseconds (e.g., "750us")
//
// This formatting strikes a balance between precision for performance analysis
// and readability in HTTP headers. The approach ensures that both fast cache
// hits (typically < 1ms) and slower cache misses with database roundtrips
// (typically > 1ms) are displayed with appropriate granularity.
//
// Examples:
//
//	formatDuration(1500 * time.Microsecond) // "1.50ms"
//	formatDuration(750 * time.Microsecond)  // "750us"
//	formatDuration(2 * time.Millisecond)    // "2.00ms"
//	formatDuration(100 * time.Nanosecond)   // "0us"
func formatDuration(d time.Duration) string {
	if d >= time.Millisecond {
		return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000)
	}
	return fmt.Sprintf("%.0fus", float64(d.Nanoseconds())/1000)
}
