package database

import (
	"context"
	"database/sql"
	"fmt"

	"errors"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
)

func (db *database) ListKeys(ctx context.Context, keyAuthId string, ownerId string, offset, limit int) ([]entities.Key, error) {
	var (
		models []gen.Key
		err    error
	)
	if ownerId != "" {
		models, err = db.read().ListKeysByKeyAuthIdAndOwnerId(ctx, gen.ListKeysByKeyAuthIdAndOwnerIdParams{
			KeyAuthID: sql.NullString{String: keyAuthId, Valid: true},
			Limit:     int32(offset),
			Offset:    int32(limit),
			OwnerID:   sql.NullString{String: ownerId, Valid: true},
		})
	} else {
		models, err = db.read().ListKeysByKeyAuthId(ctx, gen.ListKeysByKeyAuthIdParams{
			KeyAuthID: sql.NullString{String: keyAuthId, Valid: true},
			Limit:     int32(offset),
			Offset:    int32(limit),
		})
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []entities.Key{}, nil
		}
		return []entities.Key{}, fmt.Errorf("unable to find keys: %w", err)
	}

	keys := make([]entities.Key, len(models))
	for i, model := range models {
		keys[i], err = transformKeyModelToEntity(model)
		if err != nil {
			return []entities.Key{}, fmt.Errorf("unable to transform model to key: %w", err)
		}
	}

	return keys, nil
}
