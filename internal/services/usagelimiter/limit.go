package usagelimiter

import (
	"context"

	"github.com/unkeyed/unkey/internal/services/usagelimiter/metrics"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
)

func (s *service) Limit(ctx context.Context, req UsageRequest) (UsageResponse, error) {
	ctx, span := tracing.Start(ctx, "usagelimiter.Limit")
	defer span.End()

	remaining, hasLimit, err := s.findKeyCredits(ctx, req.KeyID)
	if err != nil {
		return UsageResponse{Valid: false, Remaining: 0}, err
	}

	if !hasLimit {
		return UsageResponse{Valid: true, Remaining: -1}, nil
	}

	// Key doesn't have enough credits to cover the request cost
	if req.Cost > 0 && remaining < req.Cost {
		metrics.UsagelimiterDecisions.WithLabelValues("db", "denied").Inc()
		return UsageResponse{Valid: false, Remaining: 0}, nil
	}

	err = s.decrementKeyCredits(ctx, req.KeyID, req.Cost)
	if err != nil {
		return UsageResponse{}, err
	}

	metrics.UsagelimiterDecisions.WithLabelValues("db", "allowed").Inc()
	return UsageResponse{Valid: true, Remaining: max(0, remaining-req.Cost)}, nil
}

func (s *service) Close() error {
	// Direct DB service has no resources to clean up
	return nil
}
