package db

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/retry"
	"github.com/unkeyed/unkey/pkg/wide"
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
	attempt := 0
	return retry.DoWithResultContext(
		retry.New(
			retry.Attempts(DefaultAttempts),
			retry.Backoff(backoffStrategy),
			retry.ShouldRetry(func(err error) bool {
				shouldRetry := shouldRetryError(err)
				if shouldRetry {
					// Track retry reason for debugging
					wide.Set(ctx, wide.FieldDBRetryReason, ClassifyDBError(err))
				}

				return shouldRetry
			}),
		),
		ctx,
		func() (T, error) {
			attempt++
			if attempt > 1 {
				wide.Set(ctx, wide.FieldDBRetryAttempt, attempt)
			}
			return fn()
		},
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
//
// Returns true for transient errors that may succeed on retry:
//   - Deadlocks (MySQL error 1213)
//   - Lock wait timeouts (MySQL error 1205)
//   - Connection errors (MySQL errors 2006, 2013, network errors)
//   - Too many connections (MySQL error 1040)
//
// Returns false for permanent errors that won't succeed on retry:
//   - Not found errors
//   - Duplicate key errors (MySQL error 1062)
//
// See: https://dev.mysql.com/doc/refman/8.0/en/innodb-error-handling.html
func shouldRetryError(err error) bool {
	// Not found and duplicate key errors are permanent - don't retry
	if IsNotFound(err) || IsDuplicateKeyError(err) {
		return false
	}

	// Only retry known transient errors
	return IsTransientError(err)
}

// TxWithResultRetry executes a transaction with automatic retry on transient errors.
// It wraps TxWithResult with retry logic, retrying the entire transaction (begin -> fn -> commit)
// on retryable errors.
//
// This is useful for transactions that may encounter transient errors due to concurrent access
// patterns or temporary resource constraints. When such errors occur, MySQL rolls back the
// transaction, so we retry from the beginning.
//
// Configuration:
//   - 3 attempts maximum
//   - Exponential backoff: 50ms, 100ms, 200ms
//
// Retries on transient errors:
//   - Deadlocks (MySQL error 1213)
//   - Lock wait timeouts (MySQL error 1205)
//   - Connection errors (MySQL errors 2006, 2013, network errors)
//   - Too many connections (MySQL error 1040)
//
// Does NOT retry on permanent errors:
//   - Not found errors
//   - Duplicate key errors (MySQL error 1062)
//
// Usage:
//
//	result, err := db.TxWithResultRetry(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) (*Result, error) {
//		// Perform transactional operations
//		return &Result{}, nil
//	})
func TxWithResultRetry[T any](ctx context.Context, db *Replica, fn func(context.Context, DBTX) (T, error)) (T, error) {
	attempt := 0
	return retry.DoWithResultContext(
		retry.New(
			retry.Attempts(DefaultAttempts),
			retry.Backoff(backoffStrategy),
			retry.ShouldRetry(func(err error) bool {
				shouldRetry := shouldRetryError(err)
				if shouldRetry {
					wide.Set(ctx, wide.FieldDBRetryReason, ClassifyDBError(err))
				}
				return shouldRetry
			}),
		),
		ctx,
		func() (T, error) {
			attempt++
			if attempt > 1 {
				wide.Set(ctx, wide.FieldDBRetryAttempt, attempt)
			}
			return TxWithResult(ctx, db, fn)
		},
	)
}

// TxRetry executes a transaction with automatic retry on transient errors like deadlocks.
// It is a convenience wrapper around TxWithResultRetry for operations that don't return a value.
//
// Usage:
//
//	err := db.TxRetry(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) error {
//		// Perform transactional operations
//		return nil
//	})
func TxRetry(ctx context.Context, db *Replica, fn func(context.Context, DBTX) error) error {
	_, err := TxWithResultRetry(ctx, db, func(ctx context.Context, tx DBTX) (any, error) {
		return nil, fn(ctx, tx)
	})

	return err
}
