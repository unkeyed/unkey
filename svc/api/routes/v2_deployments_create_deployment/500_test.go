package handler_test

import (
	"net/http"
	"testing"

	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

// TestCreateRestateFailure pins what a caller sees when the create cannot be
// submitted at all. The ServiceUnavailable code maps to a 500 in the api error
// middleware rather than a 503, and no deployment id is handed out for work that
// was never accepted.
func TestCreateRestateFailure(t *testing.T) {
	t.Run("submission rejected", func(t *testing.T) {
		assertCreateRestateFailure(t, testutil.NewRestateIngressClient(t, http.StatusInternalServerError))
	})
	t.Run("transport unavailable", func(t *testing.T) {
		assertCreateRestateFailure(t, testutil.NewUnavailableRestateIngressClient(t))
	})
}

func assertCreateRestateFailure(t *testing.T, restate *restateingress.Client) {
	t.Helper()

	h := testutil.NewHarness(t)
	route := newRoute(h, restate)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	seedDeployableRegion(t, h, setup)

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
}
