package handler_test

import (
	"net/http"
	"testing"

	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_stop_deployment"
)

func TestStopDeploymentRestateFailure(t *testing.T) {
	t.Run("submission rejected", func(t *testing.T) {
		assertRestateFailure(t, testutil.NewRestateIngressClient(t, http.StatusInternalServerError))
	})

	t.Run("transport unavailable", func(t *testing.T) {
		assertRestateFailure(t, testutil.NewUnavailableRestateIngressClient(t))
	})
}

func assertRestateFailure(t *testing.T, restate *restateingress.Client) {
	t.Helper()

	h := testutil.NewHarness(t)
	route := newRoute(h, restate)
	h.Register(route)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.stop_deployment"},
	})
	preview := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		AppID:       setup.App.ID,
		Slug:        "preview",
	})
	dep := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: preview.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, authHeaders(setup.RootKey), handler.Request{DeploymentId: dep.ID})
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	require.Equal(t, "Failed to stop deployment.", res.Body.Error.Detail)
}
