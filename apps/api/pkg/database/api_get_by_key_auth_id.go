package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/unkeyed/unkey/apps/api/pkg/database/models"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
)

func (db *database) GetApiByKeyAuthId(ctx context.Context, keyAuthId string) (entities.Api, error) {

	api, err := models.APIByKeyAuthID(ctx, db.read(), sql.NullString{String: keyAuthId, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.Api{}, ErrNotFound
		}
		return entities.Api{}, fmt.Errorf("unable to load api by keyA %s from db: %w", keyAuthId, err)
	}
	if api == nil {
		return entities.Api{}, ErrNotFound
	}
	return apiModelToEntity(api), nil
}
