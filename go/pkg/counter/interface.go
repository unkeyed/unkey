// Package counter provides abstractions for distributed counters.
// It defines interfaces and implementations for tracking and incrementing
// counters in a distributed environment.
package counter

import (
	"context"
	"time"
)

// Counter defines the interface for a distributed counter.
// It provides operations to increment and retrieve counter values
// in a thread-safe and distributed manner.
//
// Implementations of this interface are expected to handle:
// - Thread safety for concurrent operations
// - Persistence of counter values
// - Distribution across nodes if applicable
//
// Concurrency: All methods are safe for concurrent use.
type Counter interface {
	// Increment increases the counter by the given value and returns the new count.
	//
	// Parameters:
	//   - ctx: Context for cancellation and tracing
	//   - key: Unique identifier for the counter
	//   - value: Amount to increment the counter by
	//   - ttl: Optional time-to-live duration for the counter. If provided and
	//          the counter is newly created, implementations should set this TTL.
	//          If nil, no TTL is set.
	//
	// Returns:
	//   - int64: The new counter value after incrementing
	//   - error: Any errors that occurred during the operation
	Increment(ctx context.Context, key string, value int64, ttl ...time.Duration) (int64, error)

	// Get retrieves the current value of a counter.
	//
	// Parameters:
	//   - ctx: Context for cancellation and tracing
	//   - key: Unique identifier for the counter
	//
	// Returns:
	//   - int64: The current counter value
	//   - error: Any errors that occurred during the operation
	//           If the counter doesn't exist, implementations should
	//           return 0 and nil error, not an error.
	Get(ctx context.Context, key string) (int64, error)

	// MultiGet retrieves the values of multiple counters in a single operation.
	//
	// Parameters:
	//   - ctx: Context for cancellation and tracing
	//   - keys: Slice of unique identifiers for the counters
	//
	// Returns:
	//   - map[string]int64: Map of counter keys to their current values
	//   - error: Any errors that occurred during the operation
	//            If a counter doesn't exist, its value in the map will be 0.
	MultiGet(ctx context.Context, keys []string) (map[string]int64, error)

	// Decrement decreases the counter by the given value and returns the new count.
	// This is a convenience method equivalent to Increment(ctx, key, -value, ttl...).
	//
	// Parameters:
	//   - ctx: Context for cancellation and tracing
	//   - key: Unique identifier for the counter
	//   - value: Amount to decrement the counter by (must be positive)
	//   - ttl: Optional time-to-live duration for the counter
	//
	// Returns:
	//   - int64: The new counter value after decrementing
	//   - error: Any errors that occurred during the operation
	Decrement(ctx context.Context, key string, value int64, ttl ...time.Duration) (int64, error)

	// DecrementIfExists decrements a counter only if it already exists.
	// This is useful for atomic operations where you need to distinguish between
	// "key doesn't exist" and "key exists but result is negative/zero".
	//
	// Parameters:
	//   - ctx: Context for cancellation and tracing
	//   - key: Unique identifier for the counter
	//   - value: Amount to decrement the counter by (must be positive)
	//
	// Returns:
	//   - int64: The new counter value after decrementing (only valid if existed=true)
	//   - bool: Whether the key existed before the operation
	//   - error: Any errors that occurred during the operation
	DecrementIfExists(ctx context.Context, key string, value int64) (int64, bool, error)

	// IncrementIfExists increments a counter only if it already exists.
	// This is useful for atomic operations where you need to distinguish between
	// "key doesn't exist" and "key exists but result is negative/zero".
	//
	// Parameters:
	//   - ctx: Context for cancellation and tracing
	//   - key: Unique identifier for the counter
	//   - value: Amount to increment the counter by
	//
	// Returns:
	//   - int64: The new counter value after incrementing (only valid if existed=true)
	//   - bool: Whether the key existed before the operation
	//   - error: Any errors that occurred during the operation
	IncrementIfExists(ctx context.Context, key string, value int64) (int64, bool, error)

	// Close releases any resources held by the counter implementation.
	// After calling Close(), the counter instance should not be used again.
	//
	// Returns:
	//   - error: Any errors that occurred during shutdown
	Close() error
}

// Middleware defines a function type that wraps a Counter with additional functionality.
// It can be used to add logging, metrics, validation, or other cross-cutting concerns.
//
// Example Usage:
//
//	func LoggingMiddleware(logger Logger) Middleware {
//	    return func(next Counter) Counter {
//	        return &loggingCounter{next: next, logger: logger}
//	    }
//	}
type Middleware func(Counter) Counter
