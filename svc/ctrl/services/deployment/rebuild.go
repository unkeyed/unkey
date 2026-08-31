package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Rebuild creates a new deployment that re-runs what the source deployment ran:
// its commit when the app still has a repository connection, otherwise its
// image. The new deployment inherits the app's current runtime settings and
// environment variables, so configuration drift since the source applies, which
// is the point for a hotfix or a settings rollout as much as for image-loss
// recovery.
//
// Unless force is set, it refuses when a newer active deployment exists on the
// same app, environment, and branch: resurrecting an older deployment past
// something already shipped is almost never what an operator meant.
//
// The row is written by DeployService.Create, which also records the
// deployment.rebuild audit entry. No idempotency key: an operator asking twice
// means twice.
func (s *Service) Rebuild(ctx context.Context, sourceDeploymentID, reason string, force bool) (string, error) {
	if sourceDeploymentID == "" {
		return "", connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("deployment_id is required"))
	}

	// The source row supplies the target. Create resolves the source deployment
	// again for the artifact; this read is for project, app, and environment.
	src, err := s.db.FindDeploymentById(ctx, sourceDeploymentID)
	if err != nil {
		if db.IsNotFound(err) {
			return "", connect.NewError(connect.CodeNotFound,
				fmt.Errorf("source deployment %q not found", sourceDeploymentID))
		}
		return "", connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup source deployment: %w", err))
	}

	deploymentID := uid.New(uid.DeploymentPrefix)
	logger.Info("rebuilding deployment",
		"source_deployment_id", sourceDeploymentID,
		"deployment_id", deploymentID,
		"app_id", src.AppID,
		"force", force,
		"reason", reason,
	)

	resp, err := hydrav1.NewDeployServiceIngressClient(s.restate, deploymentID).
		Create().
		Request(ctx, &hydrav1.DeployCreateRequest{
			ProjectId:   src.ProjectID,
			AppId:       src.AppID,
			Environment: src.EnvironmentID,
			Source: &hydrav1.DeployCreateRequest_ExistingDeployment{
				ExistingDeployment: &hydrav1.CreateExistingDeploymentSource{
					DeploymentId:   sourceDeploymentID,
					RequireNoNewer: !force,
				},
			},
			Command:           nil,
			Decision:          hydrav1.CreateDecision_CREATE_DECISION_DEPLOY,
			OrderingTimestamp: 0,
			Trigger:           ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_UNKEY,
			TriggeredBy:       "",
			TriggerReason:     reason,
			Actor:             nil,
		})
	if err != nil {
		return "", connect.NewError(connect.CodeInternal,
			fmt.Errorf("rebuild failed: %w", err))
	}

	if resp.GetOutcome() == hydrav1.CreateOutcome_CREATE_OUTCOME_BLOCKED {
		// The text explaining the reason is in the worker's logs, filed under the
		// deployment id above.
		return "", connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("rebuild blocked: %s", resp.GetBlockedReason().String()))
	}

	return deploymentID, nil
}
