package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/internal/grant/granttest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_decide_device_login"
)

func TestApproveRequiresADashboardSession(t *testing.T) {
	h := testutil.NewHarness(t)
	devices := granttest.New(h)
	route := &handler.Handler{Grants: devices}
	h.Register(route)
	started, err := devices.Start(t.Context(), grant.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.50", UserAgent: ""})
	require.NoError(t, err)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#write", "unkey:v1:"+workspace.ID+":rootKeys/*#read")
	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, http.Header{
		"Authorization": {"Bearer " + rootKey}, "Content-Type": {"application/json"},
	}, handler.Request{UserCode: started.UserCode, Decision: "approve", Permissions: new([]string{"unkey:v1:" + workspace.ID + ":rootKeys/*#read"})})
	require.Equal(t, http.StatusForbidden, res.Status, res.RawBody)
}

func TestApproveRejectsMaliciousAndOverscopedPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID
	route := &handler.Handler{Grants: granttest.New(h)}
	developer := &principal.Principal{
		Version:               principal.Version,
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: uid.New(uid.TestPrefix), Name: "Dev", Type: principal.SubjectTypeUser},
		Source:                principal.JWTSource{Roles: []string{"developer"}},
		AuthorizedWorkspaceID: workspaceID,
		Permissions:           workos.PermissionsForRoles(workspaceID, []string{"developer"}),
	}
	h.Register(route, append(h.PublicMiddleware(), func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, session *zen.Session) error {
			session.SetPrincipal(developer)
			return next(ctx, session)
		}
	})...)
	cases := []struct {
		name        string
		permissions []string
		code        codes.Code
		status      int
	}{
		{name: "malicious", permissions: []string{"not-a-permission"}, code: codes.App.Validation.InvalidInput, status: http.StatusBadRequest},
		{name: "other workspace", permissions: []string{"unkey:v1:" + uid.New(uid.WorkspacePrefix) + ":rootKeys/*#read"}, code: codes.App.Validation.InvalidInput, status: http.StatusBadRequest},
		{name: "over scoped", permissions: []string{"unkey:v1:" + workspaceID + ":rootKeys/*#write"}, code: codes.Auth.Authorization.InsufficientPermissions, status: http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer " + uid.New(uid.TestPrefix)},
			}, handler.Request{UserCode: "ABCD-EFGH", Decision: "approve", Permissions: &tc.permissions})
			require.Equal(t, tc.status, res.Status, res.RawBody)
			require.Equal(t, tc.code.DocsURL(), res.Body.Error.Type)
		})
	}
}

func TestApproveRejectsAnInvalidBearer(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Grants: granttest.New(h)}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, http.Header{
		"Authorization": {"Bearer invalid"}, "Content-Type": {"application/json"},
	}, handler.Request{UserCode: "TEST-0001", Decision: "approve", Permissions: new([]string{"unkey:v1:" + h.Resources().UserWorkspace.ID + ":rootKeys/*#read"})})
	require.Equal(t, http.StatusUnauthorized, res.Status, res.RawBody)
}
