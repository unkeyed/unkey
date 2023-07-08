package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/unkeyed/unkey/apps/api/pkg/database/models"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
)

func (db *database) GetKeyById(ctx context.Context, keyId string) (entities.Key, error) {
	found, err := models.KeyByID(ctx, db.read(), keyId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.Key{}, ErrNotFound
		}
		return entities.Key{}, fmt.Errorf("unable to load key by keyId %s from db: %w", keyId, err)
	}
	if found == nil {
		return entities.Key{}, ErrNotFound
	}

	return keyModelToEntity(found)

}
