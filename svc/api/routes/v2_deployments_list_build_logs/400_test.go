package handler_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_list_build_logs"
)

func TestListBuildLogsBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	rootKey := buildLogsRootKey(h, setup)
	deploymentID := createDeployment(h, setup).deploymentID

	typed := func(req handler.Request) string {
		body, err := json.Marshal(req)
		require.NoError(t, err)
		return string(body)
	}

	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "missing deploymentId", body: `{}`},
		{name: "deploymentId too short", body: typed(handler.Request{DeploymentId: "d_"})},
		{name: "deploymentId with invalid characters", body: typed(handler.Request{DeploymentId: "d.1234"})},
		{name: "unknown field", body: `{"deploymentId":"` + deploymentID + `","step":"KEBAP"}`},
		{name: "malformed json", body: `{"deploymentId": }`},
		{name: "empty stepId", body: typed(handler.Request{DeploymentId: deploymentID, StepId: new("")})},
		{name: "stepId too long", body: typed(handler.Request{DeploymentId: deploymentID, StepId: new(strings.Repeat("a", 257))})},
		{name: "limit zero", body: typed(handler.Request{DeploymentId: deploymentID, Limit: new(0)})},
		{name: "limit above 500", body: typed(handler.Request{DeploymentId: deploymentID, Limit: new(501)})},
		{name: "empty cursor", body: typed(handler.Request{DeploymentId: deploymentID, Cursor: new("")})},
		{name: "cursor is not a number", body: typed(handler.Request{DeploymentId: deploymentID, Cursor: new("KEBAP")})},
		{name: "cursor is negative", body: typed(handler.Request{DeploymentId: deploymentID, Cursor: new("-1")})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(route.Method(), route.Path(), strings.NewReader(tc.body))
			require.NoError(t, err)
			req.Header = authHeaders(rootKey)

			res := testutil.CallRaw[openapi.BadRequestErrorResponse](h, req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %s, received: %s", tc.body, res.RawBody)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type, "received: %s", res.RawBody)
		})
	}
}
