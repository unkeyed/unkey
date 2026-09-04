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
	// The ops bearer token is the only identity at this boundary, so every
	// rebuild shares one actor.
	rebuildActorID   = "unkey-ops"
	rebuildActorName = "Unkey Ops"
)

// Rebuild creates a deployment that re-runs the source deployment's commit, or
// its image when the app has no repository connection, with the app's current
// settings. Unless force is set it refuses when a newer active deployment
// exists on the same app, environment, and branch.
func (s *Service) Rebuild(ctx context.Context, sourceDeploymentID, reason string, force bool) (string, error) {
	if sourceDeploymentID == "" {
		return "", connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("deployment_id is required"))
	}

	// Only the target comes from this read. Create resolves the source itself.
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
		// The worker logs the detail. Only the enum crosses the wire.
		return "", connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("rebuild rejected: %s", resp.GetRejectionReason().String()))
	}

	return deploymentID, nil
}
