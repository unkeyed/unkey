package database

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) DeleteRatelimitNamespace(ctx context.Context, id string) error {
	result, err := db.write().DeleteRatelimitNamespace(ctx, gen.DeleteRatelimitNamespaceParams{
		ID: id,
		Now: sql.NullTime{
			Time:  db.clock.Now(),
			Valid: true,
		},
	})
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("failed to delete ratelimit namespace", ""))
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("failed to get rows affected", ""))
	}

	if rowsAffected == 0 {
		return fault.Wrap(sql.ErrNoRows,
			fault.WithTag(fault.NOT_FOUND),
			fault.WithDesc("ratelimit namespace not found", ""),
		)
	}

	return nil
}
