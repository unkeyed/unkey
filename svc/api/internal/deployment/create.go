package deployment

import (
	"context"
	"strings"

	restateingress "github.com/restatedev/sdk-go/ingress"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
)

func Create(ctx context.Context, client *restateingress.Client, req *hydrav1.DeployCreateRequest) (string, error) {
	deploymentID := uid.New(uid.DeploymentPrefix)

	res, err := hydrav1.NewDeployServiceIngressClient(client, deploymentID).
		Create().
		Request(ctx, req)
	if err != nil {
		return "", fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to submit deployment create to Restate"),
			fault.Public("Failed to create deployment."),
		)
	}
	if err := errorForOutcome(res.GetOutcome(), res.GetDetail()); err != nil {
		return "", err
	}
	return deploymentID, nil
}

// TriggerFromClient reads X-Unkey-Client: the CLI sends unkey-cli/<version>,
// the dashboard proxy sends unkey-dashboard. Anything else is the API.
func TriggerFromClient(s *zen.Session) ctrlv1.DeploymentTrigger {
	client := s.Request().Header.Get("X-Unkey-Client")
	switch {
	case strings.HasPrefix(client, "unkey-cli/"):
		return ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_CLI
	case strings.HasPrefix(client, "unkey-dashboard"):
		return ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_DASHBOARD
	default:
		return ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_API
	}
}

// RequireSourceInEnvironment fails unless the deployment a caller wants to
// redeploy belongs to their workspace, app, and environment. A mismatch is
// reported as not found so a caller cannot probe for deployments it cannot read.
func RequireSourceInEnvironment(ctx context.Context, database db.Database, workspaceID, appID, environmentID, deploymentID string) error {
	source, err := db.Query.FindDeploymentById(ctx, database.RO(), deploymentID)
	if err != nil && !db.IsNotFound(err) {
		return fault.Wrap(err, fault.Internal("failed to find deployment"))
	}

	if db.IsNotFound(err) ||
		source.WorkspaceID != workspaceID ||
		source.AppID != appID ||
		source.EnvironmentID != environmentID {
		return fault.New(
			"deployment not found",
			fault.Code(codes.Data.Deployment.NotFound.URN()),
			fault.Internal("deployment does not exist or does not match this workspace, app, and environment"),
			fault.Public("The specified deployment does not exist."),
		)
	}

	return nil
}
