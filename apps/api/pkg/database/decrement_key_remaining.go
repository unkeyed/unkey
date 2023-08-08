package database

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/apps/api/pkg/entities"
)

func (db *database) DecrementRemainingKeyUsage(ctx context.Context, keyId string) (entities.Key, error) {

	tx, err := db.primary.db.BeginTx(ctx, nil)
	if err != nil {
		return entities.Key{}, fmt.Errorf("uanble to begin transaction: %w", err)
	}
	q := db.primary.query.WithTx(tx)

	err = q.DecrementKeyRemaining(ctx, keyId)
	if err != nil {
		return entities.Key{}, fmt.Errorf("unable to decrement remaining: %w", err)
	}

	res, err := q.FindKeyById(ctx, keyId)
	if err != nil {
		return entities.Key{}, fmt.Errorf("unable to find key: %w", err)
	}
	return transformKeyModelToEntity(res)
}
