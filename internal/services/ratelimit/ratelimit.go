package ratelimit

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel/tracing"

	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
	"go.opentelemetry.io/otel/attribute"
)

// checkState holds the precomputed state needed for a sliding window check.
type checkState struct {
	cur           *counterEntry
	prev          *counterEntry
	strictKey     strictKey
	windowElapsed float64
	reset         time.Time
	source        string
}

// prepareCheck loads both window counters for a request, fetching from Redis
// if either window is cold or if strict-enforcement mode is active for this
// identifier after a recent denial.
func (s *service) prepareCheck(ctx context.Context, req RatelimitRequest) checkState {
	durationMs := req.Duration.Milliseconds()
	curSeq := calculateSequence(req.Time, req.Duration)

	curKey := counterKey{name: req.Name, identifier: req.Identifier, durationMs: durationMs, sequence: curSeq}
	prevKey := counterKey{name: req.Name, identifier: req.Identifier, durationMs: durationMs, sequence: curSeq - 1}
	sk := strictKey{name: req.Name, identifier: req.Identifier, durationMs: durationMs}

	cur := s.loadCounter(curKey)
	prev := s.loadCounter(prevKey)

	// First caller per entry runs fetchFromOrigin; concurrent callers block
	// inside Do until it returns, then take the fast path (single atomic
	// load of hydrated) forever. This prevents late arrivals on a cold key
	// from reading a zero counter while the owner is still fetching.
	cur.Hydrate(ctx)
	prev.Hydrate(ctx)

	// Strict mode: a recent denial forces an additional synchronous origin
	// fetch until the deadline passes, catching up local state to the global
	// count regardless of whether the entry was already hydrated.
	if req.Time.UnixMilli() < s.loadStrictUntil(sk) {
		if v, err := s.fetchFromOrigin(ctx, curKey); err == nil {
			atomicMax(&cur.val, v)
		}
		if v, err := s.fetchFromOrigin(ctx, prevKey); err == nil {
			atomicMax(&prev.val, v)
		}
	}

	windowStartMs := curSeq * durationMs
	elapsedMs := req.Time.UnixMilli() - windowStartMs
	windowElapsed := float64(elapsedMs) / float64(durationMs)
	reset := time.UnixMilli(windowStartMs).Add(req.Duration)

	return checkState{
		cur:           cur,
		prev:          prev,
		strictKey:     sk,
		windowElapsed: windowElapsed,
		reset:         reset,
		source:        "local",
	}
}

// slidingWindowCount computes the effective request count for the sliding window
// given the current window's counter value. The previous window is loaded atomically.
func (cs *checkState) slidingWindowCount(curCount int64) int64 {
	return curCount + int64(float64(cs.prev.val.Load())*(1.0-cs.windowElapsed))
}

func (s *service) Ratelimit(ctx context.Context, req RatelimitRequest) (RatelimitResponse, error) {
	_, span := tracing.Start(ctx, "Ratelimit")
	defer span.End()

	if req.Time.IsZero() {
		req.Time = s.clock.Now()
	}

	err := assert.All(
		assert.NotEmpty(req.Identifier, "ratelimit identifier must not be empty"),
		assert.NotEmpty(req.Name, "ratelimit name must not be empty"),
		assert.Greater(req.Limit, 0, "ratelimit limit must be greater than zero"),
		assert.GreaterOrEqual(req.Cost, 0, "ratelimit cost must not be negative"),
		assert.GreaterOrEqual(req.Duration.Milliseconds(), 1000, "ratelimit duration must be at least 1s"),
		assert.False(req.Time.IsZero(), "request time must not be zero"),
	)
	if err != nil {
		return RatelimitResponse{}, err
	}

	cs := s.prepareCheck(ctx, req)

	// CAS loop: atomically check the sliding window and increment if allowed.
	// Denials are wait-free (single Load, no CAS, no retry).
	// Allows retry only when another goroutine incremented the same counter
	// between our Load and CAS — nanoseconds of spinning.
	// Bounded to maxCASRetries to prevent livelock. If exhausted, deny the
	// request (fail closed) — this indicates pathological contention that
	// should never occur in practice.
	for range maxCASRetries {
		curCount := cs.cur.val.Load()
		effectiveCount := cs.slidingWindowCount(curCount) + req.Cost

		if effectiveCount > req.Limit {
			// Enter strict mode: force origin fetches for the rest of the
			// rate-limit window so later requests converge on the true count.
			s.setStrictUntil(cs.strictKey, req.Time.Add(req.Duration).UnixMilli())
			metrics.RatelimitDecision.WithLabelValues(cs.source, "denied").Inc()
			span.SetAttributes(attribute.Bool("passed", false))
			return RatelimitResponse{
				Success:   false,
				Remaining: 0,
				Reset:     cs.reset,
				Limit:     req.Limit,
				Current:   effectiveCount,
			}, nil
		}

		if cs.cur.val.CompareAndSwap(curCount, curCount+req.Cost) {
			s.replayBuffer.Buffer(req)
			metrics.RatelimitDecision.WithLabelValues(cs.source, "passed").Inc()
			span.SetAttributes(attribute.Bool("passed", true))
			return RatelimitResponse{
				Success:   true,
				Remaining: max(0, req.Limit-effectiveCount),
				Reset:     cs.reset,
				Limit:     req.Limit,
				Current:   effectiveCount,
			}, nil
		}
	}

	// CAS retries exhausted — fail closed.
	logger.Error("ratelimit CAS retries exhausted, denying request",
		"identifier", req.Identifier,
		"name", req.Name,
	)
	metrics.RatelimitCASExhausted.Inc()
	metrics.RatelimitDecision.WithLabelValues(cs.source, "denied").Inc()
	span.SetAttributes(attribute.Bool("passed", false))
	return RatelimitResponse{
		Success:   false,
		Remaining: 0,
		Reset:     cs.reset,
		Limit:     req.Limit,
		Current:   req.Limit,
	}, nil
}

func (s *service) RatelimitMany(ctx context.Context, reqs []RatelimitRequest) ([]RatelimitResponse, error) {
	_, span := tracing.Start(ctx, "RatelimitMany")
	defer span.End()

	now := s.clock.Now()
	for i := range reqs {
		if reqs[i].Time.IsZero() {
			reqs[i].Time = now
		}

		err := assert.All(
			assert.NotEmpty(reqs[i].Name, "ratelimit name must not be empty"),
			assert.NotEmpty(reqs[i].Identifier, "ratelimit identifier must not be empty"),
			assert.Greater(reqs[i].Limit, 0, "ratelimit limit must be greater than zero"),
			assert.GreaterOrEqual(reqs[i].Cost, 0, "ratelimit cost must not be negative"),
			assert.GreaterOrEqual(reqs[i].Duration.Milliseconds(), 1000, "ratelimit duration must be at least 1s"),
			assert.False(reqs[i].Time.IsZero(), "request time must not be zero"),
		)
		if err != nil {
			return []RatelimitResponse{}, err
		}
	}

	// Prepare all checks (load counters, fetch from origin if cold).
	checks := make([]checkState, len(reqs))
	for i, req := range reqs {
		checks[i] = s.prepareCheck(ctx, req)
	}

	// Optimistically increment all counters. Add returns the new value atomically,
	// so each goroutine sees its own increment reflected.
	newCounts := make([]int64, len(reqs))
	for i, req := range reqs {
		newCounts[i] = checks[i].cur.val.Add(req.Cost)
	}

	// Check all limits using the post-increment values.
	allPassed := true
	effectives := make([]int64, len(reqs))
	for i, req := range reqs {
		effectives[i] = checks[i].slidingWindowCount(newCounts[i])
		if effectives[i] > req.Limit {
			allPassed = false
		}
	}

	span.SetAttributes(attribute.Bool("passed", allPassed))

	// If any limit was exceeded, roll back all increments.
	// The rollback window (between Add and here) is nanoseconds. During that
	// window other goroutines see temporarily inflated counters, which may cause
	// false denials but never false allows — the transient state is conservative.
	if !allPassed {
		for i, req := range reqs {
			checks[i].cur.val.Add(-req.Cost)
			// For each individual failure, set strict mode on its tuple.
			if effectives[i] > req.Limit {
				s.setStrictUntil(checks[i].strictKey, req.Time.Add(req.Duration).UnixMilli())
			}
		}
	}

	// Build responses.
	//
	// Response.Success reflects whether each entry individually passed its limit.
	// Counter side effects are all-or-nothing: if any entry failed, counters for
	// ALL entries were rolled back above. This preserves consistency (no partial
	// batches) while still telling the caller which specific limit was exceeded.
	responses := make([]RatelimitResponse, len(reqs))
	for i, req := range reqs {
		effective := effectives[i]
		individualPassed := effective <= req.Limit
		remaining := req.Limit - effective
		if !allPassed {
			// Counters were rolled back. For entries that passed individually,
			// add the cost back to Remaining since nothing was actually consumed.
			if individualPassed {
				remaining += req.Cost
			}
		}

		responses[i] = RatelimitResponse{
			Success:   individualPassed,
			Remaining: max(0, remaining),
			Reset:     checks[i].reset,
			Limit:     req.Limit,
			Current:   effective,
		}

		// Only buffer for replay if the counter is actually incremented (allPassed).
		if allPassed && individualPassed {
			s.replayBuffer.Buffer(req)
		}

		if individualPassed {
			metrics.RatelimitDecision.WithLabelValues(checks[i].source, "passed").Inc()
		} else {
			metrics.RatelimitDecision.WithLabelValues(checks[i].source, "denied").Inc()
		}
	}

	return responses, nil
}
