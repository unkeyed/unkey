package database

import (
	"context"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
)

func (db *database) UpdateWorkspace(ctx context.Context, workspace *workspacesv1.Workspace) error {
	params := transformWorkspaceEntityToUpdateWorkspaceParams(workspace)
	return db.write().UpdateWorkspace(ctx, params)
}

func transformWorkspaceEntityToUpdateWorkspaceParams(workspace *workspacesv1.Workspace) gen.UpdateWorkspaceParams {
	params := gen.UpdateWorkspaceParams{
		ID:   workspace.WorkspaceId,
		Name: workspace.Name,
		Plan: gen.NullWorkspacesPlan{Valid: true},
	}
	switch workspace.Plan {
	case workspacesv1.Plan_PLAN_FREE:
		params.Plan.WorkspacesPlan = gen.WorkspacesPlanFree
	case workspacesv1.Plan_PLAN_PRO:
		params.Plan.WorkspacesPlan = gen.WorkspacesPlanPro
	case workspacesv1.Plan_PLAN_ENTERPRISE:
		params.Plan.WorkspacesPlan = gen.WorkspacesPlanEnterprise
	}

	return params

}
