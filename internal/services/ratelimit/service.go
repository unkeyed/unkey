package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/buffer"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel/tracing"

	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
	"go.opentelemetry.io/otel/attribute"
)

// service implements lockless distributed rate limiting using a sliding window
// algorithm with atomic counters.
//
// All rate limit state is stored in a flat sync.Map of *atomic.Int64 counters,
// keyed by (name, identifier, duration, sequence). There are no mutexes in the
// hot path.
//
// Ratelimit uses a CAS loop: denials are wait-free (single atomic Load), allows
// retry only when another goroutine incremented the same counter between Load
// and CAS. RatelimitMany uses optimistic Add with rollback for atomic
// all-or-nothing batch semantics.
//
// Local counters are eventually consistent with Redis. Background replay workers
// push local increments to Redis and CAS-merge the global count back into the
// local atomic counter.
type service struct {
	clock clock.Clock

	// counters maps counterKey -> *atomic.Int64.
	// Each entry represents one sliding window's request count for a specific
	// (name, identifier, duration) combination.
	counters sync.Map

	// counter is the distributed counter backend (Redis).
	counter counter.Counter

	// replayBuffer holds rate limit events for async propagation to Redis.
	replayBuffer *buffer.Buffer[RatelimitRequest]

	// replayCircuitBreaker prevents cascading failures during Redis communication.
	replayCircuitBreaker circuitbreaker.CircuitBreaker[int64]
}

// Config holds configuration for creating a new rate limiting service.
type Config struct {
	// Clock for time-related operations. If nil, uses system clock.
	Clock clock.Clock

	// Counter is the distributed counter backend (typically Redis).
	Counter counter.Counter
}

// New creates a new rate limiting service.
//
// The service starts 8 background goroutines to process the replay buffer,
// synchronizing local rate limit state with Redis. It also starts a goroutine
// to periodically clean up expired counters.
//
// Call Close when done to release resources.
func New(config Config) (*service, error) {
	if config.Clock == nil {
		config.Clock = clock.New()
	}

	s := &service{
		clock:    config.Clock,
		counters: sync.Map{}, //nolint:exhaustruct // sync.Map zero value is ready to use
		counter:  config.Counter,
		replayBuffer: buffer.New[RatelimitRequest](buffer.Config{
			Name:     "ratelimit_replays",
			Capacity: 10_000,
			Drop:     true,
		}),
		replayCircuitBreaker: circuitbreaker.New[int64]("replayRatelimitRequest"),
	}

	s.startJanitor()

	for range 8 {
		go s.replayRequests()
	}

	return s, nil
}

// Close stops the replay buffer and releases resources.
func (s *service) Close() error {
	s.replayBuffer.Close()
	return nil
}

// loadCounter returns the atomic counter for the given key, creating it if needed.
// The second return value is true if the counter already existed.
func (s *service) loadCounter(key counterKey) (*atomic.Int64, bool) {
	val, loaded := s.counters.LoadOrStore(key, &atomic.Int64{})
	if !loaded {
		metrics.RatelimitWindowsCreated.Inc()
	}
	return val.(*atomic.Int64), loaded
}

// fetchFromOrigin fetches the current counter value from Redis and CAS-merges
// it into the local atomic counter. Errors are logged but do not fail the request.
func (s *service) fetchFromOrigin(ctx context.Context, key counterKey, ptr *atomic.Int64) {
	rk := redisKey(key)
	res, err := s.counter.Get(ctx, rk)
	if err != nil {
		logger.Error("unable to get counter value from origin",
			"key", rk,
			"error", err.Error(),
		)
		return
	}
	casMerge(ptr, res)
}

// casMerge atomically updates ptr to val if val is higher than the current value.
// Bounded to maxCASRetries attempts to prevent livelock under pathological contention.
const maxCASRetries = 100

func casMerge(ptr *atomic.Int64, val int64) {
	for range maxCASRetries {
		cur := ptr.Load()
		if val <= cur {
			return
		}
		if ptr.CompareAndSwap(cur, val) {
			return
		}
	}
	logger.Warn("casMerge retries exhausted, proceeding with stale counter")
}

// checkState holds the precomputed state needed for a sliding window check.
type checkState struct {
	curPtr        *atomic.Int64
	prevPtr       *atomic.Int64
	windowElapsed float64
	reset         time.Time
	source        string
}

// prepareCheck loads both window counters for a request, fetching from Redis
// if either window is cold.
func (s *service) prepareCheck(ctx context.Context, req RatelimitRequest) checkState {
	durationMs := req.Duration.Milliseconds()
	curSeq := calculateSequence(req.Time, req.Duration)

	curKey := counterKey{name: req.Name, identifier: req.Identifier, durationMs: durationMs, sequence: curSeq}
	prevKey := counterKey{name: req.Name, identifier: req.Identifier, durationMs: durationMs, sequence: curSeq - 1}

	curPtr, curExisted := s.loadCounter(curKey)
	prevPtr, prevExisted := s.loadCounter(prevKey)

	source := "local"
	if !curExisted || !prevExisted {
		source = "origin"
		if !curExisted {
			s.fetchFromOrigin(ctx, curKey, curPtr)
		}
		if !prevExisted {
			s.fetchFromOrigin(ctx, prevKey, prevPtr)
		}
	}

	windowStartMs := curSeq * durationMs
	elapsedMs := req.Time.UnixMilli() - windowStartMs
	windowElapsed := float64(elapsedMs) / float64(durationMs)
	reset := time.UnixMilli(windowStartMs).Add(req.Duration)

	return checkState{
		curPtr:        curPtr,
		prevPtr:       prevPtr,
		windowElapsed: windowElapsed,
		reset:         reset,
		source:        source,
	}
}

// slidingWindowCount computes the effective request count for the sliding window
// given the current window's counter value. The previous window is loaded atomically.
func (cs *checkState) slidingWindowCount(curCount int64) int64 {
	return curCount + int64(float64(cs.prevPtr.Load())*(1.0-cs.windowElapsed))
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
		curCount := cs.curPtr.Load()
		effectiveCount := cs.slidingWindowCount(curCount) + req.Cost

		if effectiveCount > req.Limit {
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

		if cs.curPtr.CompareAndSwap(curCount, curCount+req.Cost) {
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

	for i := range reqs {
		if reqs[i].Time.IsZero() {
			reqs[i].Time = s.clock.Now()
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
		newCounts[i] = checks[i].curPtr.Add(req.Cost)
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
			checks[i].curPtr.Add(-req.Cost)
		}
	}

	// Build responses.
	responses := make([]RatelimitResponse, len(reqs))
	for i, req := range reqs {
		effective := effectives[i]
		passed := allPassed && effective <= req.Limit
		remaining := req.Limit - effective
		if !allPassed {
			// Counters were rolled back, so remaining should reflect un-incremented state.
			remaining += req.Cost
		}

		responses[i] = RatelimitResponse{
			Success:   passed,
			Remaining: max(0, remaining),
			Reset:     checks[i].reset,
			Limit:     req.Limit,
			Current:   effective,
		}

		if passed {
			s.replayBuffer.Buffer(req)
			metrics.RatelimitDecision.WithLabelValues(checks[i].source, "passed").Inc()
		} else {
			metrics.RatelimitDecision.WithLabelValues(checks[i].source, "denied").Inc()
		}
	}

	return responses, nil
}
