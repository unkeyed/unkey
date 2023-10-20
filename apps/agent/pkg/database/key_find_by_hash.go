package database

import (
	"context"
	"database/sql"
	"fmt"

	"errors"

	keysv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/keys/v1"
)

func (db *database) FindKeyByHash(ctx context.Context, hash string) (*keysv1.Key, bool, error) {

	model, err := db.read().FindKeyByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("unable to find key: %w", err)
	}

	key, err := transformKeyModelToEntity(model)
	if err != nil {
		return nil, true, fmt.Errorf("unable to transform key model to entity: %w", err)
	}
	return key, true, nil
}
