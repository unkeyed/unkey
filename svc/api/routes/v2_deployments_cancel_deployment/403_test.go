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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_cancel_deployment"
)

func TestCancelDeploymentAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	restateClient, cancellations := newRecordingRestate(t)
	route := newRoute(h, restateClient)

	setup := h.CreateTestDeploymentSetup()
	other := seedDeployment(h, setup, mysqltype.DeploymentsStatusBuilding)

	testCases := []struct {
		name        string
		permissions func(deploymentID string) []string
		shouldPass  bool
	}{
		{name: "urn write on every deployment", permissions: func(string) []string { return []string{writeAnyDeployment(setup.Workspace.ID)} }, shouldPass: true},
		{name: "urn write on this deployment", permissions: func(id string) []string { return []string{deploymentURN(setup, id, "write")} }, shouldPass: true},
		{name: "urn write alongside unrelated", permissions: func(id string) []string { return []string{"api.*.read_api", deploymentURN(setup, id, "write")} }, shouldPass: true},
		{name: "legacy create deployment", permissions: func(string) []string { return []string{"environment.*.create_deployment"} }, shouldPass: false},
		{name: "legacy start deployment", permissions: func(string) []string { return []string{"environment.*.start_deployment"} }, shouldPass: false},
		{name: "urn read on this deployment", permissions: func(id string) []string { return []string{deploymentURN(setup, id, "read")} }, shouldPass: false},
		{name: "urn write on another deployment", permissions: func(string) []string { return []string{deploymentURN(setup, other.ID, "write")} }, shouldPass: false},
		{name: "no permissions", permissions: func(string) []string { return []string{} }, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dep := seedDeployment(h, setup, mysqltype.DeploymentsStatusBuilding)
			headers := authHeaders(h.CreateRootKey(setup.Workspace.ID, tc.permissions(dep.ID)...))
			req := handler.Request{DeploymentId: dep.ID}

			if tc.shouldPass {
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
				require.Equal(t, http.StatusAccepted, res.Status, "expected 202, received: %s", res.RawBody)
				require.Equal(t, dep.ID, testutil.Receive(t, cancellations, 10*time.Second).virtualObjectKey)
				return
			}

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.NotContains(t, res.RawBody, dep.ID)
		})
	}

	testutil.RequireNoReceive(t, cancellations, time.Second)
}

func TestCancelDeploymentExistenceNotLeaked(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))

	setup := h.CreateTestDeploymentSetup()
	dep := seedDeployment(h, setup, mysqltype.DeploymentsStatusBuilding)
	headers := authHeaders(h.CreateRootKey(setup.Workspace.ID, deploymentURN(setup, dep.ID, "read")))

	realRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{DeploymentId: dep.ID})
	missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{DeploymentId: uid.New(uid.DeploymentPrefix)})

	require.Equal(t, http.StatusNotFound, realRes.Status, "expected 404, received: %s", realRes.RawBody)
	require.Equal(t, http.StatusNotFound, missingRes.Status, "expected 404, received: %s", missingRes.RawBody)
	require.NotContains(t, realRes.RawBody, dep.ID)
	require.Equal(t, missingRes.Body.Error.Detail, realRes.Body.Error.Detail)
	require.Equal(t, missingRes.Body.Error.Type, realRes.Body.Error.Type)
	require.Equal(t, missingRes.Body.Error.Status, realRes.Body.Error.Status)
}
