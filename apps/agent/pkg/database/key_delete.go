package database

import (
	"context"
	"database/sql"
	"time"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
)

func (db *database) DeleteKey(ctx context.Context, keyId string) error {
	return db.write().MarkKeyDeleted(ctx, gen.MarkKeyDeletedParams{
		KeyID: keyId,
		Now: sql.NullTime{
			Valid: true,
			Time:  time.Now(),
		},
	})
}
