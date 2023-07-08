package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/unkeyed/unkey/apps/api/pkg/database/models"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
)

func (db *database) GetApi(ctx context.Context, apiId string) (entities.Api, error) {

	api, err := models.APIByID(ctx, db.read(), apiId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.Api{}, ErrNotFound
		}
		return entities.Api{}, fmt.Errorf("unable to load api %s from db: %w", apiId, err)
	}
	if api == nil {
		return entities.Api{}, ErrNotFound
	}
	return apiModelToEntity(api), nil
}
