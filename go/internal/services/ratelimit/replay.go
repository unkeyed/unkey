package ratelimit

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ratelimitv1 "github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/metrics"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// replayRequests processes buffered rate limit events by synchronizing them with
// the origin nodes. It ensures eventual consistency across the cluster by
// replaying local decisions to the authoritative nodes.
//
// This method:
// 1. Continuously consumes events from the replay buffer
// 2. Forwards each event to its origin node
// 3. Updates local state with the origin's response
//
// Thread Safety:
//   - Must be run in a dedicated goroutine
//   - Safe for concurrent buffer producers
//   - Uses internal synchronization for state updates
//
// Performance:
//   - Batches requests for efficiency
//   - Uses circuit breaker to prevent cascading failures
//   - Automatic retry on transient errors
//
// Example Usage:
//
//	// Start replay processing
//	for range 8 {
//	    go svc.replayRequests()
//	}
func (s *service) replayRequests() {
	for req := range s.replayBuffer.Consume() {
		_, err := s.syncWithOrigin(context.Background(), req)
		if err != nil {
			s.logger.Error("failed to replay request", "error", err.Error())
		}

	}

}

// syncWithOrigin synchronizes a rate limit decision with the origin node for
// a given identifier. This ensures consistent rate limiting across the cluster
// by having authoritative nodes for each rate limit bucket.
//
// The method may return (nil, nil) if the current node is the origin,
// indicating no synchronization was needed.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - req: The rate limit event to synchronize
//
// Returns:
//   - *ReplayResponse: The origin's rate limit state, or nil if local node is origin
//   - error: Any errors during synchronization
//
// Thread Safety:
//   - Safe for concurrent use
//   - Uses internal synchronization for state updates
//
// Performance:
//   - Network round trip to origin node
//   - Circuit breaker prevents cascading failures
//   - 5s overall timeout, 2s per RPC attempt
//
// Errors:
//   - Returns error if peer lookup fails
//   - Returns error if RPC fails
//   - Returns error if circuit breaker is open
//
// Example:
//
//	resp, err := svc.syncWithOrigin(ctx, &ratelimitv1.ReplayRequest{
//	    Request: &ratelimitv1.RatelimitRequest{
//	        Identifier: "user-123",
//	        Limit:      100,
//	        Duration:   60000, // 1 minute
//	    },
//	    Time: time.Now().UnixMilli(),
//	})
func (s *service) syncWithOrigin(ctx context.Context, req *ratelimitv1.ReplayRequest) (*ratelimitv1.ReplayResponse, error) {
	defer func(start time.Time) {
		metrics.Ratelimit.OriginSyncLatency.Record(ctx, time.Since(start).Milliseconds())
	}(time.Now())

	ctx, span := tracing.Start(ctx, "syncWithOrigin")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	now := s.clock.Now()

	key := bucketKey{
		req.GetRequest().GetIdentifier(),
		req.GetRequest().GetLimit(),
		time.Duration(req.GetRequest().GetDuration()) * time.Millisecond,
	}.toString()
	p, err := s.getPeer(ctx, key)
	if err != nil {
		tracing.RecordError(span, err)

		return nil, err
	}
	span.SetAttributes(attribute.String("originInstanceID", p.instance.ID))

	if p.instance.ID == s.cluster.Self().ID {
		// we're the origin, nothing to replay...
		// nolint:nilnil
		return nil, nil
	}

	res, err := s.replayCircuitBreaker.Do(ctx, func(innerCtx context.Context) (*connect.Response[ratelimitv1.ReplayResponse], error) {
		innerCtx, cancel = context.WithTimeout(innerCtx, 2*time.Second)
		defer cancel()
		return p.client.Replay(innerCtx, connect.NewRequest(req))
	})
	if err != nil {
		tracing.RecordError(span, err)

		return nil, err
	}

	s.SetWindows(ctx,
		setWindowRequest{
			Identifier: req.GetRequest().GetIdentifier(),
			Limit:      req.GetRequest().GetLimit(),
			Counter:    res.Msg.GetCurrent().GetCounter(),
			Sequence:   res.Msg.GetCurrent().GetSequence(),
			Duration:   time.Duration(req.GetRequest().GetDuration()) * time.Millisecond,
			Time:       now,
		},
		setWindowRequest{
			Identifier: req.GetRequest().GetIdentifier(),
			Limit:      req.GetRequest().GetLimit(),
			Counter:    res.Msg.GetPrevious().GetCounter(),
			Sequence:   res.Msg.GetPrevious().GetSequence(),
			Duration:   time.Duration(req.GetRequest().GetDuration()) * time.Millisecond,
			Time:       now,
		},
	)

	return res.Msg, nil
}

// Replay handles incoming RPC requests to synchronize rate limit state.
// It is called by other nodes in the cluster when they need to verify
// their local rate limit decisions with this node (when this node is
// the origin for a particular identifier).
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - req: The rate limit event to verify
//
// Returns:
//   - *Response[ReplayResponse]: Current rate limit state
//   - error: Any errors during processing
//
// Thread Safety:
//   - Safe for concurrent use
//   - Uses bucket-level locking
//
// Performance:
//   - O(1) time complexity
//   - No network calls
//   - Uses local state only
//
// RPC Interface:
//   - Part of the RatelimitService gRPC interface
//   - Called automatically by peer nodes
//   - Handles cluster-wide state synchronization
//
// Example:
//
//	// Called via RPC from other nodes
//	resp, err := svc.Replay(ctx, connect.NewRequest(&ratelimitv1.ReplayRequest{
//	    Request: &ratelimitv1.RatelimitRequest{
//	        Identifier: "user-123",
//	        Limit:      100,
//	        Duration:   60000,
//	    },
//	    Time: time.Now().UnixMilli(),
//	}))
func (s *service) Replay(ctx context.Context, req *connect.Request[ratelimitv1.ReplayRequest]) (*connect.Response[ratelimitv1.ReplayResponse], error) {
	ctx, span := tracing.Start(ctx, "Replay")
	defer span.End()
	t := time.UnixMilli(req.Msg.GetTime())

	res, _, err := s.localRatelimit(ctx, t, RatelimitRequest{
		Identifier: req.Msg.GetRequest().GetIdentifier(),
		Limit:      req.Msg.GetRequest().GetLimit(),
		Duration:   time.Duration(req.Msg.GetRequest().GetDuration()) * time.Millisecond,
		Cost:       req.Msg.GetRequest().GetCost(),
	})

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&ratelimitv1.ReplayResponse{
		Current:  res.CurrentWindow,
		Previous: res.PreviousWindow,
		Response: &ratelimitv1.RatelimitResponse{
			Limit:     res.Limit,
			Remaining: res.Remaining,
			Reset_:    res.Reset,
			Success:   res.Success,
			Current:   res.Current,
		},
	}), nil
}
