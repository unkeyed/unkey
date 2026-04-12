package ratelimit

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel/tracing"

	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
)

// replayRequests processes buffered rate limit events by synchronizing them
// with Redis. It runs in a dedicated goroutine and consumes from the replay
// buffer until the buffer is closed.
func (s *service) replayRequests() {
	for ptr := range s.replayBuffer.Consume() {
		if ptr == nil {
			continue
		}
		err := s.syncWithOrigin(context.Background(), *ptr)
		if err != nil {
			logger.Error("failed to replay request", "error", err.Error())
		}
	}
}

// syncWithOrigin pushes a local rate limit increment to Redis and CAS-merges
// the global count back into the local atomic counter. No locks are held.
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

	durationMs := req.Duration.Milliseconds()
	sequence := calculateSequence(req.Time, req.Duration)
	key := counterKey{
		name:       req.Name,
		identifier: req.Identifier,
		durationMs: durationMs,
		sequence:   sequence,
	}

	newCounter, err := s.replayCircuitBreaker.Do(ctx, func(innerCtx context.Context) (int64, error) {
		innerCtx, innerCancel := context.WithTimeout(innerCtx, 2*time.Second)
		defer innerCancel()

		return s.counter.Increment(
			innerCtx,
			redisKey(key),
			req.Cost,
			req.Duration*3,
		)
	})

	if err != nil {
		tracing.RecordError(span, err)
		return err
	}

	// CAS-merge: update local counter if Redis value is higher.
	ptr, _ := s.loadCounter(key)
	casMerge(ptr, newCounter)

	return nil
}
