package database

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) InsertWorkspace(ctx context.Context, workspace entities.Workspace) error {

	params := gen.InsertWorkspaceParams{
		ID:       workspace.ID,
		TenantID: workspace.TenantID,
		Name:     workspace.Name,
	}

	err := db.write().InsertWorkspace(ctx, params)
	if err != nil {
		return fault.Wrap(err,
			fault.WithDesc("failed to insert workspace", ""),
		)
	}

	return nil
}
