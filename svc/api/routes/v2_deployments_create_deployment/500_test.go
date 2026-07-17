package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

// TestControlPlaneInternalError verifies a non-precondition ctrl error is routed
// through HandleError. The ServiceUnavailable code it returns currently maps to
// a 500 in the api error middleware (not 503), so this pins that behavior.
func TestControlPlaneInternalError(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{
		err: connect.NewError(connect.CodeInternal, fmt.Errorf("ctrl exploded")),
	}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	require.True(t, capture.called)
}
