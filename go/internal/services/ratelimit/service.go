package ratelimit

import (
	"context"
	"sync"

	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/counter"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
	"go.opentelemetry.io/otel/attribute"
)

// service implements distributed rate limiting using a sliding window algorithm.
//
// The service maintains an in-memory cache of rate limit windows that is synchronized
// with Redis via an async replay buffer. This hybrid approach provides low latency
// for common cases while ensuring accuracy across multiple nodes.
//
// Local counters are eventually consistent with Redis. During the brief window before
// replay completes, multiple nodes may each allow slightly more requests than the
// configured limit (over-admission). This trade-off was chosen to minimize latency
// for the 99% case while accepting occasional slack in the rate limit.
type service struct {
	// clock provides time-related functionality, can be mocked for testing
	clock clock.Clock

	// logger handles structured logging output
	logger logging.Logger

	// shutdownCh signals service shutdown
	shutdownCh chan struct{}

	// bucketsMu protects access to the buckets map
	bucketsMu sync.RWMutex

	// buckets maps identifier+sequence to rate limit buckets
	// Protected by bucketsMu
	buckets map[string]*bucket

	// counter is the distributed counter implementation
	counter counter.Counter

	// replayBuffer holds rate limit events for async propagation
	// Thread-safe internally
	replayBuffer *buffer.Buffer[RatelimitRequest]

	// replayCircuitBreaker prevents cascading failures during peer communication
	// Thread-safe internally
	replayCircuitBreaker circuitbreaker.CircuitBreaker[int64]
}

// Config holds configuration for creating a new rate limiting service.
type Config struct {
	Logger logging.Logger

	// Clock for time-related operations. If nil, uses system clock.
	Clock clock.Clock

	// Counter is the distributed counter backend (typically Redis).
	// Required - rate limiting cannot function without a counter.
	Counter counter.Counter
}

// New creates a new rate limiting service.
//
// The service starts 8 background goroutines to process the replay buffer,
// synchronizing local rate limit state with Redis. It also starts a goroutine
// to periodically clean up expired rate limit windows.
//
// Call Close when done to release resources.
func New(config Config) (*service, error) {
	if config.Clock == nil {
		config.Clock = clock.New()
	}

	s := &service{
		clock:      config.Clock,
		logger:     config.Logger,
		shutdownCh: make(chan struct{}),
		bucketsMu:  sync.RWMutex{},
		buckets:    make(map[string]*bucket),
		counter:    config.Counter,
		replayBuffer: buffer.New[RatelimitRequest](buffer.Config{
			Name:     "ratelimit_replays",
			Capacity: 10_000,
			Drop:     true,
		}),
		replayCircuitBreaker: circuitbreaker.New[int64]("replayRatelimitRequest"),
	}

	s.expireWindowsAndBuckets()

	// start multiple goroutines to do replays
	for range 8 {
		go s.replayRequests()
	}

	return s, nil
}

// Close stops the replay buffer and releases resources.
// The service must not be used after calling Close.
func (s *service) Close() error {
	s.replayBuffer.Close()
	return nil
}

// calculateRateLimit implements the sliding window algorithm to determine if a request
// would exceed the rate limit.
//
// The algorithm smoothly transitions between time windows by weighting the previous
// window's count based on how far into the current window we are. This provides more
// accurate rate limiting than fixed windows while remaining simple to implement.
//
// Returns (exceeded, effectiveCount, remaining) where remaining can be negative if
// the limit is exceeded. Callers should clamp remaining to 0 before returning to users.
func (s *service) calculateRateLimit(req RatelimitRequest, currentWindow, previousWindow *window) (bool, int64, int64) {
	// Calculate time elapsed in current window (as a fraction)
	windowElapsed := float64(req.Time.Sub(currentWindow.start).Milliseconds()) / float64(req.Duration.Milliseconds())

	// Pure sliding window calculation:
	// - We count 100% of current window
	// - We count a decreasing portion of previous window based on how far we are into current window
	effectiveCount := currentWindow.counter + int64(float64(previousWindow.counter)*(1.0-windowElapsed))

	effectiveCount += req.Cost

	// Calculate remaining (could be negative if limit is exceeded)
	remaining := req.Limit - effectiveCount

	// Check if this request would exceed the limit
	exceeded := effectiveCount > req.Limit

	return exceeded, effectiveCount, remaining
}

// Ratelimit checks multiple rate limits atomically.
//
// All rate limit checks must pass for the request to be allowed. If any limit fails,
// none of the counters are incremented. This all-or-nothing behavior prevents counter
// leaks when a key has multiple rate limits (e.g., per-minute and per-month).
//
// The method tries to make decisions using local cached data when possible. If local
// data is insufficient (first request or after strictUntil period), it fetches current
// counts from Redis. When all checks pass, counters are incremented locally and the
// changes are asynchronously propagated to Redis via the replay buffer.
//
// Returns validation errors for invalid request parameters (empty identifier, zero limit,
// negative cost, or duration less than 1 second).
func (s *service) Ratelimit(ctx context.Context, reqs []RatelimitRequest) ([]RatelimitResponse, error) {
	_, span := tracing.Start(ctx, "Ratelimit")
	defer span.End()

	for i := range reqs {
		if reqs[i].Time.IsZero() {
			reqs[i].Time = s.clock.Now()
		}

		err := assert.All(
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

	responses := make([]RatelimitResponse, len(reqs))

	for i, req := range reqs {
		res, err := s.handleBucket(ctx, req)
		if err != nil {
			return nil, err
		}
		responses[i] = res
	}

	allPassed := true

	for i, res := range responses {
		if !res.Success {
			allPassed = false
			for j := range i {
				s.rollback(reqs[j])
			}
			span.SetAttributes(attribute.Bool("denied", true))
			break
		}
	}

	if allPassed {
		span.SetAttributes(attribute.Bool("passed", true))

		for _, req := range reqs {
			s.replayBuffer.Buffer(req)
		}
	} else {
		// When batch fails, add back the cost since we're not consuming tokens
		for i := range responses {
			responses[i].Remaining += reqs[i].Cost
		}
	}

	// Clamp all Remaining values to 0 before returning
	for i := range responses {
		responses[i].Remaining = max(0, responses[i].Remaining)
	}

	return responses, nil

}

// rollback decrements the local counter for a request that was optimistically incremented
// during checking but later failed due to another rate limit in the batch failing.
//
// This is only called when processing a batch of rate limits where earlier checks passed
// but a later check failed, requiring us to undo the optimistic counter increments.
func (s *service) rollback(req RatelimitRequest) {

	key := bucketKey{req.Identifier, req.Limit, req.Duration}

	b, bucketExisted := s.getOrCreateBucket(key)
	if !bucketExisted {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	currentWindow, existed := b.getCurrentWindow(req.Time)
	if existed {
		currentWindow.counter = max(0, currentWindow.counter-req.Cost)
	}

}

// handleBucket evaluates a single rate limit request against its bucket.
//
// The method attempts to make decisions using only local cached data when possible.
// If both the current and previous windows exist locally, it can determine whether
// the request would exceed the limit without contacting Redis.
//
// If local data is insufficient (first request or during strictUntil period after
// a denial), it fetches the current counts from Redis to ensure accuracy.
//
// The strictUntil mechanism forces a Redis lookup for a full window duration after
// any rate limit is exceeded. This prevents over-admission during the decay period
// of the sliding window when relying only on stale local data.
func (s *service) handleBucket(ctx context.Context, req RatelimitRequest) (RatelimitResponse, error) {

	key := bucketKey{req.Identifier, req.Limit, req.Duration}

	b, _ := s.getOrCreateBucket(key)

	b.mu.Lock()
	defer b.mu.Unlock()
	currentWindow, currentWindowExisted := b.getCurrentWindow(req.Time)
	previousWindow, previousWindowExisted := b.getPreviousWindow(req.Time)

	decisionSource := "local"
	// First, try to make a decision based only on local data
	if currentWindowExisted && previousWindowExisted {
		// Check if we can reject based on local data alone
		//
		exceeded, effectiveCount, remaining := s.calculateRateLimit(req, currentWindow, previousWindow)
		if exceeded {
			b.strictUntil = req.Time.Add(req.Duration)

			metrics.RatelimitDecision.WithLabelValues(decisionSource, "denied").Inc()

		}
		return RatelimitResponse{
			Success:   !exceeded,
			Remaining: remaining,
			Reset:     currentWindow.start.Add(currentWindow.duration),
			Limit:     req.Limit,
			Current:   effectiveCount,
		}, nil

	}

	// If we couldn't make a local rejection decision, proceed with Redis checks if needed
	goToOrigin := req.Time.UnixMilli() < b.strictUntil.UnixMilli()
	if goToOrigin || !currentWindowExisted {
		decisionSource = "origin"
		currentKey := counterKey(key, currentWindow.sequence)
		res, err := s.counter.Get(ctx, currentKey)
		if err != nil {
			s.logger.Error("unable to get counter value",
				"key", currentKey,
				"error", err.Error(),
			)
		} else {
			currentWindow.counter = max(currentWindow.counter, res)
		}
	}

	if goToOrigin || !previousWindowExisted {
		decisionSource = "origin"
		previousKey := counterKey(key, previousWindow.sequence)
		res, err := s.counter.Get(ctx, previousKey)
		if err != nil {
			s.logger.Error("unable to get counter value",
				"key", previousKey,
				"error", err.Error(),
			)
		} else {
			previousWindow.counter = max(previousWindow.counter, res)
		}
	}

	// Now check again with potentially updated data from Redis
	exceeded, effectiveCount, remaining := s.calculateRateLimit(req, currentWindow, previousWindow)

	if exceeded {
		// Set strictUntil to prevent further requests
		b.strictUntil = req.Time.Add(req.Duration)

		metrics.RatelimitDecision.WithLabelValues(decisionSource, "denied").Inc()
		return RatelimitResponse{
			Success:   false,
			Remaining: remaining,
			Reset:     currentWindow.start.Add(currentWindow.duration),
			Limit:     req.Limit,
			Current:   effectiveCount,
		}, nil
	}

	metrics.RatelimitDecision.WithLabelValues(decisionSource, "allowed").Inc()
	return RatelimitResponse{
		Success:   true,
		Remaining: remaining,
		Reset:     currentWindow.start.Add(currentWindow.duration),
		Limit:     req.Limit,
		Current:   currentWindow.counter,
	}, nil

}
