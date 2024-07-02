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
	start := time.Now()
	defer func() {
		s.logger.Info().
			Int64("latency", time.Since(start).Milliseconds()).
			Msg("service.Ratelimit")
	}()
	res := s.ratelimiter.Take(ctx, ratelimit.RatelimitRequest{
		Identifier:     req.Identifier,
		Max:            req.Limit,
		RefillRate:     req.Limit,
		RefillInterval: req.Duration,
		Cost:           req.Cost,
	})
	s.logger.Info().Interface("req", req).Interface("res", res).Msg("ratelimit")

	if s.batcher != nil {
		_, span := tracing.Start(ctx, "emitting pushPull event")
		span.SetAttributes(attribute.Int("channelSize", s.batcher.Size()))
		s.batcher.Buffer(&ratelimitv1.PushPullEvent{
			Identifier: req.Identifier,
			Limit:      req.Limit,
			Duration:   req.Duration,
			Cost:       req.Cost,
		})

		span.End()

	}

	return &ratelimitv1.RatelimitResponse{
		Limit:     int64(res.Limit),
		Remaining: int64(res.Remaining),
		Reset_:    res.Reset,
		Success:   res.Pass,
	}, nil

}
