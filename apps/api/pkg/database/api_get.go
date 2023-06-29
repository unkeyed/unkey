package database

import (
	"context"
	"fmt"

	"github.com/chronark/unkey/apps/api/pkg/database/models"
	"github.com/chronark/unkey/apps/api/pkg/entities"
)

func (db *Database) GetApi(ctx context.Context, apiId string) (entities.Api, error) {

	api, err := models.APIByID(ctx, db.read(), apiId)
	if err != nil {
		return entities.Api{}, fmt.Errorf("unable to load api %s from db: %w", apiId, err)
	}
	if api == nil {
		return entities.Api{}, fmt.Errorf("unable to find api %s in db", apiId)
	}
	return entities.Api{
		Id:          api.ID,
		Name:        api.Name,
		WorkspaceId: api.WorkspaceID,
	}, nil
}
