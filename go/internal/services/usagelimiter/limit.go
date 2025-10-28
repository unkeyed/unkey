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

	// Determine which credit system to use
	// Priority: CreditID (new system) > KeyID (legacy system)
	useCreditSystem := req.CreditID != ""
	useLegacySystem := req.KeyID != "" && req.CreditID == ""

	if useCreditSystem {
		// New credits table system
		remaining, err := db.WithRetryContext(ctx, func() (int32, error) {
			return db.Query.FindRemainingCredits(ctx, s.db.RO(), req.CreditID)
		})
		if err != nil {
			if db.IsNotFound(err) {
				return UsageResponse{Valid: false, Remaining: 0}, nil
			}
			return UsageResponse{Valid: false, Remaining: 0}, err
		}

		// Credit doesn't have enough to cover the request cost
		if req.Cost > 0 && remaining < req.Cost {
			metrics.UsagelimiterDecisions.WithLabelValues("db", "denied").Inc()
			return UsageResponse{Valid: false, Remaining: 0}, nil
		}

		err = db.Query.UpdateCreditDecrement(ctx, s.db.RW(), db.UpdateCreditDecrementParams{
			ID:      req.CreditID,
			Credits: req.Cost,
		})
		if err != nil {
			return UsageResponse{}, err
		}

		metrics.UsagelimiterDecisions.WithLabelValues("db", "allowed").Inc()
		return UsageResponse{Valid: true, Remaining: max(0, remaining-req.Cost)}, nil
	} else if useLegacySystem {
		// Legacy keys.remaining_requests system
		remainingReq, err := db.WithRetryContext(ctx, func() (sql.NullInt32, error) {
			return db.Query.FindRemainingKey(ctx, s.db.RO(), req.KeyID)
		})
		if err != nil {
			if db.IsNotFound(err) {
				return UsageResponse{Valid: false, Remaining: 0}, nil
			}
			return UsageResponse{Valid: false, Remaining: 0}, err
		}

		if !remainingReq.Valid {
			return UsageResponse{Valid: true, Remaining: 0}, nil
		}

		remaining := remainingReq.Int32

		// Key doesn't have enough remaining to cover the request cost
		if req.Cost > 0 && remaining < req.Cost {
			metrics.UsagelimiterDecisions.WithLabelValues("db", "denied").Inc()
			return UsageResponse{Valid: false, Remaining: 0}, nil
		}

		err = db.Query.UpdateKeyCreditsDecrement(ctx, s.db.RW(), db.UpdateKeyCreditsDecrementParams{
			ID:      req.KeyID,
			Credits: sql.NullInt32{Int32: req.Cost, Valid: true},
		})
		if err != nil {
			return UsageResponse{}, err
		}

		metrics.UsagelimiterDecisions.WithLabelValues("db", "allowed").Inc()
		return UsageResponse{Valid: true, Remaining: max(0, remaining-req.Cost)}, nil

	} else {
		// Neither CreditID nor KeyID provided
		return UsageResponse{Valid: false, Remaining: 0}, nil
	}
}

func (s *service) Close() error {
	// Direct DB service has no resources to clean up
	return nil
}
