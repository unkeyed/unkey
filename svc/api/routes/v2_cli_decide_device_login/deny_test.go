package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/internal/grant/granttest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_decide_device_login"
)

func TestDenyRequiresADashboardSession(t *testing.T) {
	h := testutil.NewHarness(t)
	devices := granttest.New(h)
	route := &handler.Handler{Grants: devices}
	h.Register(route)
	started, err := devices.Start(t.Context(), grant.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.51", UserAgent: ""})
	require.NoError(t, err)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write", "unkey:v1:"+workspace.ID+":rootKeys/*#read")
	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, http.Header{
		"Authorization": {"Bearer " + rootKey}, "Content-Type": {"application/json"},
	}, handler.Request{UserCode: started.UserCode, Decision: "deny"})
	require.Equal(t, http.StatusForbidden, res.Status, res.RawBody)
}

func TestDenyRejectsAnInvalidBearer(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Grants: granttest.New(h)}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, http.Header{
		"Authorization": {"Bearer invalid"}, "Content-Type": {"application/json"},
	}, handler.Request{UserCode: "TEST-0001", Decision: "deny"})
	require.Equal(t, http.StatusUnauthorized, res.Status, res.RawBody)
}
