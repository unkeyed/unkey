package ratelimit

import (
	"context"
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

func (s *service) MultiRatelimit(ctx context.Context, req *ratelimitv1.RatelimitMultiRequest) (*ratelimitv1.RatelimitMultiResponse, error) {
	start := time.Now()
	defer func() {
		s.logger.Info().
			Int64("latency", time.Since(start).Milliseconds()).
			Msg("service.Ratelimit")
	}()

	responses := make([]*ratelimitv1.RatelimitResponse, len(req.Ratelimits))
	for i, r := range req.Ratelimits {
		res := s.ratelimiter.Take(ctx, ratelimit.RatelimitRequest{
			Identifier:     r.Identifier,
			Max:            r.Limit,
			RefillRate:     r.Limit,
			RefillInterval: r.Duration,
			Cost:           r.Cost,
		})

		responses[i] = &ratelimitv1.RatelimitResponse{
			Limit:     res.Limit,
			Remaining: res.Remaining,
			Reset_:    res.Reset,
			Success:   res.Pass,
			Current:   res.Current,
		}

		if s.batcher != nil {
			_, span := tracing.Start(ctx, "emitting pushPull event")
			span.SetAttributes(attribute.Int("channelSize", s.batcher.Size()))
			s.batcher.Buffer(&ratelimitv1.PushPullEvent{
				Identifier: r.Identifier,
				Limit:      r.Limit,
				Duration:   r.Duration,
				Cost:       r.Cost,
			})
			span.End()

		}
	}

	return &ratelimitv1.RatelimitMultiResponse{
		Ratelimits: responses}, nil

}
