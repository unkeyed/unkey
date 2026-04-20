// Package dedup centralises the "cancel in-progress siblings" logic used
// when a new deployment is created for a branch that already has an
// active build. A fresh commit supersedes the current build instead of
// queueing behind it.
package dedup

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
)

// SupersededByNewerCommitMessage is stamped onto the in-flight deployment
// step of a sibling that is being cancelled because a newer commit for the
// same branch landed. The frontend matches on this exact string to render
// the superseded deployment with a dedicated view instead of the red
// FailedDeploymentBanner. Must stay in sync with SUPERSEDED_BY_NEWER in
// web/apps/dashboard/.../cancelled-marker.ts.
const SupersededByNewerCommitMessage = "Superseded by newer commit"

// Service handles cancellation of superseded sibling deployments.
type Service struct {
	db    db.Database
	admin *restateadmin.Client
}

// New creates a dedup Service.
func New(database db.Database, admin *restateadmin.Client) *Service {
	return &Service{db: database, admin: admin}
}

// Newer identifies the deployment that triggered sibling cancellation.
type Newer struct {
	ID            string
	AppID         string
	EnvironmentID string
	GitBranch     string
	CreatedAt     int64
}

// CancelOlderSiblings finds deployments for the same (app, environment,
// branch) that were created before newer and are still in the build queue
// (status `pending` or `awaiting_approval` -- haven't acquired a build slot
// yet), then marks them superseded and cancels their Restate invocations.
//
// Once a deployment transitions out of `pending` (acquired a build slot and
// moved to `starting`/`building`/etc), it is committed and will not be
// superseded by a newer commit. This avoids the pathological "rapid pushes
// keep cancelling builds and nothing ever finishes" scenario.
//
// Best-effort: returns an error only when the initial DB lookup fails.
// Per-deployment errors are logged but don't stop the loop.
//
// Only git-sourced deployments with a branch are deduplicated -- docker
// image redeploys are manual and should never cancel siblings.
func (s *Service) CancelOlderSiblings(ctx context.Context, newer Newer) error {
	if newer.GitBranch == "" {
		return nil
	}

	older, err := db.Query.ListOlderActiveDeploymentsForDedup(ctx, s.db.RO(), db.ListOlderActiveDeploymentsForDedupParams{
		AppID:         newer.AppID,
		EnvironmentID: newer.EnvironmentID,
		GitBranch:     sql.NullString{Valid: true, String: newer.GitBranch},
		CreatedAt:     newer.CreatedAt,
		DeploymentID:  newer.ID,
	})
	if err != nil {
		return fmt.Errorf("list older active deployments: %w", err)
	}

	if len(older) == 0 {
		return nil
	}

	logger.Info("cancelling superseded sibling deployments",
		"count", len(older),
		"newer_deployment_id", newer.ID,
		"app_id", newer.AppID,
		"environment_id", newer.EnvironmentID,
		"branch", newer.GitBranch,
	)

	now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}

	deploymentIDs := make([]string, 0, len(older))
	for _, old := range older {
		deploymentIDs = append(deploymentIDs, old.ID)
	}

	// ONE query to stamp all in-flight steps with the superseded marker.
	// First-write-wins (WHERE ended_at IS NULL) so the Deploy handler's
	// own step-end is a no-op and our marker wins.
	err = db.Query.EndActiveDeploymentStepsForDeployments(ctx, s.db.RW(), db.EndActiveDeploymentStepsForDeploymentsParams{
		EndedAt:       now,
		Error:         sql.NullString{Valid: true, String: SupersededByNewerCommitMessage},
		DeploymentIds: deploymentIDs,
	})
	if err != nil {
		logger.Warn("failed to batch-stamp superseded marker on deployment steps",
			"deployment_ids", deploymentIDs,
			"error", err,
		)
	}

	// ONE query to transition all siblings to superseded.
	err = db.Query.UpdateDeploymentStatusBatch(ctx, s.db.RW(), db.UpdateDeploymentStatusBatchParams{
		Status:    db.DeploymentsStatusSuperseded,
		UpdatedAt: now,
		Ids:       deploymentIDs,
	})
	if err != nil {
		logger.Error("failed to batch-mark deployments as superseded",
			"deployment_ids", deploymentIDs,
			"error", err,
		)
	}

	// Cancel each Restate invocation. Deployments without an invocation ID
	// (orphans -- workflow never started) are already marked superseded
	// above, so there's nothing else to do for them.
	for _, old := range older {
		if !old.InvocationID.Valid || old.InvocationID.String == "" || s.admin == nil {
			continue
		}

		err = s.admin.CancelInvocation(ctx, old.InvocationID.String)
		if err != nil {
			logger.Error("failed to cancel superseded deployment invocation",
				"deployment_id", old.ID,
				"invocation_id", old.InvocationID.String,
				"error", err,
			)
		}
	}

	return nil
}
