package database

import (
	"context"
	"database/sql"

	"errors"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/database/transform"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) FindRatelimitOverridesByIdentifier(ctx context.Context, workspaceId, namespaceId, identifier string) ([]entities.RatelimitOverride, error) {

	models, err := db.read().FindRatelimitOverridesByIdentifier(ctx, gen.FindRatelimitOverridesByIdentifierParams{
		WorkspaceID: workspaceId,
		NamespaceID: namespaceId,
		Identifier:  identifier,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []entities.RatelimitOverride{}, nil
		}
		return []entities.RatelimitOverride{}, fault.Wrap(err, fault.WithTag(fault.DATABASE_ERROR))
	}

	es := make([]entities.RatelimitOverride, len(models))
	for i := 0; i < len(models); i++ {

		es[i], err = transform.RatelimitOverrideModelToEntity(models[i])
		if err != nil {
			return []entities.RatelimitOverride{}, fault.Wrap(err, fault.WithDesc("cannot transform model to entity", ""))
		}
	}
	return es, nil
}
