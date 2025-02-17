package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"errors"

	"github.com/unkeyed/unkey/go/pkg/database/transform"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) FindKeyByHash(ctx context.Context, hash string) (entities.Key, error) {

	model, err := db.read().FindKeyByHash(ctx, hash)
	if err != nil {
		db.logger.Error(ctx, "found key by hash", slog.Any("model", model), slog.Any("error", err))

		if errors.Is(err, sql.ErrNoRows) {
			return entities.Key{}, fault.Wrap(err,
				fault.WithTag(fault.NOT_FOUND),
				fault.WithDesc("not found", fmt.Sprintf("The key %s does not exist.", hash)),
			)
		}
		return entities.Key{}, fault.Wrap(err, fault.WithTag(fault.DATABASE_ERROR))
	}

	key, err := transform.KeyModelToEntity(model)
	if err != nil {
		return entities.Key{}, fault.Wrap(err, fault.WithDesc("cannot transform key model to entity", ""))
	}

	return key, nil
}
