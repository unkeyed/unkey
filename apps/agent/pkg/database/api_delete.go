package database

import (
	"context"
)

func (db *database) DeleteApi(ctx context.Context, apiId string) error {
	return db.write().DeleteApi(ctx, apiId)
}
