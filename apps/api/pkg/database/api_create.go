package database

import (
	"context"
	"fmt"

	"github.com/chronark/unkey/apps/api/pkg/entities"
)

func (db *Database) CreateApi(ctx context.Context, newApi entities.Api) error {

	api := apiEntityToModel(newApi)

	err := api.Insert(ctx, db.write())
	if err != nil {
		return fmt.Errorf("unable to insert key, %w", err)
	}
	return nil
}
