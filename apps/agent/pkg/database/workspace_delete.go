package database

import (
	"context"
)

func (db *database) DeleteWorkspace(ctx context.Context, workspaceId string) error {
	return db.write().DeleteWorkspace(ctx, workspaceId)
}
