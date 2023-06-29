package database

import (
	"context"
	"fmt"

	"github.com/chronark/unkey/apps/api/pkg/database/models"
	"github.com/chronark/unkey/apps/api/pkg/entities"
)

func (db *Database) GetKeyByHash(ctx context.Context, hash string) (entities.Key, error) {

	found, err := models.KeyByHash(ctx, db.read(), hash)
	if err != nil {
		return entities.Key{}, fmt.Errorf("unable to load key by hash %s from db: %w", hash, err)
	}
	if found == nil {
		return entities.Key{}, fmt.Errorf("unable to find key by hash %s in db", hash)
	}

	return keyModelToEntity(found)

}
