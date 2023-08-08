package database

import (
	"context"

	gen "github.com/unkeyed/unkey/apps/api/gen/database"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
)

func (db *database) CreateWorkspace(ctx context.Context, ws entities.Workspace) error {

	return db.primary.query.CreateWorkspace(ctx, gen.CreateWorkspaceParams{
		ID:       ws.Id,
		Name:     ws.Name,
		Slug:     ws.Slug,
		TenantID: ws.TenantId,
		Plan:     gen.NullWorkspacesPlan{WorkspacesPlan: gen.WorkspacesPlan(ws.Plan), Valid: ws.Plan != ""},
	})

}
