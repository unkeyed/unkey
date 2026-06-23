package ratelimit

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel/tracing"

	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
)

// errorReason buckets err for the origin_errors_total{reason} label so breaker
// short-circuits don't drown out the real failures that tripped it.
func errorReason(err error) string {
	if isCircuitOpen(err) {
		return "circuit_open"
	}
	// go-redis surfaces a blown deadline as a socket timeout that doesn't unwrap
	// to context.DeadlineExceeded, so check net.Error too.
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		return "timeout"
	}
	return "other"
}

// isCircuitOpen reports whether err is the breaker short-circuiting rather than a real origin failure.
func isCircuitOpen(err error) bool {
	return circuitbreaker.IsErrTripped(err) || circuitbreaker.IsErrTooManyRequests(err)
}

// originFetchTimeout caps how long a single Redis GET on the hot path may
// run before we fall back to local state. It is intentionally tight: the
// rate-limit decision sits on every authenticated request, and a stalled
// Redis must not propagate latency into customer traffic. If the fetch
// times out we treat the origin as unavailable for this request — local
// state is used, the circuit breaker tracks the failure, and the next
// caller will retry.
const originFetchTimeout = 300 * time.Millisecond

// originBreakerOpenTimeout caps how long the breaker stays open — i.e. how long
// ratelimiting runs local-only after a blip. Far below the 60s library default
// since in-region Redis recovers fast.
const originBreakerOpenTimeout = 5 * time.Second

// fetchFromOrigin returns the current counter value from the origin. The call is
// wrapped in the origin circuit breaker so that Redis outages fail fast instead
// of stalling every request in strict mode. On any failure (circuit tripped,
// timeout, or Redis error) it returns ok=false so callers can preserve local
// state without marking it fresh.
func (s *service) fetchFromOrigin(ctx context.Context, key counterKey, op string) (count int64, ok bool) {
	rk := key.redisKey()
	metrics.RatelimitOriginOperations.WithLabelValues(op).Inc()

	res, err := s.originCircuitBreaker.Do(ctx, func(ctx context.Context) (int64, error) {
		start := time.Now()
		timeout, cancel := context.WithTimeout(ctx, originFetchTimeout)
		defer cancel()

		res, err := s.origin.Get(timeout, rk)

		metrics.RatelimitOriginLatency.WithLabelValues(op).Observe(time.Since(start).Seconds())
		return res, err
	})
	if err != nil {
		metrics.RatelimitOriginErrors.WithLabelValues(op, errorReason(err)).Inc()
		// Don't log breaker short-circuits — they'd flood the log for the whole
		// open window; the circuit_open metric already tracks them.
		if !isCircuitOpen(err) {
			logger.Error("unable to get counter value from origin",
				"key", rk,
				"error", err.Error(),
			)
		}
		return 0, false
	}
	return res, true
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
		// Don't log breaker short-circuits (see fetchFromOrigin).
		if err != nil && !isCircuitOpen(err) {
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

	metrics.RatelimitOriginOperations.WithLabelValues("sync").Inc()

	durationMs := req.Duration.Milliseconds()
	sequence := calculateSequence(req.Time, req.Duration)
	key := counterKey{
		workspaceID: req.WorkspaceID,
		namespace:   req.Namespace,
		identifier:  req.Identifier,
		durationMs:  durationMs,
		sequence:    sequence,
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
		metrics.RatelimitOriginErrors.WithLabelValues("sync", errorReason(err)).Inc()
		tracing.RecordError(span, err)
		return err
	}

	// CAS-merge: update local counter if Redis value is higher.
	counter := s.loadCounter(key)
	atomicMax(&counter.val, newCounter)
	atomicMax(&counter.originFreshUntilMs, s.clock.Now().Add(originFreshDuration).UnixMilli())

	return nil
}
