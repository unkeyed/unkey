package database

import (
	"context"
	"errors"

	"github.com/go-sql-driver/mysql"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
)

func (db *database) InsertWorkspace(ctx context.Context, ws *workspacesv1.Workspace) error {

	params := gen.InsertWorkspaceParams{
		ID:           ws.WorkspaceId,
		Name:         ws.Name,
		TenantID:     ws.TenantId,
		Features:     []byte("{}"),
		BetaFeatures: []byte("{}"),
		Plan:         gen.NullWorkspacesPlan{Valid: true},
	}
	switch ws.Plan {
	case workspacesv1.Plan_PLAN_FREE:
		params.Plan.WorkspacesPlan = gen.WorkspacesPlanFree
	case workspacesv1.Plan_PLAN_PRO:
		params.Plan.WorkspacesPlan = gen.WorkspacesPlanPro
	case workspacesv1.Plan_PLAN_ENTERPRISE:
		params.Plan.WorkspacesPlan = gen.WorkspacesPlanEnterprise
	default:
		params.Plan.WorkspacesPlan = gen.WorkspacesPlanFree
	}

	err := db.write().InsertWorkspace(ctx, params)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			switch mysqlErr.Number {
			case 1062:
				return ErrNotUnique
			}
		}
		return err
	}
	return nil

}
