package database

import (
	"context"
	"fmt"

	"github.com/chronark/unkey/apps/api/pkg/entities"
)

func (db *Database) CreateKey(ctx context.Context, newKey entities.Key) error {
	ctx, span := db.tracer.Start(ctx, "db.createKey")
	defer span.End()
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
