package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

func TestInsufficientPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	// A key with an unrelated permission on the project.
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
	require.False(t, capture.called, "ctrl must not be called without create_deployment")
}
