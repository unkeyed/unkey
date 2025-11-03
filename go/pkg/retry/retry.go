// Package retry provides functionality to retry operations with configurable attempts and backoff.
package retry

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// retry holds the configuration for retry attempts and backoff strategy.
type retry struct {
	// attempts is the maximum number of times to try the operation
	attempts int

	// backoff is a function that returns the duration to wait before the next retry
	// based on the current attempt number (starting at 1)
	backoff func(n int) time.Duration

	// shouldRetry is a function that determines if an error is retryable
	// If nil, all errors are considered retryable
	shouldRetry func(error) bool

	// used for testing
	// overwrite time.Sleep to speed up tests
	sleep func(d time.Duration)
}

// New creates a new retry instance with default configuration.
// Default configuration:
//   - 3 retry attempts
//   - Linear backoff starting at 100ms, increasing by 100ms per attempt
//
// Example:
//
//	r := retry.New()
//	err := r.Do(func() error {
//		// Simulate an operation that might fail
//		resp, err := http.Get("https://api.example.com")
//		if err != nil {
//			return err
//		}
//		defer resp.Body.Close()
//
//		if resp.StatusCode != http.StatusOK {
//			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
//		}
//		return nil
//	})
//
//	if err != nil {
//		log.Printf("operation failed after 3 attempts: %v", err)
//	}
//
// The retry behavior can be customized using Attempts() and Backoff():
//
// r := retry.New(
//
//	retry.Attempts(5),
//	retry.Backoff(func(n int) time.Duration {
//		return time.Duration(1<<uint(n)) * time.Second // exponential backoff
//	}),
//
// )
func New(applies ...Apply) *retry {
	r := &retry{
		attempts:    3,
		backoff:     func(n int) time.Duration { return time.Duration(n) * 100 * time.Millisecond },
		shouldRetry: nil, // nil means all errors are retryable
		sleep:       time.Sleep,
	}
	for _, a := range applies {
		r = a(r)
	}
	return r
}

// Apply modifies r and returns it
type Apply func(r *retry) *retry

// Attempts sets the maximum number of retry attempts.
// The operation will be attempted up to this many times before giving up.
func Attempts(attempts int) Apply {
	return func(r *retry) *retry {
		r.attempts = attempts
		return r
	}
}

// Backoff sets the backoff strategy function.
// The function receives the current attempt number (starting with 1) and
// should return the duration to wait before the next attempt.
func Backoff(backoff func(n int) time.Duration) Apply {
	return func(r *retry) *retry {
		r.backoff = backoff
		return r
	}
}

// ShouldRetry sets a function to determine if an error should trigger a retry.
// If not set or set to nil, all errors will trigger retries.
// This is useful for skipping retries on non-transient errors like "not found".
//
// Example:
//
//	r := retry.New(
//		retry.Attempts(3),
//		retry.ShouldRetry(func(err error) bool {
//			// Don't retry if the error is a "not found" error
//			if errors.Is(err, ErrNotFound) {
//				return false
//			}
//			// Retry all other errors
//			return true
//		}),
//	)
func ShouldRetry(shouldRetry func(error) bool) Apply {
	return func(r *retry) *retry {
		r.shouldRetry = shouldRetry
		return r
	}
}

// Do executes the given function with configured retry behavior.
// The function is retried until it succeeds or the maximum number of attempts is reached.
// Between attempts, the backoff function determines the wait duration.
// If shouldRetry is configured, it will be called to determine if a retry should occur.
//
// Returns nil if the operation succeeds, or the last error encountered if all retries fail
// or if the error is non-retryable according to shouldRetry.
// Returns an error if attempts is configured to less than 1.
func (r *retry) Do(fn func() error) error {
	if r.attempts < 1 {
		return fmt.Errorf("attempts must be greater than 0")
	}

	var err error
	for i := 1; i <= r.attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		// Check if we should retry this error
		if r.shouldRetry != nil && !r.shouldRetry(err) {
			// Error is not retryable, return immediately
			return err
		}

		// Don't sleep after the last attempt
		if i < r.attempts {
			r.sleep(r.backoff(i))
		}
	}

	return err
}

// DoWithResult executes the given function with configured retry behavior and returns a result.
// Works like Do() but for functions that return a value along with an error.
// On failure, returns the result from the last attempt along with the final error.
//
// Example:
//
//	r := retry.New(retry.Attempts(3))
//	user, err := retry.DoWithResult(r, func() (*User, error) {
//		return fetchUserFromAPI(userID)
//	})
//	if err != nil {
//		log.Printf("failed to fetch user after 3 attempts: %v", err)
//	}
func DoWithResult[T any](r *retry, fn func() (T, error)) (T, error) {
	var result T
	err := r.Do(func() error {
		var retryErr error
		result, retryErr = fn()
		return retryErr
	})
	return result, err
}

// DoContext executes the given function with configured retry behavior while respecting context cancellation and deadlines.
// The function is retried until it succeeds, the maximum number of attempts is reached, or the context is cancelled/expired.
//
// Context awareness:
//   - Checks context before each attempt and returns immediately if cancelled or deadline exceeded
//   - Uses select during backoff sleep to detect context cancellation without waiting for full sleep duration
//   - Returns context.Canceled if context was cancelled, or context.DeadlineExceeded if deadline passed
//
// Returns nil if the operation succeeds, the context error if context is done, or the last error encountered if all retries fail
// or if the error is non-retryable according to shouldRetry.
// Returns an error if attempts is configured to less than 1.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	r := retry.New(retry.Attempts(3))
//	err := r.DoContext(ctx, func() error {
//		return someNetworkCall()
//	})
func (r *retry) DoContext(ctx context.Context, fn func() error) error {
	if r.attempts < 1 {
		return fmt.Errorf("attempts must be greater than 0")
	}

	var err error
	for i := 1; i <= r.attempts; i++ {
		// Check BEFORE attempt for context.Canceled or context.DeadlineExceeded errors
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		err = fn()
		if err == nil {
			return nil
		}

		// Without this check there is no way to catch derived context cancellations. E.g.
		//
		// ctx := context.Background()
		// err := retrier.DoContext(ctx, func() error {
		// 	derivedCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond) ----> We can't catch this
		// 	defer cancel()
		// 	return derivedCtx.Err()
		// })
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}

		// Check if we should retry this error
		if r.shouldRetry != nil && !r.shouldRetry(err) {
			// Error is not retryable, return immediately
			return err
		}

		if i < r.attempts {
			timer := time.NewTimer(r.backoff(i))
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}

	return err
}

// DoWithResultContext executes the given function with configured retry behavior, context support, and returns a result.
// Works like DoContext() but for functions that return a value along with an error.
// On failure, returns the result from the last attempt along with the final error.
func DoWithResultContext[T any](r *retry, ctx context.Context, fn func() (T, error)) (T, error) {
	var result T
	err := r.DoContext(ctx, func() error {
		var retryErr error
		result, retryErr = fn()
		return retryErr
	})
	return result, err
}
