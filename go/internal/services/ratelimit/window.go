package ratelimit

import (
	"context"
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/metrics"
)

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
func newWindow(sequence int64, t time.Time, duration time.Duration) *ratelimitv1.Window {
	metrics.Ratelimit.CreatedWindows.Add(context.Background(), 1)
	return &ratelimitv1.Window{
		Sequence: sequence,
		Start:    t.Truncate(duration).UnixMilli(),
		Duration: duration.Milliseconds(),
		Counter:  0,
	}
}

// setWindowRequest contains parameters for updating a rate limit window.
// Used to synchronize window state across cluster nodes.
//
// Thread Safety:
//   - Immutable after creation
//   - Safe for concurrent use
type setWindowRequest struct {
	// Identifier uniquely identifies the rate limit subject
	Identifier string

	// Limit is the maximum allowed requests per duration
	Limit int64

	// Duration is the time window length
	Duration time.Duration

	// Sequence uniquely identifies this window
	Sequence int64

	// Time is any timestamp within the target window
	// Will be aligned to window boundaries
	Time time.Time

	// Counter is the new request count for this window
	Counter int64
}

// SetWindows updates the state of one or more rate limit windows.
// Used to synchronize window state across cluster nodes and handle
// replay requests from other nodes.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - requests: Window states to update
//
// Thread Safety:
//   - Safe for concurrent use
//   - Updates are atomic per window
//
// Performance:
//   - O(n) where n is number of requests
//   - Acquires/releases bucket mutex for each window
//
// Behavior:
//   - Only increases window counters, never decreases
//   - Creates missing windows/buckets as needed
//   - Maintains monotonic counter invariant
//
// Example:
//
//	svc.SetWindows(ctx, setWindowRequest{
//	    Identifier: "user-123",
//	    Limit:      100,
//	    Duration:   time.Minute,
//	    Sequence:   42,
//	    Time:       time.Now(),
//	    Counter:    5,
//	})
func (r *service) SetWindows(ctx context.Context, requests ...setWindowRequest) {
	for _, req := range requests {
		key := bucketKey{req.Identifier, req.Limit, req.Duration}
		bucket, _ := r.getOrCreateBucket(key)
		bucket.mu.Lock()
		window, ok := bucket.windows[req.Sequence]
		if !ok {
			window = newWindow(req.Sequence, req.Time, req.Duration)
			bucket.windows[req.Sequence] = window
		}

		// Only increment the current value if the new value is greater than the current value
		// Due to varying network latency, we may receive out of order responses and could decrement the
		// current value, which would result in inaccurate rate limiting
		if req.Counter > window.GetCounter() {
			window.Counter = req.Counter
		}
		bucket.mu.Unlock()

	}
}
