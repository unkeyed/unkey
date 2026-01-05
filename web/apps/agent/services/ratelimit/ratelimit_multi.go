package ratelimit

import (
	"context"
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/svc/agent/gen/proto/ratelimit/v1"
)

func (s *service) MultiRatelimit(ctx context.Context, req *ratelimitv1.RatelimitMultiRequest) (*ratelimitv1.RatelimitMultiResponse, error) {

	responses := make([]*ratelimitv1.RatelimitResponse, len(req.Ratelimits))
	for i, r := range req.Ratelimits {
		res := s.Take(ctx, ratelimitRequest{
			Identifier: r.Identifier,
			Limit:      r.Limit,
			Duration:   time.Duration(r.Duration) * time.Millisecond,
			Cost:       r.Cost,
		})

		responses[i] = &ratelimitv1.RatelimitResponse{
			Limit:     res.Limit,
			Remaining: res.Remaining,
			Reset_:    res.Reset,
			Success:   res.Pass,
			Current:   res.Current,
		}

	}

	return &ratelimitv1.RatelimitMultiResponse{
		Ratelimits: responses}, nil

}
