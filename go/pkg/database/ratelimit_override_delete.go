package database

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) DeleteRatelimitOverride(ctx context.Context, id string) error {
	result, err := db.write().DeleteRatelimitOverride(ctx, gen.DeleteRatelimitOverrideParams{
		ID: id,
		Now: sql.NullTime{
			Time:  db.clock.Now(),
			Valid: true,
		},
	})
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("failed to delete ratelimit override", ""))
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
