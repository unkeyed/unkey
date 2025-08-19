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
	// This is implemented as a call to Increment with a negative value and will create
	// the counter if it does not already exist (mirroring Increment behavior).
	//
	// Parameters:
	//   - ctx: Context for cancellation and tracing
	//   - key: Unique identifier for the counter
	//   - value: Amount to decrement the counter by (must be positive)
	//   - ttl: Optional time-to-live duration for the counter. Only the first TTL
	//           argument, if provided, is honored when creating/refreshing the counter.
	//
	// Returns:
	//   - int64: The new counter value after decrementing
	//   - error: Any errors that occurred during the operation
	Decrement(ctx context.Context, key string, value int64, ttl ...time.Duration) (int64, error)

	// DecrementIfExists decrements a counter only if it already exists.
	// This is useful for atomic operations where you need to distinguish between
	// "key doesn't exist" and "key exists".
	//
	// Behavior Guarantees:
	//   - The operation will NEVER decrement a counter below zero
	//   - If the current value is less than the decrement amount, no decrement occurs
	//   - When decrement is refused due to insufficient credits, existed=true and newValue=-1
	//   - The value parameter must be positive (> 0)
	//
	// Parameters:
	//   - ctx: Context for cancellation and tracing
	//   - key: Unique identifier for the counter
	//   - value: Amount to decrement the counter by (must be positive, > 0)
	//
	// Returns:
	//   - int64: The counter value after attempting decrement. Special sentinel values:
	//            * When existed=false: returns 0 (key didn't exist)
	//            * When existed=true but insufficient credits: returns -1 (insufficient credits)
	//            * When existed=true and sufficient credits: returns new value after decrement (>= 0)
	//   - bool: Whether the key existed before the operation
	//   - error: Any errors that occurred during the operation
	//
	// Sentinel Value Semantics:
	//   - newValue == 0 && existed == false: Key doesn't exist, no operation performed
	//   - newValue == -1 && existed == true: Key exists but insufficient credits, no decrement performed
	//   - newValue >= 0 && existed == true: Successful decrement, newValue is the non-negative result
	//
	// Implementers: Treat newValue == -1 as "insufficient credits but key existed" sentinel value.
	// Callers: Check for -1 return value to unambiguously detect insufficient credit scenarios.
	DecrementIfExists(ctx context.Context, key string, value int64) (int64, bool, error)

	// SetIfNotExists sets a counter to a specific value only if it doesn't already exist.
	// This is useful for atomic initialization without race conditions.
	//
	// Parameters:
	//   - ctx: Context for cancellation and tracing
	//   - key: Unique identifier for the counter
	//   - value: Value to set the counter to
	//   - ttl: Optional time-to-live duration for the counter
	//
	// Returns:
	//   - bool: Whether the key was set (true) or already existed (false)
	//   - error: Any errors that occurred during the operation
	SetIfNotExists(ctx context.Context, key string, value int64, ttl ...time.Duration) (bool, error)

	// Delete removes a counter key from the store.
	// This is useful for forcing reinitialization or cleaning up stale data.
	//
	// Parameters:
	//   - ctx: Context for cancellation and tracing
	//   - key: Unique identifier for the counter to delete
	//
	// Returns:
	//   - error: Any errors that occurred during the operation
	Delete(ctx context.Context, key string) error

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
