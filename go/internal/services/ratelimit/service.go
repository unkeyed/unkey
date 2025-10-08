package ratelimit

import (
	"context"
	"sort"
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

// Type to track request with its key and index
type reqWithKey struct {
	req   RatelimitRequest
	key   bucketKey
	index int
}

type bucketInfo struct {
	bucket *bucket
	key    bucketKey
}

// RatelimitMany checks multiple rate limits atomically.
//
// All rate limit checks must pass for the request to be allowed. If any limit fails,
// none of the counters are incremented. This all-or-nothing behavior prevents counter
// leaks when a key has multiple rate limits (e.g., per-minute and per-month).
//
// The method acquires locks on all unique buckets (sorted to prevent deadlock) and
// holds them while checking limits and incrementing counters. This ensures no race
// conditions occur between check and increment.
//
// Returns validation errors for invalid request parameters (empty identifier, zero limit,
// negative cost, or duration less than 1 second).
func (s *service) RatelimitMany(ctx context.Context, reqs []RatelimitRequest) ([]RatelimitResponse, error) {
	_, span := tracing.Start(ctx, "RatelimitMany")
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

	// Build and sort keys first (before getting buckets)
	reqsWithKeys := make([]reqWithKey, len(reqs))
	for i, req := range reqs {
		key := bucketKey{req.Identifier, req.Limit, req.Duration}
		reqsWithKeys[i] = reqWithKey{
			req:   req,
			key:   key,
			index: i,
		}
	}

	// Sort by key to ensure consistent ordering (prevents deadlock)
	sort.Slice(reqsWithKeys, func(i, j int) bool {
		ki := reqsWithKeys[i].key
		kj := reqsWithKeys[j].key
		// Compare: identifier, then limit, then duration
		if ki.identifier != kj.identifier {
			return ki.identifier < kj.identifier
		}
		if ki.limit != kj.limit {
			return ki.limit < kj.limit
		}
		return ki.duration < kj.duration
	})

	// Get unique buckets in sorted order and deduplicate
	uniqueBuckets := make([]bucketInfo, 0, len(reqs))
	bucketMap := make(map[bucketKey]*bucket)

	for _, rwk := range reqsWithKeys {
		if _, exists := bucketMap[rwk.key]; !exists {
			b, _ := s.getOrCreateBucket(rwk.key)
			bucketMap[rwk.key] = b
			uniqueBuckets = append(uniqueBuckets, bucketInfo{
				bucket: b,
				key:    rwk.key,
			})
		}
	}

	// Acquire locks on unique buckets only (already sorted)
	for _, bi := range uniqueBuckets {
		bi.bucket.mu.Lock()
	}
	// Ensure locks are released on function exit
	defer func() {
		for i := len(uniqueBuckets) - 1; i >= 0; i-- {
			uniqueBuckets[i].bucket.mu.Unlock()
		}
	}()

	// Check all limits while holding locks
	responses := make([]RatelimitResponse, len(reqs))
	allPassed := true

	for _, rwk := range reqsWithKeys {
		bucket := bucketMap[rwk.key]

		// Check limit with lock already held
		res, err := s.checkBucketWithLockHeld(ctx, rwk.req, bucket, rwk.key)
		if err != nil {
			return nil, err
		}
		responses[rwk.index] = res

		if !res.Success {
			allPassed = false
			// Don't break - check all limits to return complete status
		}
	}

	// If all passed, increment all counters (still holding locks!)
	if allPassed {
		span.SetAttributes(attribute.Bool("passed", true))
		for _, rwk := range reqsWithKeys {
			bucket := bucketMap[rwk.key]
			currentWindow, _ := bucket.getCurrentWindow(rwk.req.Time)
			currentWindow.counter += rwk.req.Cost

			// Buffer for async replay to Redis
			s.replayBuffer.Buffer(rwk.req)
		}
	} else {
		span.SetAttributes(attribute.Bool("denied", true))

		// At least one failed - adjust remaining values
		for i := range responses {
			responses[i].Remaining += reqs[i].Cost
		}
	}

	// Clamp all remaining values
	for i := range responses {
		responses[i].Remaining = max(0, responses[i].Remaining)
	}

	return responses, nil
}

func (s *service) Ratelimit(ctx context.Context, req RatelimitRequest) (RatelimitResponse, error) {
	_, span := tracing.Start(ctx, "Ratelimit")
	defer span.End()

	if req.Time.IsZero() {
		req.Time = s.clock.Now()
	}

	err := assert.All(
		assert.NotEmpty(req.Identifier, "ratelimit identifier must not be empty"),
		assert.Greater(req.Limit, 0, "ratelimit limit must be greater than zero"),
		assert.GreaterOrEqual(req.Cost, 0, "ratelimit cost must not be negative"),
		assert.GreaterOrEqual(req.Duration.Milliseconds(), 1000, "ratelimit duration must be at least 1s"),
		assert.False(req.Time.IsZero(), "request time must not be zero"),
	)
	if err != nil {
		return RatelimitResponse{}, err
	}

	key := bucketKey{req.Identifier, req.Limit, req.Duration}
	span.SetAttributes(attribute.String("key", key.toString()))

	b, _ := s.getOrCreateBucket(key)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Get current and previous windows
	currentWindow, currentWindowExisted := b.getCurrentWindow(req.Time)
	previousWindow, previousWindowExisted := b.getPreviousWindow(req.Time)

	// track whether we were able to handle the request locally or if we had to call redis
	decisionSource := "local"

	// First, try to make a decision based only on local data
	if currentWindowExisted && previousWindowExisted {
		// Check if we can reject based on local data alone
		exceeded, effectiveCount, remaining := s.calculateRateLimit(req, currentWindow, previousWindow)
		if exceeded {
			b.strictUntil = req.Time.Add(req.Duration)

			// Record the denied request
			span.SetAttributes(attribute.Bool("passed", false))
			metrics.RatelimitDecision.WithLabelValues(decisionSource, "denied").Inc()

			return RatelimitResponse{
				Success:   false,
				Remaining: remaining,
				Reset:     currentWindow.start.Add(currentWindow.duration),
				Limit:     req.Limit,
				Current:   effectiveCount,
			}, nil
		}
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

		span.SetAttributes(attribute.Bool("passed", false))
		metrics.RatelimitDecision.WithLabelValues(decisionSource, "denied").Inc()

		return RatelimitResponse{
			Success:   false,
			Remaining: remaining,
			Reset:     currentWindow.start.Add(currentWindow.duration),
			Limit:     req.Limit,
			Current:   effectiveCount,
		}, nil
	}

	// If we get here, the request is allowed
	// Increment current window counter
	currentWindow.counter += req.Cost

	// Buffer the request for async propagation
	s.replayBuffer.Buffer(req)

	span.SetAttributes(attribute.Bool("passed", true))
	metrics.RatelimitDecision.WithLabelValues(decisionSource, "passed").Inc()

	return RatelimitResponse{
		Success:   true,
		Remaining: remaining,
		Reset:     currentWindow.start.Add(currentWindow.duration),
		Limit:     req.Limit,
		Current:   currentWindow.counter,
	}, nil
}

// checkBucketWithLockHeld evaluates a rate limit request with the bucket lock already held.
// This is used by RatelimitMany to check limits while holding all locks.
// The caller MUST hold bucket.mu.Lock() before calling this.
func (s *service) checkBucketWithLockHeld(ctx context.Context, req RatelimitRequest, b *bucket, key bucketKey) (RatelimitResponse, error) {
	currentWindow, currentWindowExisted := b.getCurrentWindow(req.Time)
	previousWindow, previousWindowExisted := b.getPreviousWindow(req.Time)

	decisionSource := "local"

	// First, try to make a decision based only on local data
	if currentWindowExisted && previousWindowExisted {
		exceeded, effectiveCount, remaining := s.calculateRateLimit(req, currentWindow, previousWindow)

		if exceeded {
			b.strictUntil = req.Time.Add(req.Duration)
			metrics.RatelimitDecision.WithLabelValues(decisionSource, "denied").Inc()
		} else {
			metrics.RatelimitDecision.WithLabelValues(decisionSource, "allowed").Inc()
		}

		return RatelimitResponse{
			Success:   !exceeded,
			Remaining: remaining,
			Reset:     currentWindow.start.Add(currentWindow.duration),
			Limit:     req.Limit,
			Current:   effectiveCount,
		}, nil
	}

	// If we couldn't make a local decision, proceed with Redis checks if needed
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
