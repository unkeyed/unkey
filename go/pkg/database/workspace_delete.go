package database

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/unkeyed/unkey/go/pkg/database/gen"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (db *database) DeleteWorkspace(ctx context.Context, id string, hardDelete bool) error {
	tx, err := db.writeReplica.db.BeginTx(ctx, nil)
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("failed to start transaction", ""))
	}
	defer func() {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			db.logger.Error(ctx, "failed to rollback transaction", slog.String("error", rollbackErr.Error()))
		}
	}()
	qtx := db.write().WithTx(tx)

	// Check protection within the transaction
	workspace, err := qtx.FindWorkspaceByID(ctx, id)
	if err != nil {
		return fault.Wrap(err,

			fault.WithDesc("failed to load workspace", ""),
		)
	}

	if workspace.DeleteProtection.Valid && workspace.DeleteProtection.Bool {
		return fault.New("unable to delete workspace",
			fault.WithTag(fault.PROTECTED_RESOURCE),
			fault.WithDesc("workspace is protected", "This workspace has delete protection enabled and cannot be deleted."),
		)
	}
	if hardDelete {
		_, err = qtx.HardDeleteWorkspace(ctx, id)
		if err != nil {
			return fault.Wrap(err, fault.WithDesc("failed to hard delete workspace", ""))
		}
	} else {
		_, err = qtx.SoftDeleteWorkspace(ctx, gen.SoftDeleteWorkspaceParams{
			ID: id,
			Now: sql.NullTime{
				Time:  db.clock.Now(),
				Valid: true,
			},
		})
		if err != nil {
			return fault.Wrap(err, fault.WithDesc("failed to soft delete workspace", ""))
		}

	}

	err = tx.Commit()
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("failed to commit transaction", ""))
	}

	return nil
}
