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
			expectedSleep: 600 * time.Millisecond,
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
			expectedSleep: 14 * time.Second,
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
			expectedSleep: 3 * time.Second,
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
