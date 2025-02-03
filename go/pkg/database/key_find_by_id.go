package database

import (
	"context"
	"database/sql"
	"fmt"

	"errors"

	"github.com/unkeyed/unkey/go/pkg/database/transform"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) FindKeyByID(ctx context.Context, keyID string) (entities.Key, error) {

	model, err := db.read().FindKeyByID(ctx, keyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.Key{}, fault.Wrap(err,
				fault.WithTag(fault.NOT_FOUND),
				fault.WithDesc("not found", fmt.Sprintf("The key %s does not exist.", keyID)),
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
