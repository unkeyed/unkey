package handler_test

import (
	"fmt"
	"net/http"

	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_start_deployment"
)

func newRoute(h *testutil.Harness, restate *restateingress.Client) *handler.Handler {
	return &handler.Handler{
		DB:      h.DB,
		Restate: restate,
	}
}

func deploymentRead(setup testutil.DeploymentTestSetup) string {
	return fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s/environments/%s/deployments/*#read",
		setup.Workspace.ID, setup.Project.ID, setup.App.ID, setup.Environment.ID)
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + rootKey},
	}
}

// writeDeployments is the permission to write every deployment, which this
// handler checks.
func writeDeployments(workspaceID string) string {
	return rbac.U(urn.New().Workspace(workspaceID).Project("*").App("*").Environment("*").Deployment("*"), permissions.Write).Value
}
