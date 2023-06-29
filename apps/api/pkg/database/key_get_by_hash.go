package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/chronark/unkey/apps/api/pkg/database/models"
	"github.com/chronark/unkey/apps/api/pkg/entities"
)

func (db *Database) GetKeyByHash(ctx context.Context, hash string) (entities.Key, error) {

	found, err := models.KeyByHash(ctx, db.read(), hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.Key{}, ErrNotFound
		}
		return entities.Key{}, fmt.Errorf("unable to load key by hash %s from db: %w", hash, err)
	}
	if found == nil {
		return entities.Key{}, ErrNotFound
	}

	return keyModelToEntity(found)

}
