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

func TestAuthorizeDeploymentUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))

	setup := h.CreateTestDeploymentSetup()
	dep := seedDeployment(h, setup, mysqltype.DeploymentsStatusAwaitingApproval)

	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, authHeaders("invalid_token"), handler.Request{
		DeploymentId: dep.ID,
	})
	require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
}
