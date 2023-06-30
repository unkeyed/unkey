package database

import (
	"context"
	"fmt"

	"github.com/chronark/unkey/apps/api/pkg/entities"
)

func (db *database) CreateWorkspace(ctx context.Context, newWorkspace entities.Workspace) error {
	workpspace := workspaceEntityToModel(newWorkspace)

	err := workpspace.Insert(ctx, db.write())
	if err != nil {
		return fmt.Errorf("unable to insert key, %w", err)
	}
	return nil
}
