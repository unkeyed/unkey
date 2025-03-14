package ratelimit

import (
	"context"
	"sync"
	"time"

	"connectrpc.com/connect"
	ratelimitv1 "github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/cluster"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// service implements the ratelimit.Service interface using a sliding window algorithm.
//
// The implementation distributes rate limit state across the cluster using consistent
// hashing to determine an "origin node" for each client identifier. This approach
// balances the need for accuracy with performance by:
//
// 1. Making local decisions at each node to minimize latency
// 2. Asynchronously propagating state to the origin node to maintain consistency
// 3. Broadcasting limit exceeded events to all nodes to prevent over-admission
//
// The service handles node joins/leaves automatically by rebalancing the consistent
// hash ring, ensuring smooth operation during cluster changes.
type service struct {
	clock clock.Clock

	logger  logging.Logger
	cluster cluster.Cluster

	shutdownCh chan struct{}

	bucketsMu sync.RWMutex
	// identifier+sequence -> bucket
	buckets map[string]*bucket

	peerMu               sync.RWMutex
	peers                map[string]peer
	replayBuffer         *buffer.Buffer[*ratelimitv1.ReplayRequest]
	replayCircuitBreaker circuitbreaker.CircuitBreaker[*connect.Response[ratelimitv1.ReplayResponse]]
}

type Config struct {
	Logger  logging.Logger
	Cluster cluster.Cluster
	Clock   clock.Clock
}

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

	// start multiple goroutines to do replays
	for range 8 {
		go s.replayRequests()
	}

	return s, nil
}

func (r *service) Ratelimit(ctx context.Context, req RatelimitRequest) (RatelimitResponse, error) {
	_, span := tracing.Start(ctx, "slidingWindow.Ratelimit")
	defer span.End()

	err := assert.Multi(
		assert.NotEmpty(req.Identifier, "ratelimit identifier must not be empty"),
		assert.Greater(req.Limit, 0, "ratelimit limit must be greater than zero"),
		assert.GreaterOrEqual(req.Cost, 0, "ratelimit cost must not be negative"),
		assert.GreaterOrEqual(req.Duration.Milliseconds(), 1000, "ratelimit duration must be at least 1s"),
	)
	if err != nil {
		return RatelimitResponse{}, err
	}

	now := r.clock.Now()

	res, err := r.ratelimit(ctx, now, req)
	if err != nil {
		return RatelimitResponse{}, err
	}

	r.replayBuffer.Buffer(&ratelimitv1.ReplayRequest{
		Request: &ratelimitv1.RatelimitRequest{
			Identifier: req.Identifier,
			Limit:      req.Limit,
			Duration:   req.Duration.Milliseconds(),
			Cost:       req.Cost,
		},
		Time:   now.UnixMilli(),
		Denied: !res.Success,
	})
	return res, nil
}
func (r *service) ratelimit(ctx context.Context, now time.Time, req RatelimitRequest) (RatelimitResponse, error) {
	_, span := tracing.Start(ctx, "slidingWindow.ratelimit")
	defer span.End()

	key := bucketKey{req.Identifier, req.Limit, req.Duration}
	span.SetAttributes(attribute.String("key", key.toString()))

	b, _ := r.getOrCreateBucket(key)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Get current and previous windows
	currentWindow := b.getCurrentWindow(now)
	previousWindow := b.getPreviousWindow(now)

	// Calculate time elapsed in current window (as a fraction)
	windowElapsed := float64(now.UnixMilli()-currentWindow.GetStart()) / float64(req.Duration.Milliseconds())

	// Pure sliding window calculation:
	// - We count 100% of current window
	// - We count a decreasing portion of previous window based on how far we are into current window
	effectiveCount := currentWindow.GetCounter() + int64(float64(previousWindow.GetCounter())*(1.0-windowElapsed))

	// Check if this request would exceed the limit
	if effectiveCount+req.Cost > req.Limit {
		remaining := req.Limit - effectiveCount
		if remaining < 0 {
			remaining = 0
		}

		return RatelimitResponse{
			Success:   false,
			Remaining: remaining,
			Reset:     currentWindow.GetStart() + currentWindow.GetDuration(),
			Limit:     req.Limit,
			Current:   effectiveCount,
		}, nil
	}

	// Increment current window counter
	currentWindow.Counter += req.Cost

	// Recalculate effective count with updated current window
	effectiveCount = currentWindow.GetCounter() + int64(float64(previousWindow.GetCounter())*(1.0-windowElapsed))

	remaining := req.Limit - effectiveCount
	if remaining < 0 {
		remaining = 0
	}

	return RatelimitResponse{
		Success:   true,
		Remaining: remaining,
		Reset:     currentWindow.GetStart() + currentWindow.GetDuration(),
		Limit:     req.Limit,
		Current:   effectiveCount,
	}, nil
}

type setWindowRequest struct {
	Identifier string
	Limit      int64
	Duration   time.Duration
	Sequence   int64
	// any time within the window
	Time    time.Time
	Counter int64
}

func (r *service) SetWindows(ctx context.Context, requests ...setWindowRequest) error {
	_, span := tracing.Start(ctx, "slidingWindow.SetWindows")
	defer span.End()
	for _, req := range requests {
		key := bucketKey{req.Identifier, req.Limit, req.Duration}
		bucket, _ := r.getOrCreateBucket(key)
		// Only increment the current value if the new value is greater than the current value
		// Due to varying network latency, we may receive out of order responses and could decrement the
		// current value, which would result in inaccurate rate limiting
		bucket.mu.Lock()
		window, ok := bucket.windows[req.Sequence]
		if !ok {
			window = newWindow(req.Sequence, req.Time, req.Duration)
			bucket.windows[req.Sequence] = window
		}
		if req.Counter > window.GetCounter() {
			window.Counter = req.Counter
		}
		bucket.mu.Unlock()

	}
	return nil
}
