package database

import (
	"context"
	"fmt"

	"github.com/chronark/unkey/apps/api/pkg/database/models"
	"github.com/chronark/unkey/apps/api/pkg/entities"
)

func (db *Database) GetWorkspace(ctx context.Context, workspaceId string) (entities.Workspace, error) {

	wprkspace, err := models.WorkspaceByID(ctx, db.read(), workspaceId)
	if err != nil {
		return entities.Workspace{}, fmt.Errorf("unable to load wprkspace %s from db: %w", workspaceId, err)
	}
	if wprkspace == nil {
		return entities.Workspace{}, fmt.Errorf("unable to find wprkspace %s in db", workspaceId)
	}
	return entities.Workspace{
		Id:   wprkspace.ID,
		Name: wprkspace.Name,
	}, nil
}
