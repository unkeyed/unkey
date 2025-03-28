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

var _ ratelimitv1connect.RatelimitServiceHandler = (*service)(nil)

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

	localRes, goToOrigin, err := r.ratelimit(ctx, now, req)
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
		r.logger.Info("windows are new. syncing with origin",
			"identifier", req.Identifier,
		)
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

	}

	return localRes, nil
}
func (r *service) ratelimit(ctx context.Context, now time.Time, req RatelimitRequest) (RatelimitResponse, bool, error) {
	_, span := tracing.Start(ctx, "ratelimit")
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

	// Check if this request would exceed the limit
	if effectiveCount+req.Cost > req.Limit {
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

	// Recalculate effective count with updated current window
	effectiveCount = currentWindow.GetCounter() + int64(float64(previousWindow.GetCounter())*(1.0-windowElapsed))

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
		Current:        effectiveCount,
		CurrentWindow:  currentWindow,
		PreviousWindow: previousWindow,
	}, goToOrigin, nil
}
