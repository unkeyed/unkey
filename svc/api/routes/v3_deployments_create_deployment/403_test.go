package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v3_deployments_create_deployment"
)

func TestMissingPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Restate: testutil.UncalledDeployRestate(t)}
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
		Oci:         &openapi.DeploymentSourceOCI{Image: "nginx:latest"},
	})
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
}
