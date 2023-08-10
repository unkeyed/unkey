package database

import (
	"context"
	"database/sql"
	"fmt"

	"errors"

	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
)

func (db *database) FindKeyByHash(ctx context.Context, hash string) (entities.Key, bool, error) {

	model, err := db.read().FindKeyByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.Key{}, false, nil
		}
		return entities.Key{}, false, fmt.Errorf("unable to find key: %w", err)
	}

	api, err := transformKeyModelToEntity(model)
	if err != nil {
		return entities.Key{}, true, fmt.Errorf("unable to transform key model to entity: %w", err)
	}
	return api, true, nil
}
