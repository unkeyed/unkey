package database

import (
	"context"
)

func (db *database) CountKeys(ctx context.Context, keyAuthId string) (int64, error) {
	return db.readReplica.query.CountKeysByKeyAuth(ctx, keyAuthId)

}
