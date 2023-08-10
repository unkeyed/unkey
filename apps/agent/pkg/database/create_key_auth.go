package database

import (
	"context"
	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
)

func (db *database) CreateKeyAuth(ctx context.Context, keyAuth entities.KeyAuth) error {
	return db.write().CreateKeyAuth(ctx, gen.CreateKeyAuthParams{
		ID:          keyAuth.Id,
		WorkspaceID: keyAuth.WorkspaceId,
	})

}
