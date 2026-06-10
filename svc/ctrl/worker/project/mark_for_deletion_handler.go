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

// MarkForDeletion enters the soft-delete grace window using the
// deletion id minted by the public ctrl handler. Writes one deletions
// row, points the project at it, and cascades the same id to every
// live app. Children share the id (they don't mint their own) so the
// whole cascade tree carries a single identifier.
//
// Idempotent: if the project already has a deletion_id, skip the writes
// and the cascade (a re-issued cascade does not bump anything).
//
// Key: project_id
func (s *Service) MarkForDeletion(
	ctx restate.ObjectContext,
	req *hydrav1.MarkProjectForDeletionRequest,
) (*hydrav1.MarkProjectForDeletionResponse, error) {
	projectID := restate.Key(ctx)
	deletionID := req.GetDeletionId()
	deletePermanentlyAt := req.GetDeletePermanentlyAt()

	logger.Info("project mark for deletion",
		"project_id", projectID,
		"deletion_id", deletionID,
		"delete_permanently_at", deletePermanentlyAt,
	)

	project, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Project, error) {
		return db.Query.FindProjectAnyById(runCtx, s.db.RO(), projectID)
	}, restate.WithName("find project"))
	if err != nil {
		return nil, fmt.Errorf("find project: %w", err)
	}

	if project.DeletionID.Valid {
		logger.Info("project mark for deletion no-op: already scheduled",
			"project_id", projectID,
			"existing_deletion_id", project.DeletionID.String,
		)
		return &hydrav1.MarkProjectForDeletionResponse{}, nil
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.scheduleProject(runCtx, deletionID, project.WorkspaceID, projectID, deletePermanentlyAt)
	}, restate.WithName("schedule project")); err != nil {
		return nil, fmt.Errorf("schedule project: %w", err)
	}

	// Cascade to live apps only. Independently-deleted apps already have
	// a different deletion_id and stay untouched.
	appIDs, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		return db.Query.ListLiveAppIdsByProject(runCtx, s.db.RO(), projectID)
	}, restate.WithName("list live apps"))
	if err != nil {
		return nil, fmt.Errorf("list live apps: %w", err)
	}

	for _, appID := range appIDs {
		hydrav1.NewAppServiceClient(ctx, appID).
			MarkForDeletion().
			Send(&hydrav1.MarkAppForDeletionRequest{DeletionId: deletionID})
	}

	logger.Info("project mark for deletion cascade dispatched",
		"project_id", projectID,
		"deletion_id", deletionID,
		"apps_cascaded", len(appIDs),
	)

	return &hydrav1.MarkProjectForDeletionResponse{}, nil
}

// scheduleProject atomically inserts the deletions row and points the
// project's deletion_id at it. Both writes go together so a crash
// between them can't leave the resource scheduled-but-unreferenced (or
// vice versa).
func (s *Service) scheduleProject(
	ctx restate.RunContext,
	deletionID, workspaceID, projectID string,
	deletePermanentlyAt int64,
) error {
	tx, err := s.db.RW().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UnixMilli()

	if err := db.Query.InsertDeletion(ctx, tx, db.InsertDeletionParams{
		ID:                  deletionID,
		WorkspaceID:         workspaceID,
		ResourceType:        "project",
		ResourceID:          projectID,
		DeletePermanentlyAt: deletePermanentlyAt,
	}); err != nil {
		return fmt.Errorf("insert deletion row: %w", err)
	}

	res, err := db.Query.UpdateProjectDeletionId(ctx, tx, db.UpdateProjectDeletionIdParams{
		ID:                 projectID,
		ExpectedDeletionID: sql.NullString{Valid: false},
		DeletionID:         sql.NullString{Valid: true, String: deletionID},
		UpdatedAt:          sql.NullInt64{Valid: true, Int64: now},
	})
	if err != nil {
		return fmt.Errorf("set project deletion id: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		// Lost the CAS — someone else scheduled this project between our
		// read and our write. Surface as an error so Restate replays
		// the handler; the second pass will short-circuit on the
		// already-scheduled check at the top.
		return fmt.Errorf("project %s: CAS lost (concurrently scheduled)", projectID)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
