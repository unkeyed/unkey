package mysql

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
func shouldRetryError(err error) bool {
	// Not found and duplicate key errors are permanent - don't retry
	if IsNotFound(err) || IsDuplicateKeyError(err) {
		return false
	}

	// Only retry known transient errors
	return IsTransientError(err)
}

// TxWithResultRetry executes a transaction with automatic retry on transient errors.
func TxWithResultRetry[T any](ctx context.Context, db *Replica, fn func(context.Context, DBTX) (T, error)) (T, error) {
	return retry.DoWithResultContext(
		retry.New(
			retry.Attempts(DefaultAttempts),
			retry.Backoff(backoffStrategy),
			retry.ShouldRetry(shouldRetryError),
		),
		ctx,
		func() (T, error) {
			return TxWithResult(ctx, db, fn)
		},
	)
}

// TxRetry executes a transaction with automatic retry on transient errors like deadlocks.
// It is a convenience wrapper around TxWithResultRetry for operations that don't return a value.
func TxRetry(ctx context.Context, db *Replica, fn func(context.Context, DBTX) error) error {
	_, err := TxWithResultRetry(ctx, db, func(ctx context.Context, tx DBTX) (any, error) {
		return nil, fn(ctx, tx)
	})

	return err
}
