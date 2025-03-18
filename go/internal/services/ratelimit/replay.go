package ratelimit

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ratelimitv1 "github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// consumes the replay buffer and sends out replay requests to peers
//
// This is blocking and should be called in a goroutine
func (s *service) replayRequests() {
	for req := range s.replayBuffer.Consume() {
		s.replayToOrigin(context.Background(), req)
	}

}

func (s *service) replayToOrigin(ctx context.Context, req *ratelimitv1.ReplayRequest) {
	ctx, span := tracing.Start(context.Background(), "replayToOrigin")
	defer span.End()

	now := s.clock.Now()

	res, err := s.askOrigin(ctx, req)

	err = s.SetWindows(ctx,
		setWindowRequest{
			Identifier: req.GetRequest().GetIdentifier(),
			Limit:      req.GetRequest().GetLimit(),
			Counter:    res.GetCurrent().GetCounter(),
			Sequence:   res.GetCurrent().GetSequence(),
			Duration:   time.Duration(req.GetRequest().GetDuration()) * time.Millisecond,
			Time:       now,
		},
		setWindowRequest{
			Identifier: req.GetRequest().GetIdentifier(),
			Limit:      req.GetRequest().GetLimit(),
			Counter:    res.GetPrevious().GetCounter(),
			Sequence:   res.GetPrevious().GetSequence(),
			Duration:   time.Duration(req.GetRequest().GetDuration()) * time.Millisecond,
			Time:       now,
		},
	)

	if err != nil {
		tracing.RecordError(span, err)

		s.logger.Error("unable to set windows",
			"error", err.Error(),
		)
		return
	}
	// if we got this far, we pushpulled successfully with a peer and don't need to try the rest

}

func (s *service) askOrigin(ctx context.Context, req *ratelimitv1.ReplayRequest) (*ratelimitv1.ReplayResponse, error) {
	ctx, span := tracing.Start(ctx, "askOrigin")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	now := s.clock.Now()

	key := bucketKey{
		req.Request.GetIdentifier(),
		req.Request.GetLimit(),
		time.Duration(req.Request.GetDuration()) * time.Millisecond,
	}.toString()
	p, err := s.getPeer(ctx, key)
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Warn("unable to create peer client",
			"error", err.Error(),
			"key", key,
		)
		return nil, err
	}
	span.SetAttributes(attribute.String("originInstanceID", p.instance.ID))

	if p.instance.ID == s.cluster.Self().ID {
		// we're the origin, nothing to replay...
		return nil, nil
	}

	res, err := s.replayCircuitBreaker.Do(ctx, func(innerCtx context.Context) (*connect.Response[ratelimitv1.ReplayResponse], error) {
		innerCtx, cancel = context.WithTimeout(innerCtx, 10*time.Second)
		defer cancel()
		return p.client.Replay(innerCtx, connect.NewRequest(req))
	})
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Warn("unable to replay request",
			"peer", p.instance,
			"error", err.Error(),
			"key", key,
		)
		return nil, err
	}

	err = s.SetWindows(ctx,
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

	if err != nil {
		tracing.RecordError(span, err)

		s.logger.Error("unable to set windows",
			"error", err.Error(),
			"key", key,
		)
		return nil, err
	}
	// if we got this far, we pushpulled successfully with a peer and don't need to try the rest
	return res.Msg, nil
}

func (s *service) Replay(ctx context.Context, req *connect.Request[ratelimitv1.ReplayRequest]) (*connect.Response[ratelimitv1.ReplayResponse], error) {
	ctx, span := tracing.Start(ctx, "Replay")
	defer span.End()
	t := time.UnixMilli(req.Msg.GetTime())

	res, err := s.ratelimit(ctx, t, RatelimitRequest{
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
