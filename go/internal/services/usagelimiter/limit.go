package usagelimiter

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
)

func (s *service) Limit(ctx context.Context, req UsageRequest) (UsageResponse, error) {
	ctx, span := tracing.Start(ctx, "usagelimiter.Limit")
	defer span.End()

	limit, err := db.Query.FindKeyCredits(ctx, s.db.RW(), req.KeyId)
	if err != nil {
		if db.IsNotFound(err) {
			return UsageResponse{Valid: false, Remaining: 0}, nil
		}

		return UsageResponse{Valid: false, Remaining: 0}, err
	}

	if !limit.Valid {
		return UsageResponse{Valid: true, Remaining: -1}, nil
	}

	remaining := limit.Int32
	// Key doesn't have enough credits to cover the request cost
	if req.Cost > 0 && remaining < req.Cost {
		metrics.UsagelimiterDecisions.WithLabelValues("db", "denied").Inc()
		return UsageResponse{Valid: false, Remaining: 0}, nil
	}

	err = db.Query.UpdateKeyCreditsDecrement(ctx, s.db.RW(), db.UpdateKeyCreditsDecrementParams{
		ID:      req.KeyId,
		Credits: sql.NullInt32{Int32: req.Cost, Valid: true},
	})
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
