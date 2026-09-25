package handler_test

import (
	"net/http"
	"testing"

	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

// imageRequest builds a create-deployment request for the image source.
func imageRequest(t *testing.T, project, app, env, dockerImage string) handler.Request {
	t.Helper()
	return handler.Request{
		Project:     project,
		App:         app,
		Environment: env,
		Image:       &openapi.DeploymentSourceImage{DockerImage: dockerImage},
	}
}

// gitRequest builds a create-deployment request for the git source. Callers set
// branch, commitSha, and repository on the passed value.
func gitRequest(t *testing.T, project, app, env string, git openapi.DeploymentSourceGit) handler.Request {
	t.Helper()
	return handler.Request{
		Project:     project,
		App:         app,
		Environment: env,
		Git:         &git,
	}
}

// deploymentRequest builds a create-deployment request for the deployment
// (redeploy) source.
func deploymentRequest(t *testing.T, project, app, env, deploymentID string) handler.Request {
	t.Helper()
	return handler.Request{
		Project:     project,
		App:         app,
		Environment: env,
		Deployment:  &openapi.DeploymentSourceDeployment{DeploymentId: deploymentID},
	}
}

func newRoute(h *testutil.Harness, restate *restateingress.Client) *handler.Handler {
	return &handler.Handler{
		DB:      h.DB,
		Restate: restate,
	}
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + rootKey},
	}
}
