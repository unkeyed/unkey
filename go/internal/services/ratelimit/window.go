package ratelimit

import (
	"context"
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1"
)

func newWindow(sequence int64, t time.Time, duration time.Duration) *ratelimitv1.Window {
	return &ratelimitv1.Window{
		Sequence: sequence,
		Start:    t.Truncate(duration).UnixMilli(),
		Duration: duration.Milliseconds(),
		Counter:  0,
	}
}

type setWindowRequest struct {
	Identifier string
	Limit      int64
	Duration   time.Duration
	Sequence   int64
	// any time within the window
	Time    time.Time
	Counter int64
}

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
			r.logger.Info("updating window",
				"key", key.toString(),
				"current", window.GetCounter(),
				"new", req.Counter,
			)

			window.Counter = req.Counter
		}
		bucket.mu.Unlock()

	}
}
