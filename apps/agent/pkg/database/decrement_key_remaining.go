package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
)

func (db *database) DecrementRemainingKeyUsage(ctx context.Context, keyId string) (entities.Key, error) {

	tx, err := db.writeReplica.db.BeginTx(ctx, nil)
	if err != nil {
		return entities.Key{}, fmt.Errorf("unable to begin transaction: %w", err)
	}
	defer func() {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			db.logger.Err(err).Msg("unable to rollback transaction")
		}
	}()
	q := db.write().WithTx(tx)

	err = q.DecrementKeyRemaining(ctx, keyId)
	if err != nil {
		return entities.Key{}, fmt.Errorf("unable to decrement remaining: %w", err)
	}

	res, err := q.FindKeyById(ctx, keyId)
	if err != nil {
		return entities.Key{}, fmt.Errorf("unable to find key: %w", err)
	}
	err = tx.Commit()
	if err != nil {
		return entities.Key{}, fmt.Errorf("unable to commit tx: %w", err)
	}
	return transformKeyModelToEntity(res)
}
