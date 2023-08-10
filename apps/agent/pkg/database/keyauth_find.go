package database

import (
	"context"
	"database/sql"
	"fmt"

	"errors"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
)

func (db *database) FindKeyAuth(ctx context.Context, keyauthId string) (entities.KeyAuth, bool, error) {

	model, err := db.read().FindKeyAuth(ctx, keyauthId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.KeyAuth{}, false, nil
		}
		return entities.KeyAuth{}, false, fmt.Errorf("unable to find key: %w", err)
	}

	api, err := transformKeyAuthModelToEntity(model)
	if err != nil {
		return entities.KeyAuth{}, true, fmt.Errorf("unable to transform keyauth model to entity: %w", err)
	}
	return api, true, nil
}

func transformKeyAuthModelToEntity(m gen.KeyAuth) (entities.KeyAuth, error) {
	return entities.KeyAuth{
		Id:          m.ID,
		WorkspaceId: m.WorkspaceID,
	}, nil

}
