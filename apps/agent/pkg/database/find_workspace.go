package database

import (
	"context"
	"database/sql"
	"fmt"

	"errors"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
)

func (db *database) FindWorkspace(ctx context.Context, workspaceId string) (*workspacesv1.Workspace, bool, error) {

	model, err := db.read().FindWorkspace(ctx, workspaceId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("unable to find workspace: %w", err)
	}

	workspace := transformWorkspaceModelToEntity(model)

	return workspace, true, nil
}

func transformWorkspaceModelToEntity(m gen.Workspace) *workspacesv1.Workspace {
	ws := &workspacesv1.Workspace{
		WorkspaceId: m.ID,
		Name:        m.Name,
		TenantId:    m.TenantID,
	}
	if m.Plan.Valid {
		switch m.Plan.WorkspacesPlan {
		case gen.WorkspacesPlanFree:
			ws.Plan = workspacesv1.Plan_PLAN_FREE
		case gen.WorkspacesPlanPro:
			ws.Plan = workspacesv1.Plan_PLAN_PRO
		case gen.WorkspacesPlanEnterprise:
			ws.Plan = workspacesv1.Plan_PLAN_ENTERPRISE
		}
	}
	return ws

}
