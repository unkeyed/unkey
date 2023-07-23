package database

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/apps/api/pkg/entities"
)

func (db *database) CreateKeyAuth(ctx context.Context, newKeyAuth entities.KeyAuth) error {

	keyAuth := keyAuthEntityToModel(newKeyAuth)

	err := keyAuth.Insert(ctx, db.write())
	if err != nil {
		return fmt.Errorf("unable to insert keyAuth, %w", err)
	}
	return nil
}
