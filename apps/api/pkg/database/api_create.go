package database

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/apps/api/pkg/entities"
)

func (db *database) CreateApi(ctx context.Context, newApi entities.Api) error {

	api := apiEntityToModel(newApi)

	err := api.Insert(ctx, db.write())
	if err != nil {
		return fmt.Errorf("unable to insert key, %w", err)
	}
	return nil
}
