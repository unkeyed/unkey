package database

import (
	"context"
	"fmt"

	"github.com/chronark/unkey/apps/api/pkg/database/models"
	"github.com/chronark/unkey/apps/api/pkg/entities"
)

func (db *Database) GetKeyById(ctx context.Context, keyId string) (entities.Key, error) {

	found, err := models.KeyByID(ctx, db.read(), keyId)
	if err != nil {
		return entities.Key{}, fmt.Errorf("unable to load key by keyId %s from db: %w", keyId, err)
	}
	if found == nil {
		return entities.Key{}, fmt.Errorf("unable to find key by keyId %s in db", keyId)
	}

	return keyModelToEntity(found)

}
