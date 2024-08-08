package ratelimit

import (
	"context"
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

func (s *service) Ratelimit(ctx context.Context, req *ratelimitv1.RatelimitRequest) (*ratelimitv1.RatelimitResponse, error) {

	ratelimitReq := ratelimit.RatelimitRequest{
		Name:       req.Name,
		Identifier: req.Identifier,
		Limit:      req.Limit,
		Duration:   time.Duration(req.Duration) * time.Millisecond,
		Cost:       req.Cost,
	}
	if req.Lease != nil {
		ratelimitReq.Lease = &ratelimit.Lease{
			Cost:      req.Lease.Cost,
			ExpiresAt: time.Now().Add(time.Duration(req.Lease.Timeout) * time.Millisecond),
		}
	}
	taken := s.ratelimiter.Take(ctx, ratelimitReq)

	if s.batcher != nil {
		_, span := tracing.Start(ctx, "emitting pushPull event")
		span.SetAttributes(attribute.Int("channelSize", s.batcher.Size()))
		s.batcher.Buffer(&ratelimitv1.PushPullEvent{
			Identifier: req.Identifier,
			Limit:      req.Limit,
			Duration:   req.Duration,
			Cost:       req.Cost,
			Time:       time.Now().UnixMilli(),
			Pass:       taken.Pass,
			Lease:      req.Lease,
		})

		span.End()

	}

	res := &ratelimitv1.RatelimitResponse{
		Current:   int64(taken.Current),
		Limit:     int64(taken.Limit),
		Remaining: int64(taken.Remaining),
		Reset_:    taken.Reset,
		Success:   taken.Pass,
	}

	if req.Lease != nil {
		res.Lease = &ratelimitv1.Lease{
			Identifier: req.Identifier,
			Limit:      req.Limit,
			Duration:   req.Duration,
		}
	}

	return res, nil

}
