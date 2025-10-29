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
	if req.CreditID != "" {
		return s.limitWithCredits(ctx, req)
	} else if req.KeyID != "" {
		return s.limitWithLegacyKey(ctx, req)
	}

	// Neither CreditID nor KeyID provided
	return UsageResponse{Valid: false, Remaining: 0}, nil
}

// limitWithCredits handles limiting using the new credits table system
func (s *service) limitWithCredits(ctx context.Context, req UsageRequest) (UsageResponse, error) {
	remaining, err := db.WithRetryContext(ctx, func() (int32, error) {
		return db.Query.FindRemainingCredits(ctx, s.db.RO(), req.CreditID)
	})
	if err != nil {
		if db.IsNotFound(err) {
			return UsageResponse{Valid: false, Remaining: 0}, nil
		}

		return UsageResponse{Valid: false, Remaining: 0}, err
	}

	return s.processLimit(ctx, remaining, req.Cost, func() error {
		return db.Query.UpdateCreditDecrement(ctx, s.db.RW(), db.UpdateCreditDecrementParams{
			ID:      req.CreditID,
			Credits: req.Cost,
		})
	})
}

// limitWithLegacyKey handles limiting using the legacy keys.remaining_requests system
func (s *service) limitWithLegacyKey(ctx context.Context, req UsageRequest) (UsageResponse, error) {
	remainingReq, err := db.WithRetryContext(ctx, func() (sql.NullInt32, error) {
		return db.Query.FindRemainingKey(ctx, s.db.RO(), req.KeyID)
	})
	if err != nil {
		if db.IsNotFound(err) {
			return UsageResponse{Valid: false, Remaining: 0}, nil
		}

		return UsageResponse{Valid: false, Remaining: 0}, err
	}

	// If remaining is not set, allow unlimited usage
	if !remainingReq.Valid {
		return UsageResponse{Valid: true, Remaining: 0}, nil
	}

	return s.processLimit(ctx, remainingReq.Int32, req.Cost, func() error {
		return db.Query.UpdateKeyCreditsDecrement(ctx, s.db.RW(), db.UpdateKeyCreditsDecrementParams{
			ID:      req.KeyID,
			Credits: sql.NullInt32{Int32: req.Cost, Valid: true},
		})
	})
}

// processLimit contains the shared logic for checking and decrementing credits
func (s *service) processLimit(ctx context.Context, remaining int32, cost int32, decrementFn func() error) (UsageResponse, error) {
	// Check if enough remaining to cover the request cost
	if cost > 0 && remaining < cost {
		metrics.UsagelimiterDecisions.WithLabelValues("db", "denied").Inc()
		return UsageResponse{Valid: false, Remaining: 0}, nil
	}

	// Decrement the credits/remaining
	if err := decrementFn(); err != nil {
		return UsageResponse{}, err
	}

	metrics.UsagelimiterDecisions.WithLabelValues("db", "allowed").Inc()
	return UsageResponse{Valid: true, Remaining: max(0, remaining-cost)}, nil
}

func (s *service) Close() error {
	// Direct DB service has no resources to clean up
	return nil
}
