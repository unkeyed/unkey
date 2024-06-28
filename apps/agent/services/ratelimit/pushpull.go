package ratelimit

import (
	"context"
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
)

func (s *service) PushPull(ctx context.Context, req *ratelimitv1.PushPullRequest) (*ratelimitv1.PushPullResponse, error) {
	start := time.Now()
	defer func() {
		s.logger.Info().
			Int64("latency", time.Since(start).Milliseconds()).
			Msg("service.PushPull")
	}()
	res := s.ratelimiter.Take(ctx, ratelimit.RatelimitRequest{
		Identifier:     req.Identifier,
		Max:            req.Limit,
		RefillRate:     req.Limit,
		RefillInterval: req.Duration,
		Cost:           req.Cost,
	})
	s.logger.Debug().Interface("req", req).Interface("res", res).Msg("ratelimit")

	return &ratelimitv1.PushPullResponse{
		Identifier: req.Identifier,
		Current:    res.Current,
	}, nil

}
