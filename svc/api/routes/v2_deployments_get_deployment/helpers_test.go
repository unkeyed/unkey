package handler_test

import (
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"net/http"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_get_deployment"
)

func newRoute(h *testutil.Harness) *handler.Handler {
	return &handler.Handler{
		DB: h.DB,
	}
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + rootKey},
	}
}

// readDeployments grants read on every deployment in the workspace.
func readDeployments(workspaceID string) string {
	return rbac.U(urn.New().Workspace(workspaceID).Project("*").App("*").Environment("*").Deployment("*"), permissions.Read).Value
}
