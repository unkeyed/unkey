package deploy

import (
	"database/sql"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
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
// This catches the race where the proactive dedup in
// services/deployment.create_deployment didn't manage to cancel the older
// sibling before it started running (e.g. invocation_id hadn't been
// persisted yet). The workflow self-checks at the top so it can bow out
// before acquiring a build slot.
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
