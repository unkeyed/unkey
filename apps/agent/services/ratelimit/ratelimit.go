package ratelimit

import (
	"context"
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

func (s *service) Ratelimit(ctx context.Context, req *ratelimitv1.RatelimitRequest) (*ratelimitv1.RatelimitResponse, error) {
	start := time.Now()
	defer func() {
		s.logger.Info().
			Int64("latency", time.Since(start).Milliseconds()).
			Msg("service.Ratelimit")
	}()
	_, span := tracing.Start(ctx, "Ratelimit")
	defer span.End()
	res := s.ratelimiter.Take(ratelimit.RatelimitRequest{
		Identifier:     req.Identifier,
		Max:            req.Limit,
		RefillRate:     req.Limit,
		RefillInterval: req.Duration,
		Cost:           req.Cost,
	})
	s.logger.Info().Interface("req", req).Interface("res", res).Msg("ratelimit")

	if s.pushPullC != nil {
		e := pushPullEvent{
			identifier: req.Identifier,
			limit:      req.Limit,
			duration:   req.Duration,
			cost:       req.Cost,
		}
		s.logger.Info().Interface("event", e).Msg("queueing pushPull with origin")
		s.pushPullC <- e

	}

	return &ratelimitv1.RatelimitResponse{
		Limit:     int64(res.Limit),
		Remaining: int64(res.Remaining),
		Reset_:    res.Reset,
		Success:   res.Pass,
	}, nil

}
