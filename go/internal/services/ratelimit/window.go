package ratelimit

import (
	"time"

	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
)

type window struct {
	// sequence = time.Now().UnixMilli() / duration
	sequence int64
	// Duration of the window in milliseconds.
	// This matches the duration from the original request and defines
	// how long this window remains active.
	duration time.Duration
	// Current token count in this window.
	// This is the actual count of tokens consumed during this window's
	// lifetime. It must never exceed the configured limit.
	counter int64
	// Start time of the window (Unix timestamp in milliseconds).
	// Used to:
	// - Calculate window expiration
	// - Determine if a window is still active
	// - Handle sliding window calculations between current and previous windows
	start time.Time
}

// newWindow creates a new rate limit window starting at the given time.
// Windows are aligned to duration boundaries (e.g., on the minute for
// minute-based limits) to ensure consistent behavior across nodes.
//
// Parameters:
//   - sequence: Monotonically increasing window identifier
//   - t: Time within the desired window
//   - duration: Length of the window
//
// Returns:
//   - *ratelimitv1.Window: A new window with:
//   - Start time aligned to duration boundary
//   - Counter initialized to 0
//   - Sequence number set for ordering
//
// Thread Safety:
//   - Thread-safe: creates new immutable window
//
// Performance:
//   - O(1) time complexity
//   - Allocates one Window struct
//
// Example:
//
//	window := newWindow(
//	    calculateSequence(time.Now(), time.Minute),
//	    time.Now(),
//	    time.Minute,
//	)
func newWindow(sequence int64, t time.Time, duration time.Duration) *window {
	metrics.RatelimitWindowsCreated.Inc()
	return &window{
		sequence: sequence,
		start:    t.Truncate(duration),
		duration: duration,
		counter:  0,
	}
}
