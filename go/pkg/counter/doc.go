/*
Package counter provides abstractions and implementations for distributed counters.

This package contains interfaces and concrete implementations for tracking and
incrementing counter values in distributed environments. It can be used for
various purposes such as rate limiting, usage tracking, and statistics.

Architecture:
  - Uses a simple interface that can be implemented with various backends
  - Provides a Redis implementation for distributed scenarios
  - Supports middleware pattern for extending functionality

Thread Safety:
  - All implementations are safe for concurrent use
  - Operations are atomic across distributed systems (depending on implementation)

Example Usage:

	import "github.com/unkeyed/unkey/go/pkg/counter"

	// Create a Redis-backed counter
	redisCounter, err := counter.NewRedis(counter.RedisConfig{
		RedisURL: "redis://localhost:6379",
		Logger:   logger,
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
*/
package counter
