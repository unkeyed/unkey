package ratelimit

import (
	"context"
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
)

func (s *service) PushPull(ctx context.Context, req *ratelimitv1.PushPullRequest) (*ratelimitv1.PushPullResponse, error) {

	r := s.Take(ctx, ratelimitRequest{
		Time:       time.UnixMilli(req.Time),
		Name:       req.Request.Name,
		Identifier: req.Request.Identifier,
		Limit:      req.Request.Limit,
		Duration:   time.Duration(req.Request.Duration) * time.Millisecond,
		Cost:       req.Request.Cost,
	})

	return &ratelimitv1.PushPullResponse{
		Response: &ratelimitv1.RatelimitResponse{
			Current:   int64(r.Current),
			Limit:     int64(r.Limit),
			Remaining: int64(r.Remaining),
			Reset_:    r.Reset,
			Success:   r.Pass,
		},

		Current:  r.currentWindow,
		Previous: r.previousWindow,
	}, nil

}
