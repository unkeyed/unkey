package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_authorize_deployment"
)

func TestAuthorizeDeploymentAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	restateClient, authorizations := newRecordingRestate(t)
	route := newRoute(h, restateClient)

	setup := h.CreateTestDeploymentSetup()
	other := seedDeployment(h, setup, mysqltype.DeploymentsStatusAwaitingApproval)

	testCases := []struct {
		name        string
		permissions func(deploymentID string) []string
		shouldPass  bool
		readable    bool
	}{
		{name: "urn write on every deployment", permissions: func(string) []string { return []string{writeAnyDeployment(setup.Workspace.ID)} }, shouldPass: true},
		{name: "urn write on this deployment", permissions: func(id string) []string { return []string{deploymentURN(setup, id, "write")} }, shouldPass: true},
		{name: "urn write alongside unrelated", permissions: func(id string) []string { return []string{"api.*.read_api", deploymentURN(setup, id, "write")} }, shouldPass: true},
		{name: "legacy create deployment", permissions: func(string) []string { return []string{"environment.*.create_deployment"} }, shouldPass: false},
		{name: "legacy start deployment", permissions: func(string) []string { return []string{"environment.*.start_deployment"} }, shouldPass: false},
		{name: "urn read on this deployment", permissions: func(id string) []string { return []string{deploymentURN(setup, id, "read")} }, shouldPass: false, readable: true},
		{name: "urn write on another deployment", permissions: func(string) []string { return []string{deploymentURN(setup, other.ID, "write")} }, shouldPass: false},
		{name: "no permissions", permissions: func(string) []string { return []string{} }, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dep := seedDeployment(h, setup, mysqltype.DeploymentsStatusAwaitingApproval)
			headers := authHeaders(h.CreateRootKey(setup.Workspace.ID, tc.permissions(dep.ID)...))
			req := handler.Request{DeploymentId: dep.ID}

			if tc.shouldPass {
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
				require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)
				require.Equal(t, dep.ID, testutil.Receive(t, authorizations, 10*time.Second).virtualObjectKey)
				return
			}

			if tc.readable {
				res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
				require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
				require.Contains(t, res.Body.Error.Detail, "#write")
				require.Contains(t, res.Body.Error.Detail, "Log in again")
				require.NotContains(t, res.RawBody, dep.ID)
				return
			}

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.NotContains(t, res.RawBody, dep.ID)
		})
	}

	testutil.RequireNoReceive(t, authorizations, time.Second)
}

func TestAuthorizeDeploymentExistenceNotLeaked(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))

	setup := h.CreateTestDeploymentSetup()
	dep := seedDeployment(h, setup, mysqltype.DeploymentsStatusAwaitingApproval)
	headers := authHeaders(h.CreateRootKey(setup.Workspace.ID, deploymentURN(setup, dep.ID, "read")))

	realRes := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, handler.Request{DeploymentId: dep.ID})
	missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{DeploymentId: uid.New(uid.DeploymentPrefix)})

	require.Equal(t, http.StatusForbidden, realRes.Status, "expected 403, received: %s", realRes.RawBody)
	require.NotContains(t, realRes.RawBody, dep.ID)
	require.Contains(t, realRes.Body.Error.Detail, "#write")
	require.Contains(t, realRes.Body.Error.Detail, "Log in again")
	require.Equal(t, http.StatusNotFound, missingRes.Status, "expected 404, received: %s", missingRes.RawBody)
	require.NotContains(t, missingRes.Body.Error.Detail, "#write")
	require.NotContains(t, missingRes.Body.Error.Detail, "#delete")
}
