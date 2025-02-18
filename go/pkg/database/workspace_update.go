package database

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/entities"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) UpdateWorkspacePlan(ctx context.Context, id string, plan entities.WorkspacePlan) error {
	result, err := db.write().UpdateWorkspacePlan(ctx, gen.UpdateWorkspacePlanParams{
		ID: id,
		Plan: gen.NullWorkspacesPlan{
			WorkspacesPlan: gen.WorkspacesPlan(plan),
			Valid:          true,
		}})
	if err != nil {
		return fault.Wrap(err,
			fault.WithDesc("failed to update workspace plan", ""),
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fault.Wrap(err,
			fault.WithDesc("failed to get rows affected", ""),
		)
	}

	if rowsAffected == 0 {
		return fault.Wrap(sql.ErrNoRows,
			fault.WithTag(fault.NOT_FOUND),
			fault.WithDesc("workspace not found", "The workspace you're trying to update doesn't exist or has been deleted."),
		)
	}

	return nil
}

func (db *database) UpdateWorkspaceEnabled(ctx context.Context, id string, enabled bool) error {
	result, err := db.write().UpdateWorkspaceEnabled(ctx, gen.UpdateWorkspaceEnabledParams{
		ID:      id,
		Enabled: enabled,
	})
	if err != nil {
		return fault.Wrap(err,
			fault.WithDesc("failed to update workspace enabled status", ""),
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fault.Wrap(err,
			fault.WithDesc("failed to get rows affected", ""),
		)
	}

	if rowsAffected == 0 {
		return fault.Wrap(sql.ErrNoRows,
			fault.WithTag(fault.NOT_FOUND),
			fault.WithDesc("workspace not found", "The workspace you're trying to update doesn't exist or has been deleted."),
		)
	}

	return nil
}
