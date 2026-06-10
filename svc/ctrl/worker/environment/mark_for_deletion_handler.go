package environment

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// MarkForDeletion points the environment at the cascade's shared
// deletion_id and flips every non-terminal deployment under it to
// status='stopped'. Krane reconciles status='stopped' as "no pods
// desired" so this is the signal that suspends the workload during
// the grace window. The deletions row was already written by the
// cascade root; this handler only sets the per-resource reference.
//
// Idempotent on the deletion_id write; the deployment stop is also
// safe to re-run because StopDeploymentsByEnvironmentId skips already-
// terminal deployments.
//
// Key: environment_id
func (s *Service) MarkForDeletion(
	ctx restate.ObjectContext,
	req *hydrav1.MarkEnvironmentForDeletionRequest,
) (*hydrav1.MarkEnvironmentForDeletionResponse, error) {
	envID := restate.Key(ctx)
	deletionID := req.GetDeletionId()

	logger.Info("environment mark for deletion",
		"environment_id", envID,
		"deletion_id", deletionID,
	)

	env, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Environment, error) {
		return db.Query.FindEnvironmentAnyById(runCtx, s.db.RO(), envID)
	}, restate.WithName("find environment"))
	if err != nil {
		return nil, fmt.Errorf("find environment: %w", err)
	}

	if !env.DeletionID.Valid {
		if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			res, err := db.Query.UpdateEnvironmentDeletionId(runCtx, s.db.RW(), db.UpdateEnvironmentDeletionIdParams{
				ID:                 envID,
				ExpectedDeletionID: sql.NullString{Valid: false},
				DeletionID:         sql.NullString{Valid: true, String: deletionID},
				UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			if err != nil {
				return err
			}
			affected, err := res.RowsAffected()
			if err != nil {
				return err
			}
			if affected == 0 {
				return fmt.Errorf("environment %s: CAS lost (concurrently scheduled)", envID)
			}
			return nil
		}, restate.WithName("set environment deletion id")); err != nil {
			return nil, fmt.Errorf("set environment deletion id: %w", err)
		}
	}

	// Stop the deployments regardless of whether the env was already
	// scheduled: an earlier cascade may have died between scheduling and
	// stopping, and a re-run should still drive the side effect home.
	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.StopDeploymentsByEnvironmentId(runCtx, s.db.RW(), db.StopDeploymentsByEnvironmentIdParams{
			EnvironmentID: envID,
			UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("stop deployments")); err != nil {
		return nil, fmt.Errorf("stop deployments: %w", err)
	}

	return &hydrav1.MarkEnvironmentForDeletionResponse{}, nil
}
