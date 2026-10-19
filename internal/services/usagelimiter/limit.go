package usagelimiter

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/internal/services/usagelimiter/metrics"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
)

func (s *service) Limit(ctx context.Context, req UsageRequest) (UsageResponse, error) {
	ctx, span := tracing.Start(ctx, "usagelimiter.Limit")
	defer span.End()

	if err := assert.GreaterOrEqual(req.Cost, 0, "usagelimiter cost must not be negative"); err != nil {
		return UsageResponse{}, err
	}

	response, err := db.TxWithResult(ctx, s.db, func(ctx context.Context, tx db.DBTX) (UsageResponse, error) {
		if _, err := db.Query.LockKeyForUpdate(ctx, tx, req.KeyID); err != nil {
			return UsageResponse{}, err
		}
		remaining, err := db.Query.FindKeyCredits(ctx, tx, req.KeyID)
		if err != nil {
			return UsageResponse{}, err
		}

		if !remaining.Valid {
			return UsageResponse{Valid: true, Remaining: -1}, nil
		}
		if req.Cost > 0 && remaining.Int64 < req.Cost {
			return UsageResponse{Valid: false, Remaining: 0}, nil
		}

		if err := db.Query.UpdateKeyCreditsDecrement(ctx, tx, db.UpdateKeyCreditsDecrementParams{
			ID:      req.KeyID,
			Credits: sql.NullInt64{Int64: req.Cost, Valid: true},
		}); err != nil {
			return UsageResponse{}, err
		}
		return UsageResponse{Valid: true, Remaining: max(0, remaining.Int64-req.Cost)}, nil
	})

	if err != nil {
		return UsageResponse{}, err
	}
	if response.Remaining >= 0 {
		decision := "denied"
		if response.Valid {
			decision = "allowed"
		}
		metrics.UsagelimiterDecisions.WithLabelValues("db", decision).Inc()
	}
	return response, nil
}

func (s *service) Close() error {
	return nil
}
