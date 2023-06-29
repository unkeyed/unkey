package database

import (
	"context"
	"fmt"

	"github.com/chronark/unkey/apps/api/pkg/database/models"
	"github.com/chronark/unkey/apps/api/pkg/entities"
)

func (db *Database) GetWorkspace(ctx context.Context, workspaceId string) (entities.Workspace, error) {
	ctx, span := db.tracer.Start(ctx, "db.getWorkspace")
	defer span.End()
	workspace, err := models.WorkspaceByID(ctx, db.read(), workspaceId)
	if err != nil {
		return entities.Workspace{}, fmt.Errorf("unable to load workspace %s from db: %w", workspaceId, err)
	}
	if workspace == nil {
		return entities.Workspace{}, fmt.Errorf("unable to find workspace %s in db", workspaceId)
	}
	return entities.Workspace{
		Id:   workspace.ID,
		Name: workspace.Name,
	}, nil
}
