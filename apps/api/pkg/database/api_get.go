package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/chronark/unkey/apps/api/pkg/database/models"
	"github.com/chronark/unkey/apps/api/pkg/entities"
)

func (db *Database) GetApi(ctx context.Context, apiId string) (entities.Api, error) {
	ctx, span := db.tracer.Start(ctx, "db.getApi")
	defer span.End()
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
	return entities.Api{
		Id:          api.ID,
		Name:        api.Name,
		WorkspaceId: api.WorkspaceID,
	}, nil
}
