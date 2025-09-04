package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRetry(t *testing.T) {
	tests := []struct {
		name          string
		retry         *retry
		fn            func() error
		expectedCalls int
		expectedError bool
		expectedSleep time.Duration
	}{
		{
			name:  "success on first try",
			retry: New(),
			fn: func() error {
				return nil
			},
			expectedCalls: 1,
			expectedError: false,
			expectedSleep: 0,
		},
		{
			name:          "success after one failure",
			retry:         New(),
			fn:            failNTimes(1),
			expectedCalls: 2,
			expectedSleep: 100 * time.Millisecond,
			expectedError: false,
		},
		{
			name:  "all attempts fail",
			retry: New(),
			fn: func() error {
				return errors.New("persistent error")
			},
			expectedCalls: 3,
			expectedError: true,
			expectedSleep: 300 * time.Millisecond, // 100ms + 200ms (no sleep after final attempt)
		},
		{
			name:  "invalid attempts",
			retry: New(Attempts(0)),
			fn: func() error {
				return nil
			},
			expectedCalls: 0,
			expectedError: true,
			expectedSleep: 0,
		},
		{
			name:          "custom attempts - succeeds after 3 failures",
			retry:         New(Attempts(5)),
			fn:            failNTimes(3),
			expectedCalls: 4,
			expectedSleep: 600 * time.Millisecond,
			expectedError: false,
		},
		{
			name: "exponential backoff",
			retry: New(
				Attempts(3),
				Backoff(func(n int) time.Duration {
					return time.Duration(n*n) * time.Second
				}),
			),
			fn:            failNTimes(3),
			expectedCalls: 3,
			expectedSleep: 5 * time.Second, // 1s + 4s (no sleep after final attempt)
			expectedError: true,
		},
		{
			name: "constant backoff",
			retry: New(
				Attempts(3),
				Backoff(func(n int) time.Duration { return time.Second }),
			),
			fn:            failNTimes(3),
			expectedCalls: 3,
			expectedError: true,
			expectedSleep: 2 * time.Second, // 1s + 1s (no sleep after final attempt)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// instant sleep but record total time
			totalSleep := time.Duration(0)
			tt.retry.sleep = func(d time.Duration) {
				totalSleep += d
				t.Logf("fake sleeping for %s\n", d)
			}
			calls := 0

			wrappedFn := func() error {
				calls++
				t.Logf("called")
				return tt.fn()
			}

			err := tt.retry.Do(wrappedFn)

			require.Equal(t, tt.expectedCalls, calls, "unexpected number of calls")
			require.Equal(t, tt.expectedSleep, totalSleep, "unexpected sleep duration")

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

		})
	}
}

// failNTimes returns a function that will fail n times and then return the finalError.
// If finalError is nil, it will return success after n failures.
func failNTimes(n int) func() error {
	attempt := 0
	return func() error {
		attempt += 1
		if attempt <= n {
			return errors.New("temporary error")
		}
		return nil
	}
}

func TestShouldRetry(t *testing.T) {
	nonRetryableError := errors.New("non-retryable")
	retryableError := errors.New("retryable")
	
	tests := []struct {
		name          string
		shouldRetry   func(error) bool
		errorSequence []error
		expectedCalls int
		expectedError error
		expectedSleep time.Duration
	}{
		{
			name: "should retry all errors by default",
			shouldRetry: nil, // default behavior
			errorSequence: []error{retryableError, retryableError, nil},
			expectedCalls: 3,
			expectedError: nil,
			expectedSleep: 300 * time.Millisecond, // 100ms + 200ms
		},
		{
			name: "should not retry non-retryable errors",
			shouldRetry: func(err error) bool {
				return err != nonRetryableError
			},
			errorSequence: []error{nonRetryableError},
			expectedCalls: 1,
			expectedError: nonRetryableError,
			expectedSleep: 0, // no retry, no sleep
		},
		{
			name: "should retry retryable errors but not non-retryable ones",
			shouldRetry: func(err error) bool {
				return err != nonRetryableError
			},
			errorSequence: []error{retryableError, nonRetryableError},
			expectedCalls: 2,
			expectedError: nonRetryableError,
			expectedSleep: 100 * time.Millisecond, // only one retry before hitting non-retryable
		},
		{
			name: "should eventually succeed after retrying retryable errors",
			shouldRetry: func(err error) bool {
				return err != nonRetryableError
			},
			errorSequence: []error{retryableError, retryableError, nil},
			expectedCalls: 3,
			expectedError: nil,
			expectedSleep: 300 * time.Millisecond, // 100ms + 200ms
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var retrier *retry
			if tt.shouldRetry != nil {
				retrier = New(ShouldRetry(tt.shouldRetry))
			} else {
				retrier = New()
			}
			
			totalSleep := time.Duration(0)
			retrier.sleep = func(d time.Duration) {
				totalSleep += d
			}
			
			calls := 0
			err := retrier.Do(func() error {
				if calls < len(tt.errorSequence) {
					err := tt.errorSequence[calls]
					calls++
					return err
				}
				calls++
				return nil
			})
			
			require.Equal(t, tt.expectedCalls, calls, "unexpected number of calls")
			require.Equal(t, tt.expectedSleep, totalSleep, "unexpected sleep duration")
			
			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
