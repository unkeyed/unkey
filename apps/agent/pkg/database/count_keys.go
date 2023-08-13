package database

import (
	"context"
)

func (db *database) CountKeys(ctx context.Context, keyAuthId string) (int64, error) {
	return db.read().CountKeysByKeyAuth(ctx, keyAuthId)

}
