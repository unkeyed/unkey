package deploy

import (
	"database/sql"
	"fmt"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/buildslot"
)

// skipIfSuperseded marks the current deployment as superseded and returns
// (true, nil) when a newer deployment for the same (app, environment, branch)
// has already been queued. Rapid pushes to the same branch only build the
// latest commit. `skipped` is reserved for "watch paths didn't match", so
// supersession uses its own status here.
//
// Returns (false, nil) when the deployment should proceed normally, or
// (false, err) if the dedup query or status update fails.
//
// This catches an older sibling whose row landed after the newer deployment's
// CancelOlderSiblings query ran. The check sits at the top so the workflow bows
// out before it takes a build slot.
func (w *Workflow) skipIfSuperseded(
	ctx restate.ObjectContext,
	deployment db.Deployment,
) (bool, error) {
	hasNewer, err := restate.Run(ctx, func(runCtx restate.RunContext) (bool, error) {
		return w.db.HasNewerActiveDeployment(runCtx, db.HasNewerActiveDeploymentParams{
			AppID:         deployment.AppID,
			EnvironmentID: deployment.EnvironmentID,
			GitBranch:     deployment.GitBranch,
			CreatedAt:     deployment.CreatedAt,
			DeploymentID:  deployment.ID,
		})
	}, restate.WithName("check for newer active deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return false, fault.Wrap(err, fault.Public("Failed to check for newer deployments."))
	}
	if !hasNewer {
		return false, nil
	}

	logger.Info("self-superseding deployment",
		"deployment_id", deployment.ID,
		"app_id", deployment.AppID,
		"branch", deployment.GitBranch.String,
	)

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
		if updErr := w.db.UpdateDeploymentStatus(runCtx, db.UpdateDeploymentStatusParams{
			ID:        deployment.ID,
			Status:    mysqltype.DeploymentsStatusSuperseded,
			UpdatedAt: now,
		}); updErr != nil {
			return updErr
		}
		return w.db.EndDeploymentStep(runCtx, db.EndDeploymentStepParams{
			DeploymentID: deployment.ID,
			Step:         db.DeploymentStepsStepQueued,
			EndedAt:      now,
			Error:        sql.NullString{Valid: true, String: "superseded by newer commit"},
		})
	}, restate.WithName("mark deployment superseded"), restate.WithMaxRetryAttempts(runMaxAttempts)); err != nil {
		return false, fault.Wrap(err, fault.Public("Failed to mark deployment as superseded."))
	}

	return true, nil
}

// waitForBuildSlot blocks until the workspace's BuildSlotService grants a
// build slot, or [buildslot.MaxWaitDuration] elapses. Push-based via a
// Restate awakeable: the handler parks on the awakeable and BuildSlotService
// resolves it when a slot becomes available (immediately if one is free, or
// later when another deployment releases its slot and this one reaches the
// head of the wait list).
//
// The wait is bounded: the awakeable races [restate.After] (same pattern as
// [Workflow.waitForDeployments]). Without the bound, a slot accounting
// error upstream kept waiters suspended without limit. In production some
// Deploy invocations waited more than one week. On timeout the handler
// returns a terminal error. The Release compensation registered before this
// call removes the wait-list entry, or a slot granted in the same instant;
// Release handles both. The race between timeout and grant cannot leak
// occupancy.
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

	timeout := restate.After(ctx, buildslot.MaxWaitDuration)
	winner, err := restate.WaitFirst(ctx, awakeable, timeout)
	if err != nil {
		return fmt.Errorf("awaiting build slot or timeout: %w", err)
	}

	if winner != awakeable {
		return fault.Wrap(
			restate.TerminalErrorf("no build slot became available within %v", buildslot.MaxWaitDuration),
			fault.Public("Timed out waiting for a build slot. Too many builds are queued in this workspace."),
		)
	}

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
