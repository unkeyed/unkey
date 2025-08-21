package db

import (
	"time"

	"github.com/unkeyed/unkey/go/pkg/retry"
)

const (
	// DefaultBackoff is the base duration for exponential backoff in database retries
	DefaultBackoff = 50 * time.Millisecond
)

// WithRetry executes a database operation with optimized retry configuration.
// It retries transient errors with exponential backoff but skips non-retryable errors
// like "not found" or "duplicate key" to avoid unnecessary delays.
//
// Configuration:
//   - 3 attempts maximum
//   - Exponential backoff: 50ms, 100ms, 200ms
//   - Skips retries for "not found" and "duplicate key" errors
//
// Usage:
//
//	result, err := db.WithRetry(func() (SomeType, error) {
//		return db.Query.SomeOperation(ctx, db.RO(), params)
//	})
func WithRetry[T any](fn func() (T, error)) (T, error) {
	retrier := retry.New(
		retry.Attempts(3),
		retry.Backoff(func(n int) time.Duration {
			// Predefined backoff delays: 50ms, 100ms, 200ms
			delays := []time.Duration{
				DefaultBackoff,     // 50ms for attempt 1
				DefaultBackoff * 2, // 100ms for attempt 2  
				DefaultBackoff * 4, // 200ms for attempt 3
			}
			if n <= 0 || n > len(delays) {
				return DefaultBackoff // fallback to base delay
			}
			return delays[n-1]
		}),
		retry.ShouldRetry(func(err error) bool {
			// Don't retry if resource is not found - this is a valid response
			if IsNotFound(err) {
				return false
			}

			// Don't retry duplicate key errors - these won't succeed on retry
			if IsDuplicateKeyError(err) {
				return false
			}

			// Retry all other errors (network issues, timeouts, deadlocks, etc.)
			return true
		}),
	)

	var result T
	err := retrier.Do(func() error {
		var retryErr error
		result, retryErr = fn()
		return retryErr
	})

	return result, err
}
