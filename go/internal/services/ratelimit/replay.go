package ratelimit

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
)

// replayRequests processes buffered rate limit events by synchronizing them with
// the origin nodes. It ensures eventual consistency across the cluster by
// replaying local decisions to the authoritative nodes.
//
// This method:
// 1. Continuously consumes events from the replay buffer
// 2. Forwards each event to its origin node
// 3. Updates local state with the origin's response
//
// Thread Safety:
//   - Must be run in a dedicated goroutine
//   - Safe for concurrent buffer producers
//   - Uses internal synchronization for state updates
//
// Performance:
//   - Batches requests for efficiency
//   - Uses circuit breaker to prevent cascading failures
//   - Automatic retry on transient errors
//
// Example Usage:
//
//	// Start replay processing
//	for range 8 {
//	    go svc.replayRequests()
//	}
func (s *service) replayRequests() {
	for req := range s.replayBuffer.Consume() {
		err := s.syncWithOrigin(context.Background(), req)
		if err != nil {
			s.logger.Error("failed to replay request", "error", err.Error())
		}

	}

}

func (s *service) syncWithOrigin(ctx context.Context, req RatelimitRequest) error {
	defer func(start time.Time) {
		metrics.RatelimitOriginSyncLatency.Observe(time.Since(start).Seconds())
	}(time.Now())

	ctx, span := tracing.Start(ctx, "syncWithOrigin")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := assert.False(req.Time.IsZero(), "request time must not be zero when replaying")
	if err != nil {
		return err
	}
	key := bucketKey{
		identifier: req.Identifier,
		limit:      req.Limit,
		duration:   req.Duration,
	}

	bucket, _ := s.getOrCreateBucket(key)
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	currentWindow, _ := bucket.getCurrentWindow(req.Time)

	newCounter, err := s.replayCircuitBreaker.Do(ctx, func(innerCtx context.Context) (int64, error) {
		innerCtx, cancel = context.WithTimeout(innerCtx, 2*time.Second)
		defer cancel()

		return s.counter.Increment(
			innerCtx,
			counterKey(key, currentWindow.sequence),
			req.Cost,
			currentWindow.duration*3,
		)

	})
	if err != nil {
		tracing.RecordError(span, err)

		return err
	}
	if newCounter > currentWindow.counter {
		currentWindow.counter = newCounter
	}

	return nil
}
