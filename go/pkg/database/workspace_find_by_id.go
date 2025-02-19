package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/unkeyed/unkey/go/pkg/database/transform"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) FindWorkspaceByID(ctx context.Context, id string) (entities.Workspace, error) {
	model, err := db.read().FindWorkspaceByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.Workspace{}, fault.Wrap(err,
				fault.WithTag(fault.NOT_FOUND),
				fault.WithDesc("not found", fmt.Sprintf("Workspace with ID %s does not exist.", id)),
			)
		}
		return entities.Workspace{}, fault.Wrap(err, fault.WithTag(fault.DATABASE_ERROR))
	}

	workspace, err := transform.WorkspaceModelToEntity(model)
	if err != nil {
		return entities.Workspace{}, fault.Wrap(err,
			fault.WithDesc("cannot transform workspace model to entity", ""))
	}
	return workspace, nil
}
