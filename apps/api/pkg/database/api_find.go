package database

import (
	"context"
	"database/sql"
	"fmt"

	"errors"

	"github.com/unkeyed/unkey/apps/api/pkg/entities"
)

func (db *database) FindApi(ctx context.Context, apiId string) (entities.Api, bool, error) {

	model, err := db.readReplica.query.FindApi(ctx, apiId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.Api{}, false, nil
		}
		return entities.Api{}, false, fmt.Errorf("unable to find api: %w", err)
	}

	api, err := transformApiModelToEntity(model)
	if err != nil {
		return entities.Api{}, true, fmt.Errorf("unable to transform api model to entity: %w", err)
	}
	return api, true, nil
}
