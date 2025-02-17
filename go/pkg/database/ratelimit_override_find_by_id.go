package database

import (
	"context"
	"database/sql"
	"fmt"

	"errors"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/database/transform"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) FindRatelimitOverrideByID(ctx context.Context, workspaceId, overrideID string) (entities.RatelimitOverride, error) {

	model, err := db.read().FindRatelimitOverridesById(ctx, gen.FindRatelimitOverridesByIdParams{
		WorkspaceID: workspaceId,
		OverrideID:  overrideID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.RatelimitOverride{}, fault.Wrap(err,
				fault.WithTag(fault.NOT_FOUND),
				fault.WithDesc("not found", fmt.Sprintf("Ratelimit override '%s' does not exist.", overrideID)),
			)
		}
		return entities.RatelimitOverride{}, fault.Wrap(err, fault.WithTag(fault.DATABASE_ERROR))
	}

	override, err := transform.RatelimitOverrideModelToEntity(model)
	if err != nil {
		return entities.RatelimitOverride{}, fault.Wrap(err,
			fault.WithDesc("cannot transform override model to entity", ""))
	}
	return override, nil
}
