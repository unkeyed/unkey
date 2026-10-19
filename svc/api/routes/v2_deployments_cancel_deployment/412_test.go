package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_cancel_deployment"
)

func TestCancelDeploymentPreconditionFailed(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))

	for _, status := range []mysqltype.DeploymentsStatus{
		mysqltype.DeploymentsStatusReady,
		mysqltype.DeploymentsStatusFailed,
		mysqltype.DeploymentsStatusStopped,
		mysqltype.DeploymentsStatusSkipped,
		mysqltype.DeploymentsStatusSuperseded,
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
			require.Equal(t, "https://unkey.com/docs/errors/unkey/application/deployment_not_in_progress", res.Body.Error.Type)
			require.Equal(t, "The deployment is not in progress. Its status is "+string(status)+".", res.Body.Error.Detail)
		})
	}
}
