package ratelimit

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	ratelimitv1 "github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/go/pkg/tracing"
)

// consumes the replay buffer and sends out replay requests to peers
//
// This is blocking and should be called in a goroutine
func (s *service) replayRequests() {
	for req := range s.replayBuffer.Consume() {
		s.replayToOrigin(req)
	}

}

func (s *service) replayToOrigin(req *ratelimitv1.ReplayRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx, span := tracing.Start(ctx, "ratelimit.replay")
	defer span.End()

	now := s.clock.Now()

	key := bucketKey{
		req.GetRequest().GetIdentifier(),
		req.GetRequest().GetLimit(),
		time.Duration(req.GetRequest().GetDuration()) * time.Millisecond,
	}.toString()
	p, err := s.getPeer(key)
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Warn(ctx, "unable to create peer client", slog.String("error", err.Error()), slog.String("key", key))
		return
	}
	if p.node.ID == s.cluster.Self().ID {
		// we're the origin, nothing to replay...
		return
	}

	res, err := s.replayCircuitBreaker.Do(ctx, func(innerCtx context.Context) (*connect.Response[ratelimitv1.ReplayResponse], error) {
		innerCtx, cancel = context.WithTimeout(innerCtx, 10*time.Second)
		defer cancel()
		return p.client.Replay(innerCtx, connect.NewRequest(req))
	})
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Warn(ctx, "unable to replay request", slog.String("error", err.Error()), slog.String("key", key))
		return
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

		s.logger.Error(ctx, "unable to set windows", slog.String("error", err.Error()), slog.String("key", key))
		return
	}
	// if we got this far, we pushpulled successfully with a peer and don't need to try the rest

}
