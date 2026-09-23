package deploy

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploycancel"
)

// SupersededByNewerCommitMessage is shown on the open step of an older
// deployment cancelled because a newer commit landed on the same branch. The
// dashboard picks its superseded view from deployments.status, not this text
const SupersededByNewerCommitMessage = "Superseded by newer commit"

// cancelOlderSiblings moves older pending or awaiting_approval deployments on
// the same app, environment, and branch to superseded. A deployment that
// reached building has begun its Build and is left alone, otherwise rapid
// pushes could keep cancelling builds and never ship one. No audit entries are
// written because no user started this cancel.
//
// A sibling whose CancelInvocation fails is already superseded in the
// database. Deploy stops on that status at its top; if its Build is already
// queued, Build reads the row when Restate lets it run and returns without
// building
func (w *Workflow) cancelOlderSiblings(ctx context.Context, deploymentID string, payload deployPayload) error {
	branch := payload.RequestedBranch
	if branch == "" {
		return nil
	}

	older, err := w.db.ListOlderActiveDeploymentsForDedup(ctx, db.ListOlderActiveDeploymentsForDedupParams{
		AppID:         payload.Target.AppID,
		EnvironmentID: payload.Target.EnvironmentID,
		GitBranch:     sql.NullString{Valid: true, String: branch},
		DeploymentID:  deploymentID,
	})
	if err != nil {
		return fmt.Errorf("list older active deployments: %w", err)
	}
	if len(older) == 0 {
		return nil
	}

	logger.Info("cancelling superseded sibling deployments",
		"count", len(older),
		"newer_deployment_id", deploymentID,
		"app_id", payload.Target.AppID,
		"environment_id", payload.Target.EnvironmentID,
		"branch", branch,
	)

	deployments := make([]deploycancel.Deployment, 0, len(older))
	for _, old := range older {
		deployments = append(deployments, deploycancel.Deployment{ID: old.ID, InvocationID: old.InvocationID.String})
	}

	return deploycancel.Cancel(ctx, w.db, w.restateAdmin, deploycancel.Params{
		Deployments: deployments,
		Reason:      SupersededByNewerCommitMessage,
		Status:      mysqltype.DeploymentsStatusSuperseded,
		Audit:       nil,
	})
}

// skipIfSuperseded marks the current deployment as superseded and returns
// (true, nil) when a newer deployment for the same (app, environment, branch)
// has already been queued. Rapid pushes to the same branch only build the
// latest commit. The skipped status means watch paths did not match, so this
// writes superseded instead.
//
// Returns (false, nil) when the deployment should proceed normally, or
// (false, err) if the sibling query or status update fails.
//
// cancelOlderSiblings can miss a sibling whose insert committed after
// its list query ran. That sibling catches itself here, before it calls
// Build
func (w *Workflow) skipIfSuperseded(
	ctx restate.Context,
	deployment db.FindDeploymentForDeployRow,
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

	// Two Runs, not one: UpdateDeploymentStatus is unconditional, so a retry
	// driven by the step write failing would write superseded over whatever
	// the status had become in between
	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return w.db.UpdateDeploymentStatus(runCtx, db.UpdateDeploymentStatusParams{
			ID:        deployment.ID,
			Status:    mysqltype.DeploymentsStatusSuperseded,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("mark deployment superseded"), restate.WithMaxRetryAttempts(runMaxAttempts)); err != nil {
		return false, fault.Wrap(err, fault.Public("Failed to mark deployment as superseded."))
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return w.db.EndDeploymentStep(runCtx, db.EndDeploymentStepParams{
			DeploymentID: deployment.ID,
			Step:         db.DeploymentStepsStepQueued,
			EndedAt:      sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			Error:        sql.NullString{Valid: true, String: SupersededByNewerCommitMessage},
		})
	}, restate.WithName("end queued step as superseded"), restate.WithMaxRetryAttempts(runMaxAttempts)); err != nil {
		return false, fault.Wrap(err, fault.Public("Failed to mark deployment as superseded."))
	}

	return true, nil
}
