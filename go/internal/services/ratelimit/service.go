package ratelimit

import (
	"context"
	"sync"
	"time"

	"connectrpc.com/connect"
	ratelimitv1 "github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/cluster"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/metrics"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
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

	// cluster manages node discovery and state distribution
	cluster cluster.Cluster

	// shutdownCh signals service shutdown
	shutdownCh chan struct{}

	// bucketsMu protects access to the buckets map
	bucketsMu sync.RWMutex

	// buckets maps identifier+sequence to rate limit buckets
	// Protected by bucketsMu
	buckets map[string]*bucket

	// peerMu protects access to peer-related fields
	peerMu sync.RWMutex

	// peers maps node IDs to peer connections
	// Protected by peerMu
	peers map[string]peer

	// replayBuffer holds rate limit events for async propagation
	// Thread-safe internally
	replayBuffer *buffer.Buffer[*ratelimitv1.ReplayRequest]

	// replayCircuitBreaker prevents cascading failures during peer communication
	// Thread-safe internally
	replayCircuitBreaker circuitbreaker.CircuitBreaker[*connect.Response[ratelimitv1.ReplayResponse]]
}

var _ ratelimitv1connect.RatelimitServiceHandler = (*service)(nil)

type Config struct {
	Logger  logging.Logger
	Cluster cluster.Cluster
	Clock   clock.Clock
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
		clock:                config.Clock,
		logger:               config.Logger,
		cluster:              config.Cluster,
		shutdownCh:           make(chan struct{}),
		bucketsMu:            sync.RWMutex{},
		buckets:              make(map[string]*bucket),
		peerMu:               sync.RWMutex{},
		peers:                make(map[string]peer),
		replayBuffer:         buffer.New[*ratelimitv1.ReplayRequest](10_000, true),
		replayCircuitBreaker: circuitbreaker.New[*connect.Response[ratelimitv1.ReplayResponse]]("replayRatelimitRequest"),
	}

	go s.syncPeers()
	s.expireWindowsAndBuckets()

	// start multiple goroutines to do replays
	for range 8 {
		go s.replayRequests()
	}

	return s, nil
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
//   - May return errors from cluster communication
//
// Performance:
//   - O(1) time complexity for local decisions
//   - Network round trip only when syncing with origin
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
func (r *service) Ratelimit(ctx context.Context, req RatelimitRequest) (RatelimitResponse, error) {
	_, span := tracing.Start(ctx, "Ratelimit")
	defer span.End()

	err := assert.All(
		assert.NotEmpty(req.Identifier, "ratelimit identifier must not be empty"),
		assert.Greater(req.Limit, 0, "ratelimit limit must be greater than zero"),
		assert.GreaterOrEqual(req.Cost, 0, "ratelimit cost must not be negative"),
		assert.GreaterOrEqual(req.Duration.Milliseconds(), 1000, "ratelimit duration must be at least 1s"),
	)
	if err != nil {
		return RatelimitResponse{}, err
	}

	now := r.clock.Now()

	localRes, goToOrigin, err := r.localRatelimit(ctx, now, req)
	if err != nil {
		return RatelimitResponse{}, err
	}

	replayRequest := &ratelimitv1.ReplayRequest{
		Request: &ratelimitv1.RatelimitRequest{
			Identifier: req.Identifier,
			Limit:      req.Limit,
			Duration:   req.Duration.Milliseconds(),
			Cost:       req.Cost,
		},
		Time:   now.UnixMilli(),
		Denied: !localRes.Success,
	}

	if goToOrigin {
		metrics.Ratelimit.OriginDecisions.Add(ctx, 1)
		originRes, err := r.syncWithOrigin(ctx, replayRequest)
		if err != nil {
			r.logger.Error("unable to ask the origin",
				"error", err.Error(),
				"identifier", req.Identifier,
			)
		} else if originRes != nil {
			return RatelimitResponse{
				Limit:          originRes.GetResponse().GetLimit(),
				Remaining:      originRes.GetResponse().GetRemaining(),
				Reset:          originRes.GetResponse().GetReset_(),
				Success:        originRes.GetResponse().GetSuccess(),
				Current:        originRes.GetResponse().GetCurrent(),
				CurrentWindow:  originRes.GetCurrent(),
				PreviousWindow: originRes.GetPrevious(),
			}, nil
		}
	} else {
		r.replayBuffer.Buffer(replayRequest)
		metrics.Ratelimit.LocalDecisions.Add(ctx, 1)

	}

	return localRes, nil
}

// localRatelimit performs a rate limit check using only local state.
// It implements the core sliding window algorithm and determines if
// synchronization with the origin node is needed.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - now: Current time (from service clock)
//   - req: The rate limit request
//
// Returns:
//   - RatelimitResponse: The local rate limit decision
//   - bool: True if sync with origin is needed
//   - error: Any errors during processing
//
// Algorithm:
//  1. Gets or creates local rate limit bucket
//  2. Calculates effective count using sliding window
//  3. Updates local state if request is allowed
//  4. Determines if origin sync is needed
//
// Thread Safety:
//   - Protected by bucket mutex
//   - Safe for concurrent calls
//
// Performance: O(1) time complexity
func (r *service) localRatelimit(ctx context.Context, now time.Time, req RatelimitRequest) (RatelimitResponse, bool, error) {
	_, span := tracing.Start(ctx, "localRatelimit")
	defer span.End()

	key := bucketKey{req.Identifier, req.Limit, req.Duration}
	span.SetAttributes(attribute.String("key", key.toString()))

	b, _ := r.getOrCreateBucket(key)

	b.mu.Lock()
	defer b.mu.Unlock()

	goToOrigin := now.UnixMilli() < b.strictUntil.UnixMilli()
	// Get current and previous windows
	currentWindow, currentWindowExisted := b.getCurrentWindow(now)
	previousWindow, _ := b.getPreviousWindow(now)

	if !currentWindowExisted {
		goToOrigin = true
	}

	// Calculate time elapsed in current window (as a fraction)
	windowElapsed := float64(now.UnixMilli()-currentWindow.GetStart()) / float64(req.Duration.Milliseconds())

	// Pure sliding window calculation:
	// - We count 100% of current window
	// - We count a decreasing portion of previous window based on how far we are into current window
	effectiveCount := currentWindow.GetCounter() + int64(float64(previousWindow.GetCounter())*(1.0-windowElapsed))

	effectiveCount += req.Cost

	// Check if this request would exceed the limit
	if effectiveCount > req.Limit {
		remaining := req.Limit - effectiveCount
		if remaining < 0 {
			remaining = 0
		}

		b.strictUntil = now.Add(req.Duration)

		span.SetAttributes(attribute.Bool("passed", false))
		return RatelimitResponse{
			Success:        false,
			Remaining:      remaining,
			Reset:          currentWindow.GetStart() + currentWindow.GetDuration(),
			Limit:          req.Limit,
			Current:        effectiveCount,
			CurrentWindow:  currentWindow,
			PreviousWindow: previousWindow,
		}, goToOrigin, nil
	}

	// Increment current window counter
	currentWindow.Counter += req.Cost

	remaining := req.Limit - effectiveCount
	if remaining < 0 {
		remaining = 0
	}
	span.SetAttributes(attribute.Bool("passed", true))

	return RatelimitResponse{
		Success:        true,
		Remaining:      remaining,
		Reset:          currentWindow.GetStart() + currentWindow.GetDuration(),
		Limit:          req.Limit,
		Current:        currentWindow.GetCounter(),
		CurrentWindow:  currentWindow,
		PreviousWindow: previousWindow,
	}, goToOrigin, nil
}
