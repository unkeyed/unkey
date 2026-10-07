package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestListBuildLogsBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	rootKey := buildLogsRootKey(h, setup)
	deploymentID := createDeployment(h, setup).deploymentID

	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "missing deploymentId", body: `{}`},
		{name: "deploymentId too short", body: `{"deploymentId":"d_"}`},
		{name: "deploymentId with invalid characters", body: `{"deploymentId":"d.1234"}`},
		{name: "unknown field", body: `{"deploymentId":"` + deploymentID + `","step":"KEBAP"}`},
		{name: "malformed json", body: `{"deploymentId": }`},
		{name: "empty stepId", body: `{"deploymentId":"` + deploymentID + `","stepId":""}`},
		{name: "stepId too long", body: `{"deploymentId":"` + deploymentID + `","stepId":"` + strings.Repeat("a", 257) + `"}`},
		{name: "limit zero", body: `{"deploymentId":"` + deploymentID + `","limit":0}`},
		{name: "limit above 500", body: `{"deploymentId":"` + deploymentID + `","limit":501}`},
		{name: "empty cursor", body: `{"deploymentId":"` + deploymentID + `","cursor":""}`},
		{name: "cursor is not a number", body: `{"deploymentId":"` + deploymentID + `","cursor":"KEBAP"}`},
		{name: "cursor is negative", body: `{"deploymentId":"` + deploymentID + `","cursor":"-1"}`},
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
