package handler_test

import (
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

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + rootKey},
	}
}

// writeDeploymentEnv grants write on every environment and every deployment in
// it, which together are the grants the lifecycle handlers check.
func writeDeploymentEnv(workspaceID string) []string {
	return []string{
		rbac.U(urn.New().Workspace(workspaceID).Project("*").App("*").Environment("*"), permissions.Write).Value,
		rbac.U(urn.New().Workspace(workspaceID).Project("*").App("*").Environment("*").Deployment("*"), permissions.Write).Value,
	}
}

// readDeploymentEnv grants read at every level and no write, so every lifecycle
// handler refuses the caller.
func readDeploymentEnv(workspaceID string) string {
	return rbac.U(urn.New().Workspace(workspaceID).Project("*").App("*").Environment("*"), permissions.Read).Value
}
