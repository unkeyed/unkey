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

// service implements the ratelimit.Service interface using a sliding window algorithm
// with distributed state management across a cluster of nodes.
//
// Architecture:
//   - Uses consistent hashing to assign rate limit buckets to origin nodes
//   - Makes fast local decisions when possible to minimize latency
//   - Propagates state changes asynchronously to maintain eventual consistency
//   - Handles cluster topology changes automatically
//
// Thread Safety:
//   - All public methods are safe for concurrent use
//   - Internal state is protected by appropriate mutex locks
//   - Cluster state updates are handled atomically
//
// Performance Characteristics:
//   - O(1) time complexity for rate limit checks
//   - Local decisions avoid network round trips
//   - Asynchronous state propagation minimizes overhead
//   - Automatic cleanup of expired windows reduces memory usage
//
// Limitations:
//   - Brief periods of over-admission possible during node failures
//   - State propagation adds some eventual consistency delay
//   - Memory usage scales with number of active rate limit buckets
//
// Example Usage:
//
//	svc, err := ratelimit.New(ratelimit.Config{
//	    Logger:  logger,
//	    Cluster: cluster,
//	    Clock:   clock,
//	})
//	if err != nil {
//	    return err
//	}
//	defer svc.Close()
//
//	resp, err := svc.Ratelimit(ctx, RatelimitRequest{
//	    Identifier: "user-123",
//	    Limit:      100,
//	    Duration:   time.Minute,
//	})
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

type Config struct {
	Logger logging.Logger

	Clock clock.Clock

	// If provided, use this counter implementation instead of creating a Redis counter
	Counter counter.Counter
}

// New creates a new rate limiting service with the given configuration.
//
// The service starts background goroutines for:
//   - Synchronizing state with peer nodes
//   - Cleaning up expired rate limit windows
//   - Processing replay buffer events
//
// These goroutines run until the service is shut down.
//
// Parameters:
//   - config: Required configuration including logger and cluster info
//
// Returns:
//   - *service: The initialized rate limiting service
//   - error: Configuration validation or initialization errors
//
// Thread Safety:
//   - Safe to call from any goroutine
//   - Returned service is safe for concurrent use
//
// Example Usage:
//
//	svc, err := ratelimit.New(ratelimit.Config{
//	    Logger:  logger,
//	    Cluster: cluster,
//	})
//	if err != nil {
//	    return fmt.Errorf("failed to create rate limiter: %w", err)
//	}
//	defer svc.Close()
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

// Close releases all resources held by the rate limiter.
// It should be called when the service is no longer needed.
func (s *service) Close() error {
	s.replayBuffer.Close()
	return nil
}

// calculateRateLimit evaluates if a request is within rate limits and returns the calculation results
// This helper function abstracts the rate limit calculation logic
func (s *service) calculateRateLimit(req RatelimitRequest, currentWindow, previousWindow *window) (bool, int64, int64) {
	// Calculate time elapsed in current window (as a fraction)
	windowElapsed := float64(req.Time.Sub(currentWindow.start).Milliseconds()) / float64(req.Duration.Milliseconds())

	// Pure sliding window calculation:
	// - We count 100% of current window
	// - We count a decreasing portion of previous window based on how far we are into current window
	effectiveCount := currentWindow.counter + int64(float64(previousWindow.counter)*(1.0-windowElapsed))

	effectiveCount += req.Cost

	// Calculate remaining (could be negative)
	remaining := req.Limit - effectiveCount
	if remaining < 0 {
		remaining = 0
	}

	// Check if this request would exceed the limit
	exceeded := effectiveCount > req.Limit

	return exceeded, effectiveCount, remaining
}

// Ratelimit checks if a request should be allowed under current rate limit constraints.
// It implements a sliding window algorithm that considers both the current and previous
// time windows to provide accurate rate limiting across a cluster of nodes.
//
// The method follows these steps:
// 1. Validates request parameters
// 2. Makes a local rate limit decision
// 3. Synchronizes with origin node if needed
// 4. Updates local state based on response
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - req: The rate limit request parameters
//
// Returns:
//   - RatelimitResponse: Contains success/failure and current limit state
//   - error: Validation or system errors
//
// Errors:
//   - Returns validation errors for invalid parameters
//
// Thread Safety:
//   - Safe for concurrent use
//   - State updates are atomic
//
// Example:
//
//	resp, err := svc.Ratelimit(ctx, RatelimitRequest{
//	    Identifier: "user-123",
//	    Limit:      100,
//	    Duration:   time.Minute,
//	})
//	if err != nil {
//	    return err
//	}
//	if !resp.Success {
//	    return fmt.Errorf("rate limit exceeded, retry after %v",
//	        time.UnixMilli(resp.Reset))
//	}
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
