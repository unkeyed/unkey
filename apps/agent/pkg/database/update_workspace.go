package database

import (
	"context"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
)

func (db *database) UpdateWorkspace(ctx context.Context, workspace entities.Workspace) error {
	params := transformWorkspaceEntityToUpdateWorkspaceParams(workspace)
	return db.write().UpdateWorkspace(ctx, params)
}

func transformWorkspaceEntityToUpdateWorkspaceParams(workspace entities.Workspace) gen.UpdateWorkspaceParams {
	return gen.UpdateWorkspaceParams{
		ID:   workspace.Id,
		Slug: workspace.Slug,
		Name: workspace.Name,
		Plan: gen.NullWorkspacesPlan{
			WorkspacesPlan: gen.WorkspacesPlan(workspace.Plan),
			Valid:          true,
		},
	}

}
