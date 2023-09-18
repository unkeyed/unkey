package database

import (
	"context"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
)

func (db *database) InsertKeyAuth(ctx context.Context, keyAuth entities.KeyAuth) error {

	return db.write().InsertKeyAuth(ctx, gen.InsertKeyAuthParams{
		ID:          keyAuth.Id,
		WorkspaceID: keyAuth.WorkspaceId,
	},
	)
}
