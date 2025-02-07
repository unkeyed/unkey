package ratelimit

import (
	"context"
	"math"
	"sync"

	"connectrpc.com/connect"
	ratelimitv1 "github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/cluster"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type service struct {
	clock clock.Clock

	logger  logging.Logger
	cluster cluster.Cluster

	shutdownCh chan struct{}

	bucketsLock sync.RWMutex
	// identifier+sequence -> bucket
	buckets map[string]*bucket

	peerMu sync.RWMutex
	peers  map[string]ratelimitv1connect.RatelimitServiceClient

	replayCircuitBreaker circuitbreaker.CircuitBreaker[*connect.Response[ratelimitv1.ReplayResponse]]
}

func (r *service) Ratelimit(ctx context.Context, req RatelimitRequest) RatelimitResponse {
	ctx, span := tracing.Start(ctx, "slidingWindow.Ratelimit")
	defer span.End()

	now := r.clock.Now()

	key := bucketKey{req.Identifier, req.Limit, req.Duration}
	span.SetAttributes(attribute.String("key", string(key.toString())))

	bucket, _ := r.getBucket(key)

	bucket.Lock()
	defer bucket.Unlock()

	currentWindow := bucket.getCurrentWindow(now)
	previousWindow := bucket.getPreviousWindow(now)
	currentWindowPercentage := float64(now.UnixMilli()-currentWindow.Start) / float64(req.Duration)
	previousWindowPercentage := 1.0 - currentWindowPercentage

	// Calculate the current count including all leases
	fromPreviousWindow := float64(previousWindow.Counter) * previousWindowPercentage
	fromCurrentWindow := float64(currentWindow.Counter)

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
			Reset:     currentWindow.Start + currentWindow.Duration,
			Limit:     req.Limit,
			Current:   current,
		}
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
		Reset:     currentWindow.Start + currentWindow.Duration,
		Limit:     req.Limit,
		Current:   current,
	}
}
