package usagelimiter

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
)

func (s *service) Limit(ctx context.Context, req UsageRequest) (UsageResponse, error) {
	ctx, span := tracing.Start(ctx, "usagelimiter.Limit")
	defer span.End()

	limit, err := db.WithRetryContext(ctx, func() (sql.NullInt32, error) {
		return db.Query.FindKeyCredits(ctx, s.db.RO(), req.KeyID)
	})
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
		s.metrics.RecordDecision("db", "denied")
		return UsageResponse{Valid: false, Remaining: 0}, nil
	}

	err = db.Query.UpdateKeyCreditsDecrement(ctx, s.db.RW(), db.UpdateKeyCreditsDecrementParams{
		ID:      req.KeyID,
		Credits: sql.NullInt32{Int32: req.Cost, Valid: true},
	})
	if err != nil {
		return UsageResponse{}, err
	}

	s.metrics.RecordDecision("db", "allowed")
	return UsageResponse{Valid: true, Remaining: max(0, remaining-req.Cost)}, nil
}

func (s *service) Close() error {
	// Direct DB service has no resources to clean up
	return nil
}
