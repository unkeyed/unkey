package database

import (
	"context"
)

func (db *database) DeleteKey(ctx context.Context, keyId string) error {
	return db.primary.query.DeleteKey(ctx, keyId)
}
