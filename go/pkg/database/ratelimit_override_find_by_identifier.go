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

func (db *database) FindRatelimitOverrideByIdentifier(ctx context.Context, identifier string) (entities.RatelimitOverride, error) {

	model, err := db.read().FindRatelimitOverrideByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.RatelimitOverride{}, fault.Wrap(err,
				fault.WithTag(fault.NOT_FOUND),
				fault.WithDesc("not found", fmt.Sprintf("An override for %s does not exist.", identifier)),
			)
		}
		return entities.RatelimitOverride{}, fault.Wrap(err, fault.WithTag(fault.DATABASE_ERROR))
	}

	e, err := transform.RatelimitOverrideModelToEntity(model)
	if err != nil {
		return entities.RatelimitOverride{}, fault.Wrap(err, fault.WithDesc("cannot transform model to entity", ""))
	}
	return e, nil
}
