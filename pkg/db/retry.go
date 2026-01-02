package db

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/retry"
)

const (
	// DefaultBackoff is the base duration for exponential backoff in database retries
	DefaultBackoff = 50 * time.Millisecond
	// DefaultAttempts is the maximum number of retry attempts for database operations
	DefaultAttempts = 3
)

// WithRetryContext executes a database operation with optimized retry configuration while respecting context cancellation and deadlines.
// It retries transient errors with exponential backoff but skips non-retryable errors
// like "not found" or "duplicate key" to avoid unnecessary delays.
//
// Context behavior:
//   - Returns immediately if context is already cancelled or deadline exceeded
//   - Detects context cancellation during backoff sleep without waiting for full duration
//   - Returns context.Canceled or context.DeadlineExceeded on context errors
//
// Configuration:
//   - 3 attempts maximum
//   - Exponential backoff: 50ms, 100ms, 200ms
//   - Skips retries for "not found" and "duplicate key" errors
//
// Usage:
//
//	result, err := db.WithRetryContext(ctx, func() (SomeType, error) {
//		return db.Query.SomeOperation(ctx, db.RO(), params)
//	})
func WithRetryContext[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	return retry.DoWithResultContext(
		retry.New(
			retry.Attempts(DefaultAttempts),
			retry.Backoff(backoffStrategy),
			retry.ShouldRetry(shouldRetryError),
		),
		ctx,
		fn,
	)
}

// backoffStrategy defines exponential backoff delays: 50ms, 100ms, 200ms
func backoffStrategy(n int) time.Duration {
	delays := []time.Duration{
		DefaultBackoff,     // 50ms for attempt 1
		DefaultBackoff * 2, // 100ms for attempt 2
		DefaultBackoff * 4, // 200ms for attempt 3
	}
	if n <= 0 || n > len(delays) {
		return DefaultBackoff
	}
	return delays[n-1]
}

// shouldRetryError determines if a database error should trigger a retry.
// Returns false for "not found" and "duplicate key" errors as these won't succeed on retry.
func shouldRetryError(err error) bool {
	if IsNotFound(err) {
		return false
	}
	if IsDuplicateKeyError(err) {
		return false
	}
	return true
}
