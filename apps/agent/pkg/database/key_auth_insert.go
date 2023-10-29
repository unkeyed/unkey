package database

import (
	"context"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
)

func (db *database) InsertKeyAuth(ctx context.Context, keyAuth *authenticationv1.KeyAuth) error {

	return db.write().InsertKeyAuth(ctx, gen.InsertKeyAuthParams{
		ID:          keyAuth.KeyAuthId,
		WorkspaceID: keyAuth.WorkspaceId,
	},
	)
}
