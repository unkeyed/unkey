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

const (
	// rebuildActorID and rebuildActorName name the synthetic actor recorded on an
	// operator rebuild. The bearer token is the only identity at this boundary,
	// so every rebuild shares one actor and the reason carries the detail.
	rebuildActorID   = "unkey-ops"
	rebuildActorName = "Unkey Ops"
)

// Rebuild creates a new deployment that re-runs what the source deployment ran:
// its commit when the app still has a repository connection, otherwise its
// image. The new deployment takes the app's current runtime settings and
// environment variables, so any configuration drift since the source applies.
//
// Unless force is set, it refuses when a newer active deployment exists on the
// same app, environment, and branch, because resurrecting a deployment past
// something already shipped is rarely what an operator meant.
//
// DeployService.Create writes the row and the deployment.rebuild audit entry.
func (s *Service) Rebuild(ctx context.Context, sourceDeploymentID, reason string, force bool) (string, error) {
	if sourceDeploymentID == "" {
		return "", connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("deployment_id is required"))
	}

	// This read supplies the project, app, and environment only. Create resolves
	// the source deployment again for the artifact.
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
	logger.Info(
		"rebuilding deployment",
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
			Decision:      hydrav1.CreateDecision_CREATE_DECISION_DEPLOY,
			Trigger:       ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_UNKEY,
			TriggeredBy:   "",
			TriggerReason: reason,
			Actor: &ctrlv1.ActorInfo{
				Id:        rebuildActorID,
				Name:      rebuildActorName,
				Type:      ctrlv1.ActorType_ACTOR_TYPE_SYSTEM,
				RemoteIp:  "",
				UserAgent: "",
				Meta:      map[string]string{"reason": reason},
			},
		})
	if err != nil {
		return "", connect.NewError(connect.CodeInternal,
			fmt.Errorf("rebuild failed: %w", err))
	}

	if resp.GetOutcome() == hydrav1.CreateOutcome_CREATE_OUTCOME_REJECTED {
		// Only the enum crosses the wire. The worker logs the explanation under
		// the deployment id above.
		return "", connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("rebuild rejected: %s", resp.GetRejectionReason().String()))
	}

	return deploymentID, nil
}
