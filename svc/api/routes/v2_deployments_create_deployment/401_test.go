package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

func TestUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer invalid_token"},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
	require.False(t, capture.called, "ctrl must not be called for an unauthenticated request")
}
