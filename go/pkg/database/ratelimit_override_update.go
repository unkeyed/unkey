package database

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) UpdateRatelimitOverride(ctx context.Context, e entities.RatelimitOverride) error {
	params := gen.UpdateRatelimitOverrideParams{
		ID:          e.ID,
		Windowlimit: e.Limit,
		Now: sql.NullTime{
			Time:  db.clock.Now(),
			Valid: true,
		},

		Duration: int32(e.Duration.Milliseconds()), // nolint:gosec
		Async: sql.NullBool{
			Bool:  e.Async,
			Valid: true,
		},
	}

	result, err := db.write().UpdateRatelimitOverride(ctx, params)
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("failed to update ratelimit override", ""))
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("failed to get rows affected", ""))
	}

	if rows == 0 {
		return fault.Wrap(sql.ErrNoRows,
			fault.WithTag(fault.NOT_FOUND),
			fault.WithDesc("ratelimit override not found", ""),
		)
	}

	return nil
}
