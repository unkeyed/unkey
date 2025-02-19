package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/database/transform"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) FindRatelimitNamespaceByName(ctx context.Context, workspaceID string, name string) (entities.RatelimitNamespace, error) {
	model, err := db.read().FindRatelimitNamespaceByName(ctx, gen.FindRatelimitNamespaceByNameParams{
		WorkspaceID: workspaceID,
		Name:        name,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.RatelimitNamespace{}, fault.Wrap(err,
				fault.WithTag(fault.NOT_FOUND),
				fault.WithDesc("not found", fmt.Sprintf("Ratelimit namespace '%s' does not exist in workspace %s.", name, workspaceID)),
			)
		}
		return entities.RatelimitNamespace{}, fault.Wrap(err, fault.WithTag(fault.DATABASE_ERROR))
	}

	namespace, err := transform.RatelimitNamespaceModelToEntity(model)
	if err != nil {
		return entities.RatelimitNamespace{}, fault.Wrap(err,
			fault.WithDesc("cannot transform namespace model to entity", ""))
	}
	return namespace, nil
}
