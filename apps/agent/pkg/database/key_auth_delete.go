package database

import (
	"context"
)

func (db *database) DeleteKeyAuth(ctx context.Context, keyAuthId string) error {
	return db.write().DeleteKeyAuth(ctx, keyAuthId)
}
