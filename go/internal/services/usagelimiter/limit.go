package usagelimiter

import (
	"context"
	"database/sql"
	"math"

	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
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
	if remaining <= 0 && req.Cost != 0 || remaining-req.Cost < 0 {
		return UsageResponse{Valid: false, Remaining: 0}, nil
	}

	err = db.Query.UpdateKeyCredits(ctx, s.db.RW(), db.UpdateKeyCreditsParams{
		ID:        req.KeyId,
		Operation: "decrement",
		Credits:   sql.NullInt32{Int32: req.Cost, Valid: true},
	})
	if err != nil {
		return UsageResponse{}, err
	}

	return UsageResponse{Valid: true, Remaining: int32(math.Max(float64(0), float64(remaining-req.Cost)))}, nil
}
