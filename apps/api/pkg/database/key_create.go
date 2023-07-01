package database

import (
	"context"
	"fmt"

	"github.com/chronark/unkey/apps/api/pkg/entities"
)

func (db *database) CreateKey(ctx context.Context, newKey entities.Key) error {
	key, err := keyEntityToModel(newKey)
	if err != nil {
		return fmt.Errorf("uanble to convert key")
	}

	err = key.Insert(ctx, db.write())
	if err != nil {
		return fmt.Errorf("unable to insert key, %w", err)
	}
	return nil
}
