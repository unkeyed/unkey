package ratelimit

import (
	"context"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
)

func (s *service) Ratelimit(ctx context.Context, req *ratelimitv1.RatelimitRequest) (*ratelimitv1.RatelimitResponse, error) {

	res := s.ratelimiter.Take(ratelimit.RatelimitRequest{
		Identifier:     req.Identifier,
		Max:            int32(req.Limit),
		RefillRate:     int32(req.Limit),
		RefillInterval: int32(req.Duration),
		Cost:           int32(req.Cost),
	})
	s.logger.Info().Interface("req", req).Interface("res", res).Msg("ratelimit")

	return &ratelimitv1.RatelimitResponse{
		Limit:     int64(res.Limit),
		Remaining: int64(res.Remaining),
		Reset_:    res.Reset,
		Success:   res.Pass,
	}, nil

}
