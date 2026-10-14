package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_authorize_deployment"
)

func TestAuthorizeDeploymentPreconditionFailed(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))

	t.Run("not awaiting approval", func(t *testing.T) {
		for _, status := range []mysqltype.DeploymentsStatus{
			mysqltype.DeploymentsStatusPending,
			mysqltype.DeploymentsStatusBuilding,
			mysqltype.DeploymentsStatusReady,
			mysqltype.DeploymentsStatusCancelled,
		} {
			t.Run(string(status), func(t *testing.T) {
				setup := h.CreateTestDeploymentSetup()
				dep := seedDeployment(h, setup, status)
				rootKey := h.CreateRootKey(setup.Workspace.ID, deploymentURN(setup, dep.ID, "write"))

				res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(rootKey), handler.Request{
					DeploymentId: dep.ID,
				})
				require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
				require.Equal(t, "https://unkey.com/docs/errors/unkey/application/deployment_not_awaiting_approval", res.Body.Error.Type)
				require.Equal(t, "The deployment is not awaiting approval. Its status is "+string(status)+".", res.Body.Error.Detail)
			})
		}
	})

	t.Run("no compute plan", func(t *testing.T) {
		setup := h.CreateTestDeploymentSetup()
		h.ClearComputePlanOverride(setup.Workspace.ID)
		dep := seedDeployment(h, setup, mysqltype.DeploymentsStatusAwaitingApproval)
		rootKey := h.CreateRootKey(setup.Workspace.ID, deploymentURN(setup, dep.ID, "write"))

		res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(rootKey), handler.Request{
			DeploymentId: dep.ID,
		})
		require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/precondition_failed", res.Body.Error.Type)
	})
}
