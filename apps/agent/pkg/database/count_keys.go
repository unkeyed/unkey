package database

import (
	"context"
	"database/sql"
)

func (db *database) CountKeys(ctx context.Context, keyAuthId string) (int64, error) {
	return db.read().CountKeysByKeyAuth(ctx, sql.NullString{String: keyAuthId, Valid: true})

}
