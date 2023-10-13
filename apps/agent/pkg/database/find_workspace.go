package database

import (
	"context"
	"database/sql"
	"fmt"

	"errors"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"

	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
)

func (db *database) FindWorkspace(ctx context.Context, workspaceId string) (entities.Workspace, bool, error) {

	model, err := db.read().FindWorkspace(ctx, workspaceId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.Workspace{}, false, nil
		}
		return entities.Workspace{}, false, fmt.Errorf("unable to find workspace: %w", err)
	}

	workspace := transformWorkspaceModelToEntity(model)

	return workspace, true, nil
}

func transformWorkspaceModelToEntity(m gen.Workspace) entities.Workspace {
	return entities.Workspace{
		Id:       m.ID,
		Name:     m.Name,
		TenantId: m.TenantID,
		Plan:     entities.Plan(m.Plan.WorkspacesPlan),
	}

}
