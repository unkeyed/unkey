---
title: counter
description: "provides abstractions and implementations for distributed counters"
---

Package counter provides abstractions and implementations for distributed counters.

This package contains interfaces and concrete implementations for tracking and incrementing counter values in distributed environments. It can be used for various purposes such as rate limiting, usage tracking, and statistics.

Architecture:

  - Uses a simple interface that can be implemented with various backends
  - Provides a Redis implementation for distributed scenarios
  - Supports middleware pattern for extending functionality

Thread Safety:

  - All implementations are safe for concurrent use
  - Operations are atomic across distributed systems (depending on implementation)

Example Usage:

	import "github.com/unkeyed/unkey/pkg/counter"

	// Create a Redis-backed counter
	redisCounter, err := counter.NewRedis(counter.RedisConfig{
		RedisURL: "redis://localhost:6379",
	})
	if err != nil {
		return err
	}
	defer redisCounter.Close()

	// Increment a counter
	newValue, err := redisCounter.Increment(ctx, "my-counter", 1)
	if err != nil {
		return err
	}

	// Get a counter value
	value, err := redisCounter.Get(ctx, "my-counter")
	if err != nil {
		return err
	}

Package counter provides abstractions for distributed counters. It defines interfaces and implementations for tracking and incrementing counters in a distributed environment.

## Constants

```go
const (
	// decrementIfExistsScript is the Lua script for atomic decrement operations.
	// This script checks if the key exists and if there are sufficient credits
	// before decrementing, avoiding negative values that would need reverting.
	//
	// Decrement Logic:
	// - If key doesn't exist: return {0, 0, 0} (value=0, existed=false, success=false)
	// - If insufficient credits: return {current, 1, 0} (unchanged value, existed=true, success=false)
	// - If sufficient credits: return {new_value, 1, 1} (decremented value, existed=true, success=true)
	//
	// The third return value (success flag) provides unambiguous indication of whether
	// the decrement operation succeeded, eliminating the need to infer success from values.
	decrementIfExistsScript = `
		local key = KEYS[1]
		local decrement = tonumber(ARGV[1])

		-- Check if key exists
		local current = redis.call('GET', key)
		if current == false then
			return {0, 0, 0}  -- {value=0, existed=false, success=false}
		end

		current = tonumber(current)
		-- Check if we have sufficient credits before decrementing
		if current < decrement then
			return {current, 1, 0}  -- {current_unchanged, existed=true, success=false}
		end

		-- Sufficient credits, perform atomic decrement preserving TTL
		local newValue = redis.call('DECRBY', key, decrement)
		return {newValue, 1, 1}  -- {new_decremented_value, existed=true, success=true}
	`
)
```


## Variables

```go
var (
	// decrementIfExistsScriptCached is the cached script for atomic decrement operations
	decrementIfExistsScriptCached = redis.NewScript(decrementIfExistsScript)
)
```


## Functions


## Types

### type Counter

```go
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

	// DecrementIfExists decrements a counter only if it already exists and has sufficient credits.
	// This is useful for atomic operations where you need to distinguish between
	// "key doesn't exist", "key exists but insufficient credits", and "successful decrement".
	//
	// Behavior Guarantees:
	//   - The operation will NEVER decrement a counter below zero
	//   - If the current value is less than the decrement amount, no decrement occurs
	//   - Always returns the actual current counter value (never sentinel values)
	//   - The value parameter must be positive (> 0)
	//
	// Parameters:
	//   - ctx: Context for cancellation and tracing
	//   - key: Unique identifier for the counter
	//   - value: Amount to decrement the counter by (must be positive, > 0)
	//
	// Returns:
	//   - int64: The actual counter value:
	//            * When existed=false: returns 0 (key didn't exist)
	//            * When existed=true and sufficient credits: returns new value after decrement (>= 0)
	//            * When existed=true but insufficient credits: returns current value unchanged (>= 0)
	//   - bool: Whether the key existed before the operation
	//   - bool: Whether the decrement was successful (only valid when existed=true)
	//   - error: Any errors that occurred during the operation
	//
	// Usage Pattern:
	//   remaining, existed, success, err := counter.DecrementIfExists(ctx, key, cost)
	//   if !existed {
	//       // Key doesn't exist, handle accordingly
	//   } else if success {
	//       // Decrement was successful (remaining is the new value)
	//   } else {
	//       // Insufficient credits (remaining is current unchanged value)
	//   }
	//
	// Note: The success flag provides unambiguous indication of operation result,
	// eliminating the need to infer success from the returned value.
	DecrementIfExists(ctx context.Context, key string, value int64) (int64, bool, bool, error)

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
```

Counter defines the interface for a distributed counter. It provides operations to increment and retrieve counter values in a thread-safe and distributed manner.

Implementations of this interface are expected to handle: - Thread safety for concurrent operations - Persistence of counter values - Distribution across nodes if applicable

Concurrency: All methods are safe for concurrent use.

#### func NewRedis

```go
func NewRedis(config RedisConfig) (Counter, error)
```

NewRedis creates a new Redis-backed counter implementation.

Parameters:

  - config: Configuration options for the Redis counter

Returns:

  - Counter: Redis implementation of the Counter interface
  - error: Any errors during initialization

### type Middleware

```go
type Middleware func(Counter) Counter
```

Middleware defines a function type that wraps a Counter with additional functionality. It can be used to add logging, metrics, validation, or other cross-cutting concerns.

Example Usage:

	func LoggingMiddleware(logger Logger) Middleware {
	    return func(next Counter) Counter {
	        return &loggingCounter{next: next}
	    }
	}

### type RedisConfig

```go
type RedisConfig struct {
	// RedisURL is the connection URL for Redis.
	// Format: redis://[[username][:password]@][host][:port][/database]
	RedisURL string
}
```

RedisConfig holds configuration options for the Redis counter.

