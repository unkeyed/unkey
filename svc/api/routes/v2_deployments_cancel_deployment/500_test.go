package handler_test

import (
	"net/http"
	"testing"

	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_cancel_deployment"
)

func TestCancelDeploymentRestateFailure(t *testing.T) {
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

	setup := h.CreateTestDeploymentSetup()
	dep := seedDeployment(h, setup, mysqltype.DeploymentsStatusBuilding)
	rootKey := h.CreateRootKey(setup.Workspace.ID, deploymentURN(setup, dep.ID, "write"))

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, authHeaders(rootKey), handler.Request{
		DeploymentId: dep.ID,
	})
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	require.Equal(t, "Failed to cancel deployment.", res.Body.Error.Detail)
}
