// Package retry provides functionality to retry operations with configurable attempts and backoff.
package retry

import (
	"fmt"
	"time"
)

// retry holds the configuration for retry attempts and backoff strategy.
type retry struct {
	// attempts is the maximum number of times to try the operation
	attempts int

	// backoff is a function that returns the duration to wait before the next retry
	// based on the current attempt number (zero-based)
	backoff func(n int) time.Duration

	// shouldRetry is a function that determines if an error is retryable
	// If nil, all errors are considered retryable
	shouldRetry func(error) bool

	// used for testing
	// overwrite time.Sleep to speed up tests
	sleep func(d time.Duration)
}

// Build creates a new retry instance with default configuration.
// Default configuration:
//   - 3 retry attempts
//   - Linear backoff starting at 100ms, increasing by 100ms per attempt
//
// Example:
//
//	r := retry.Build()
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
// r := retry.Build(
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
