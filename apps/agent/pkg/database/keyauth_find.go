package database

import (
	"context"
	"database/sql"
	"fmt"

	"errors"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
)

func (db *database) FindKeyAuth(ctx context.Context, keyauthId string) (*authenticationv1.KeyAuth, bool, error) {

	model, err := db.read().FindKeyAuth(ctx, keyauthId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("unable to find key: %w", err)
	}

	api, err := transformKeyAuthModelToEntity(model)
	if err != nil {
		return nil, true, fmt.Errorf("unable to transform keyauth model to entity: %w", err)
	}
	return api, true, nil
}

func transformKeyAuthModelToEntity(m gen.KeyAuth) (*authenticationv1.KeyAuth, error) {
	return &authenticationv1.KeyAuth{
		KeyAuthId:   m.ID,
		WorkspaceId: m.WorkspaceID,
	}, nil

}
