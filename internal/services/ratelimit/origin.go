package ratelimit

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel/tracing"

	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
)

// fetchFromOrigin fetches the current counter value from the origin and
// CAS-merges it into the local atomic counter. The call is wrapped in the
// origin circuit breaker so that Redis outages fail fast instead of stalling
// every request in strict mode. On any failure (circuit tripped, timeout,
// or Redis error) the local counter is left unchanged — callers proceed
// with whatever local state they have.
func (s *service) fetchFromOrigin(ctx context.Context, key counterKey, local *atomic.Int64) {
	rk := key.redisKey()

	res, err := s.originCircuitBreaker.Do(ctx, func(ctx context.Context) (int64, error) {
		start := time.Now()
		timeout, cancel := context.WithTimeout(ctx, time.Millisecond*150)
		defer cancel()

		res, err := s.origin.Get(timeout, rk)

		metrics.RatelimitOriginLatency.WithLabelValues("fetch").Observe(time.Since(start).Seconds())
		return res, err
	})
	if err != nil {
		metrics.RatelimitOriginErrors.WithLabelValues("fetch").Inc()
		logger.Error("unable to get counter value from origin",
			"key", rk,
			"error", err.Error(),
		)
		return
	}
	atomicMax(local, res)
}

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
		metrics.RatelimitOriginLatency.WithLabelValues("sync").Observe(time.Since(start).Seconds())
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

	newCounter, err := s.originCircuitBreaker.Do(ctx, func(innerCtx context.Context) (int64, error) {
		innerCtx, innerCancel := context.WithTimeout(innerCtx, 2*time.Second)
		defer innerCancel()

		return s.origin.Increment(
			innerCtx,
			key.redisKey(),
			req.Cost,
			req.Duration*3,
		)
	})
	if err != nil {
		metrics.RatelimitOriginErrors.WithLabelValues("sync").Inc()
		tracing.RecordError(span, err)
		return err
	}

	// CAS-merge: update local counter if Redis value is higher.
	counter := s.loadCounter(key)
	atomicMax(&counter.val, newCounter)

	return nil
}
