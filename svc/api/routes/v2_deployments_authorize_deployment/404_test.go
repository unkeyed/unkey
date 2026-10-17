package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_authorize_deployment"
)

func TestAuthorizeDeploymentNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))

	t.Run("missing deployment", func(t *testing.T) {
		setup := h.CreateTestDeploymentSetup()
		rootKey := h.CreateRootKey(setup.Workspace.ID, writeAnyDeployment(setup.Workspace.ID))

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(rootKey), handler.Request{
			DeploymentId: uid.New(uid.DeploymentPrefix),
		})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/deployment_not_found", res.Body.Error.Type)
	})

	t.Run("deployment in another workspace", func(t *testing.T) {
		caller := h.CreateTestDeploymentSetup()
		other := h.CreateTestDeploymentSetup()
		dep := seedDeployment(h, other, mysqltype.DeploymentsStatusAwaitingApproval)
		rootKey := h.CreateRootKey(caller.Workspace.ID, writeAnyDeployment(caller.Workspace.ID))

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, authHeaders(rootKey), handler.Request{
			DeploymentId: dep.ID,
		})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
		require.NotContains(t, res.RawBody, dep.ID)
	})
}
