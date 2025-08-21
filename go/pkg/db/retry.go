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
			// Exponential backoff: 50ms, 100ms, 200ms
			return time.Duration(1<<uint(n-1)) * DefaultBackoff
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
