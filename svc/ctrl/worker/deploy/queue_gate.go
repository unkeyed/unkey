package deploy

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
)

// skipIfSuperseded marks the current deployment as skipped and returns
// (true, nil) when a newer deployment for the same (app, environment, branch)
// has already been queued. Rapid pushes to the same branch only build the
// latest commit.
//
// Returns (false, nil) when the deployment should proceed normally, or
// (false, err) if the dedup query or status update fails.
//
// The check is race-free because Deploy is keyed by app_id — when this
// handler runs, any subsequent deploys for the same app are already queued
// in the Restate VO inbox.
func (w *Workflow) skipIfSuperseded(
	ctx restate.ObjectContext,
	deployment db.Deployment,
) (bool, error) {
	hasNewer, err := restate.Run(ctx, func(runCtx restate.RunContext) (bool, error) {
		return db.Query.HasNewerActiveDeployment(runCtx, w.db.RO(), db.HasNewerActiveDeploymentParams{
			AppID:         deployment.AppID,
			EnvironmentID: deployment.EnvironmentID,
			GitBranch:     deployment.GitBranch,
			CreatedAt:     deployment.CreatedAt,
			DeploymentID:  deployment.ID,
		})
	}, restate.WithName("check for newer active deployment"))
	if err != nil {
		return false, fault.Wrap(err, fault.Public("Failed to check for newer deployments."))
	}
	if !hasNewer {
		return false, nil
	}

	logger.Info("skipping superseded deployment",
		"deployment_id", deployment.ID,
		"app_id", deployment.AppID,
		"branch", deployment.GitBranch.String,
	)

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
		if updErr := db.Query.UpdateDeploymentStatus(runCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        deployment.ID,
			Status:    db.DeploymentsStatusSkipped,
			UpdatedAt: now,
		}); updErr != nil {
			return updErr
		}
		return db.Query.EndDeploymentStep(runCtx, w.db.RW(), db.EndDeploymentStepParams{
			DeploymentID: deployment.ID,
			Step:         db.DeploymentStepsStepQueued,
			EndedAt:      now,
			Error:        sql.NullString{Valid: true, String: "superseded by newer commit"},
		})
	}, restate.WithName("mark deployment skipped")); err != nil {
		return false, fault.Wrap(err, fault.Public("Failed to mark deployment as skipped."))
	}

	return true, nil
}

// waitForBuildSlot blocks until the workspace's BuildSlotService grants a
// build slot. Push-based via a Restate awakeable: the handler parks on
// `awakeable.Result()` and BuildSlotService resolves it when a slot becomes
// available (immediately if one is free, or later when another deployment
// releases its slot and this one reaches the head of the wait list).
//
// Production deployments always receive a slot immediately so a hotfix is
// never blocked behind a preview build.
//
// The caller is responsible for releasing the slot on both the success and
// failure paths (see releaseBuildSlot). On cancellation mid-wait, the
// Deploy handler's defer calls Release, which removes this deployment from
// BuildSlotService's wait_list so no orphan entries are left behind.
func (w *Workflow) waitForBuildSlot(
	ctx restate.ObjectContext,
	deployment db.Deployment,
	isProduction bool,
) error {
	workspaceID := deployment.WorkspaceID
	deploymentID := deployment.ID

	awakeable := restate.Awakeable[bool](ctx)

	if _, err := hydrav1.NewBuildSlotServiceClient(ctx, workspaceID).AcquireOrWait().Request(&hydrav1.AcquireOrWaitRequest{
		DeploymentId: deploymentID,
		AwakeableId:  awakeable.Id(),
		IsProduction: isProduction,
	}); err != nil {
		return fault.Wrap(err, fault.Public("Failed to request build slot."))
	}

	logger.Info("waiting for build slot",
		"workspace_id", workspaceID,
		"deployment_id", deploymentID,
	)

	granted, err := awakeable.Result()
	if err != nil {
		return fmt.Errorf("awaiting build slot: %w", err)
	}
	if !granted {
		// BuildSlotService only resolves with true, so this should never
		// happen — defensive.
		return fault.New("build slot was not granted", fault.Public("Failed to acquire build slot."))
	}

	logger.Info("build slot acquired",
		"workspace_id", workspaceID,
		"deployment_id", deploymentID,
	)
	return nil
}

// releaseBuildSlot frees the build slot held by a deployment. It is
// fire-and-forget and idempotent: releasing a non-held slot is a no-op, so
// it is safe to call from both the success path and the failure/cancel path.
func releaseBuildSlot(ctx restate.ObjectContext, workspaceID, deploymentID string) {
	hydrav1.NewBuildSlotServiceClient(ctx, workspaceID).Release().Send(
		&hydrav1.ReleaseSlotRequest{DeploymentId: deploymentID},
	)
}
