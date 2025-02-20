package ratelimit

import (
	"context"
	"math"
	"sync"
	"time"

	"connectrpc.com/connect"
	ratelimitv1 "github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/cluster"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/"github.com/unkeyed/unkey/go/pkg/otel""
	"go.opentelemetry.io/otel/attribute"
)

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

	currentWindow := b.getCurrentWindow(now)
	previousWindow := b.getPreviousWindow(now)
	currentWindowPercentage := float64(now.UnixMilli()-currentWindow.GetStart()) / float64(req.Duration)
	previousWindowPercentage := 1.0 - currentWindowPercentage

	// Calculate the current count including all leases
	fromPreviousWindow := float64(previousWindow.GetCounter()) * previousWindowPercentage
	fromCurrentWindow := float64(currentWindow.GetCounter())

	current := int64(math.Ceil(fromCurrentWindow + fromPreviousWindow))

	// Evaluate if the request should pass or not

	if current+req.Cost > req.Limit {

		remaining := req.Limit - current
		if remaining < 0 {
			remaining = 0
		}
		return RatelimitResponse{
			Success:   false,
			Remaining: remaining,
			Reset:     currentWindow.GetStart() + currentWindow.GetDuration(),
			Limit:     req.Limit,
			Current:   current,
		}, nil
	}

	currentWindow.Counter += req.Cost

	current += req.Cost

	remaining := req.Limit - current
	if remaining < 0 {
		remaining = 0
	}
	return RatelimitResponse{
		Success:   true,
		Remaining: remaining,
		Reset:     currentWindow.GetStart() + currentWindow.GetDuration(),
		Limit:     req.Limit,
		Current:   current,
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
