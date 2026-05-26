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

// checkState holds the precomputed state needed for a sliding window
// check. One instance is built per request by prepareCheck and consumed
// by the CAS loop in Ratelimit (or the per-entry pass in RatelimitMany).
type checkState struct {
	cur       *counterEntry
	prev      *counterEntry
	strictKey strictKey

	// curGlobal and prevGlobal are snapshots of
	// counterEntry.globalCount taken at prepareCheck time so the
	// CAS retry loop in Ratelimit does not re-load them on every retry.
	// Cross-region counts only mutate from the cross-region pull
	// goroutine on a 10s cadence, so they are effectively constant for
	// the lifetime of any single request; snapshotting matches that
	// lifetime and saves two atomic loads per CAS attempt.
	curGlobal  int64
	prevGlobal int64

	curSequence   int64
	windowElapsed float64
	reset         time.Time
	source        string
}

// prepareCheck loads both counters for a request and fetches from
// Redis when strict mode is active for this identifier after a recent
// denial.
func (s *service) prepareCheck(ctx context.Context, req RatelimitRequest) checkState {
	durationMs := req.Duration.Milliseconds()
	curSeq := calculateSequence(req.Time, req.Duration)

	curKey := counterKey{workspaceID: req.WorkspaceID, namespace: req.Namespace, identifier: req.Identifier, durationMs: durationMs, sequence: curSeq}
	prevKey := counterKey{workspaceID: req.WorkspaceID, namespace: req.Namespace, identifier: req.Identifier, durationMs: durationMs, sequence: curSeq - 1}
	sk := strictKey{workspaceID: req.WorkspaceID, namespace: req.Namespace, identifier: req.Identifier, durationMs: durationMs}

	cur := s.loadCounter(curKey)
	prev := s.loadCounter(prevKey)

	// First caller per entry runs fetchFromOrigin; concurrent callers block
	// inside Do until it returns. Warm entries refresh from origin when their
	// last origin fetch is stale, preventing idle replicas from serving an old
	// local view for the rest of a long window.
	cur.EnsureFreshFromOrigin(ctx, req.Time)
	prev.EnsureFreshFromOrigin(ctx, req.Time)

	// Strict mode: a recent denial in this region forces an additional
	// synchronous origin fetch until the deadline passes. This is the
	// in-region convergence aid — instances within a region share state
	// through Redis, so the forced fetch drains any lag between this
	// instance's local view and the region's Redis-backed truth.
	// Cross-region convergence is handled separately via
	// cur.globalCount.
	if req.Time.UnixMilli() < s.loadStrictUntil(sk) {
		atomicMax(&cur.val, s.fetchFromOrigin(ctx, curKey))
		atomicMax(&prev.val, s.fetchFromOrigin(ctx, prevKey))
	}

	windowStartMs := curSeq * durationMs
	elapsedMs := req.Time.UnixMilli() - windowStartMs
	windowElapsed := float64(elapsedMs) / float64(durationMs)
	reset := time.UnixMilli(windowStartMs).Add(req.Duration)

	return checkState{
		cur:           cur,
		prev:          prev,
		strictKey:     sk,
		curGlobal:     cur.globalCount.Load(),
		prevGlobal:    prev.globalCount.Load(),
		curSequence:   curSeq,
		windowElapsed: windowElapsed,
		reset:         reset,
		source:        "local",
	}
}

// slidingWindowCount computes the effective request count for the
// sliding window given the current window's local counter value. Each
// window's total count is its own region's val (passed in for cur,
// atomically loaded for prev) plus the cross-region sum of other regions'
// contributions from the most recent cross-region pull. The two
// contributions are tracked separately to keep the cross-region merge
// from feeding back into the next flush, but for the deny decision
// they're equivalent: any source of count increases pressure on the
// limit. Cross-region snapshots come from checkState so the CAS retry
// loop doesn't re-pay the atomic load.
func (cs *checkState) slidingWindowCount(curCount int64) int64 {
	cur := curCount + cs.curGlobal
	prev := cs.prev.val.Load() + cs.prevGlobal
	return cur + int64(float64(prev)*(1.0-cs.windowElapsed))
}

func (s *service) Ratelimit(ctx context.Context, req RatelimitRequest) (RatelimitResponse, error) {
	_, span := tracing.Start(ctx, "Ratelimit")
	defer span.End()

	if req.Time.IsZero() {
		req.Time = s.clock.Now()
	}

	err := assert.All(
		assert.NotEmpty(req.WorkspaceID, "ratelimit workspace id must not be empty"),
		assert.NotEmpty(req.Namespace, "ratelimit namespace must not be empty"),
		assert.NotEmpty(req.Identifier, "ratelimit identifier must not be empty"),
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
			metrics.RatelimitDecision.WithLabelValues(req.WorkspaceID, cs.source, "denied").Inc()
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
			cs.cur.observeGlobalPushLimit(req.Limit)
			metrics.RatelimitDecision.WithLabelValues(req.WorkspaceID, cs.source, "passed").Inc()
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
		"workspace_id", req.WorkspaceID,
		"namespace", req.Namespace,
		"identifier", req.Identifier,
	)
	metrics.RatelimitCASExhausted.Inc()
	metrics.RatelimitDecision.WithLabelValues(req.WorkspaceID, cs.source, "denied").Inc()
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

	reqs = append([]RatelimitRequest(nil), reqs...)
	now := s.clock.Now()
	for i := range reqs {
		if reqs[i].Time.IsZero() {
			reqs[i].Time = now
		}

		err := assert.All(
			assert.NotEmpty(reqs[i].WorkspaceID, "ratelimit workspace id must not be empty"),
			assert.NotEmpty(reqs[i].Namespace, "ratelimit namespace must not be empty"),
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
		checks[i].cur.speculative.Add(req.Cost)
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
			checks[i].cur.speculative.Add(-req.Cost)
			// For each individual failure, set strict mode on its tuple so
			// later requests in this region force a Redis fetch and converge
			// with other instances' views of the count.
			if effectives[i] > req.Limit {
				s.setStrictUntil(checks[i].strictKey, req.Time.Add(req.Duration).UnixMilli())
			}
		}
	} else {
		// Increments stuck. Record each entry's global-push threshold so
		// the push goroutine can evaluate the live counter value later.
		for i, req := range reqs {
			checks[i].cur.observeGlobalPushLimit(req.Limit)
			checks[i].cur.speculative.Add(-req.Cost)
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
			metrics.RatelimitDecision.WithLabelValues(req.WorkspaceID, checks[i].source, "passed").Inc()
		} else {
			metrics.RatelimitDecision.WithLabelValues(req.WorkspaceID, checks[i].source, "denied").Inc()
		}
	}

	return responses, nil
}
