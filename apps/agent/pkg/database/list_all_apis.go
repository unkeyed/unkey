package database

import (
	"context"
	"database/sql"
	"fmt"

	"errors"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
)

func (db *database) ListAllApis(ctx context.Context, offset, limit int) ([]*apisv1.Api, error) {

	models, err := db.read().ListAllApis(ctx, gen.ListAllApisParams{
		Limit:  int32(offset),
		Offset: int32(limit),
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("unable to find apis: %w", err)
	}

	apis := make([]*apisv1.Api, len(models))
	for i, model := range models {
		apis[i], err = transformApiModelToEntity(model)
		if err != nil {
			return nil, fmt.Errorf("unable to transform model to key: %+v: %w", model, err)
		}
	}

	return apis, nil
}
