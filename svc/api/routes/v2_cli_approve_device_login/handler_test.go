package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice/clidevicetest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_approve_device_login"
)

func TestApproveRequiresADashboardSession(t *testing.T) {
	h := testutil.NewHarness(t)
	devices := clidevicetest.New(h)
	route := &handler.Handler{Devices: devices}
	h.Register(route)
	started, err := devices.Start(t.Context(), clidevice.StartRequest{DeviceName: "laptop", RemoteIP: "", UserAgent: ""})
	require.NoError(t, err)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write", "unkey:v1:"+workspace.ID+":rootKeys/*#read")
	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, http.Header{
		"Authorization": {"Bearer " + rootKey}, "Content-Type": {"application/json"},
	}, handler.Request{UserCode: started.UserCode, Permissions: []string{"unkey:v1:ws:rootKeys/*#read"}})
	require.Equal(t, http.StatusForbidden, res.Status, res.RawBody)
}

func TestApproveHandlerRejectsMissingMaliciousAndOverscopedPermissions(t *testing.T) {
	workspaceID := "ws_test"
	developer := &principal.Principal{
		Version:               principal.Version,
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: "user_dev", Name: "Dev", Type: principal.SubjectTypeUser},
		Source:                principal.JWTSource{Roles: []string{"developer"}},
		AuthorizedWorkspaceID: workspaceID,
		Permissions:           workos.PermissionsForRoles(workspaceID, []string{"developer"}),
	}
	cases := []struct {
		name string
		body string
		code codes.URN
	}{
		{name: "missing", body: `{"userCode":"ABCD-EFGH","permissions":[]}`, code: codes.App.Validation.InvalidInput.URN()},
		{name: "malicious", body: `{"userCode":"ABCD-EFGH","permissions":["not-a-permission"]}`, code: codes.App.Validation.InvalidInput.URN()},
		{name: "other workspace", body: `{"userCode":"ABCD-EFGH","permissions":["unkey:v1:ws_other:rootKeys/*#read"]}`, code: codes.App.Validation.InvalidInput.URN()},
		{name: "over scoped", body: `{"userCode":"ABCD-EFGH","permissions":["unkey:v1:ws_test:rootKeys/*#write"]}`, code: codes.Auth.Authorization.InsufficientPermissions.URN()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v2/cli.approveDeviceLogin", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			sess := &zen.Session{}
			require.NoError(t, sess.Init(httptest.NewRecorder(), req, 1<<20))
			sess.SetPrincipal(developer)
			err := (&handler.Handler{}).Handle(t.Context(), sess)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, tc.code, code)
		})
	}
}

func TestApproveRejectsAnInvalidBearer(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Devices: clidevicetest.New(h)}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, http.Header{
		"Authorization": {"Bearer invalid"}, "Content-Type": {"application/json"},
	}, handler.Request{UserCode: "TEST-0001", Permissions: []string{"unkey:v1:ws:rootKeys/*#read"}})
	require.Equal(t, http.StatusUnauthorized, res.Status, res.RawBody)
}
