package project

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Restore reverses MarkForDeletion for this project and cascades to
// apps carrying the project's deletion_id. App and environment Restore
// handlers compare their own deletion_id against the request value so
// independently-deleted descendants (different deletion_id) are not
// touched.
//
// Key: project_id
func (s *Service) Restore(
	ctx restate.ObjectContext,
	_ *hydrav1.RestoreProjectRequest,
) (*hydrav1.RestoreProjectResponse, error) {
	projectID := restate.Key(ctx)

	project, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Project, error) {
		return db.Query.FindProjectAnyById(runCtx, s.db.RO(), projectID)
	}, restate.WithName("find project"))
	if err != nil {
		return nil, fmt.Errorf("find project: %w", err)
	}

	if !project.DeletionID.Valid {
		logger.Info("project restore is a no-op: not scheduled",
			"project_id", projectID,
		)
		return &hydrav1.RestoreProjectResponse{}, nil
	}

	deletionID := project.DeletionID.String
	logger.Info("project restore",
		"project_id", projectID,
		"deletion_id", deletionID,
	)

	// Cascade first, then unschedule the project. If we unscheduled
	// the project first and a downstream Send failed during a partial
	// cascade, the project would look live while orphan apps stay
	// pending-delete; on retry the live project's Restore would no
	// longer find a deletion_id and skip the cascade.
	appIDs, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		return db.Query.ListAppIdsByProject(runCtx, s.db.RO(), projectID)
	}, restate.WithName("list all apps"))
	if err != nil {
		return nil, fmt.Errorf("list apps: %w", err)
	}

	for _, appID := range appIDs {
		hydrav1.NewAppServiceClient(ctx, appID).
			Restore().
			Send(&hydrav1.RestoreAppRequest{DeletionId: deletionID})
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.unscheduleProject(runCtx, projectID, deletionID)
	}, restate.WithName("unschedule project")); err != nil {
		return nil, fmt.Errorf("unschedule project: %w", err)
	}

	logger.Info("project restore cascade dispatched",
		"project_id", projectID,
		"apps_cascaded", len(appIDs),
	)

	return &hydrav1.RestoreProjectResponse{}, nil
}

func (s *Service) unscheduleProject(ctx restate.RunContext, projectID, deletionID string) error {
	tx, err := s.db.RW().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UnixMilli()

	res, err := db.Query.UpdateProjectDeletionId(ctx, tx, db.UpdateProjectDeletionIdParams{
		ID:                 projectID,
		ExpectedDeletionID: sql.NullString{Valid: true, String: deletionID},
		DeletionID:         sql.NullString{Valid: false},
		UpdatedAt:          sql.NullInt64{Valid: true, Int64: now},
	})
	if err != nil {
		return fmt.Errorf("clear deletion id: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		// The project's deletion_id changed since we read it. Could be
		// another restore for the same id (idempotent, fine) or a fresh
		// soft-delete with a new id. Either way, we must not delete
		// the deletions row we were targeting; returning nil here lets
		// the deferred Rollback drop the partial write.
		logger.Info("project restore no-op: deletion_id changed since read",
			"project_id", projectID,
			"expected_deletion_id", deletionID,
		)
		return nil
	}

	// Delete by id directly — the deletions row is the cascade root and
	// uniquely identified by its id.
	if err := db.Query.DeleteDeletionById(ctx, tx, deletionID); err != nil {
		return fmt.Errorf("delete deletion row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
