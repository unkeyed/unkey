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
	route := newRoute(h, testutil.UncalledDeployRestate(t))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	for _, environment := range []string{setup.Environment.Slug, "does-not-exist"} {
		t.Run(environment, func(t *testing.T) {
			req := imageRequest(t, setup.Project.Slug, setup.App.Slug, environment, "nginx:latest")
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.NotContains(t, res.RawBody, setup.Environment.ID)
			require.Contains(t, res.RawBody, "does not exist")
		})
	}
}
