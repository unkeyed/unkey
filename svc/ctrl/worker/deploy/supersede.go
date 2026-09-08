package deploy

import (
	"context"
	"database/sql"
	"fmt"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

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
// reached starting holds a build slot and is left alone, otherwise rapid
// pushes could keep cancelling builds and never ship one. No audit entries
// are written because no user started this cancel.
//
// A sibling whose CancelInvocation fails is already superseded in the
// database. Deploy stops on that status at its top; if it is already parked in
// waitForBuildSlot it resumes when a slot frees and stops then
func (w *Workflow) cancelOlderSiblings(ctx context.Context, deploymentID string, payload deployPayload) error {
	branch := payload.RequestedBranch
	if branch == "" {
		return nil
	}

	older, err := w.db.ListOlderActiveDeploymentsForDedup(ctx, db.ListOlderActiveDeploymentsForDedupParams{
		AppID:         payload.Target.AppID,
		EnvironmentID: payload.Target.EnvironmentID,
		GitBranch:     sql.NullString{Valid: true, String: branch},
		CreatedAt:     payload.CreatedAt,
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
