package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/unkeyed/unkey/apps/api/pkg/database/models"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
)

func (db *database) GetKeyAuth(ctx context.Context, keyAuthId string) (entities.KeyAuth, error) {

	keyAuth, err := models.KeyAuthByID(ctx, db.read(), keyAuthId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.KeyAuth{}, ErrNotFound
		}
		return entities.KeyAuth{}, fmt.Errorf("unable to load keyAuth %s from db: %w", keyAuthId, err)
	}
	if keyAuth == nil {
		return entities.KeyAuth{}, ErrNotFound
	}
	return keyAuthModelToEntity(keyAuth), nil
}
