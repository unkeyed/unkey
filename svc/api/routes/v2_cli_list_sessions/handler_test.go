package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice/clidevicetest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_list_sessions"
)

func TestListSessionsWithoutBodyOrContentType(t *testing.T) {
	h := testutil.NewHarness(t)
	devices := clidevicetest.New(h)
	route := &handler.Handler{Devices: devices}
	workspace := h.Resources().UserWorkspace
	p := &principal.Principal{
		Version:               principal.Version,
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: uid.New(uid.TestPrefix), Type: principal.SubjectTypeUser},
		Source:                principal.JWTSource{Roles: []string{"admin"}},
		AuthorizedWorkspaceID: workspace.ID,
		Permissions:           workos.PermissionsForRoles(workspace.ID, []string{"admin"}),
	}
	h.Register(route, append(h.PublicMiddleware(), func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, session *zen.Session) error {
			session.SetPrincipal(p)
			return next(ctx, session)
		}
	})...)
	started, err := devices.Start(t.Context(), clidevice.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.80"})
	require.NoError(t, err)
	permissions := []string{"unkey:v1:" + workspace.ID + ":rootKeys/*#read"}
	require.NoError(t, devices.Approve(t.Context(), clidevice.ApproveRequest{
		Principal:   p,
		UserCode:    started.UserCode,
		Permissions: permissions,
	}))
	tokens, err := devices.Poll(t.Context(), clidevice.PollRequest{LoginID: started.LoginID, RemoteIP: "203.0.113.81"})
	require.NoError(t, err)
	req := httptest.NewRequest(route.Method(), route.Path(), nil)
	req.Header.Set("Authorization", "Bearer "+uid.New(uid.TestPrefix))
	res := testutil.CallRaw[handler.Response](h, req)
	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Len(t, res.Body.Data.Sessions, 1)
	require.Equal(t, tokens.SessionID, res.Body.Data.Sessions[0].Id)
	require.Equal(t, p.Subject.ID, res.Body.Data.Sessions[0].UserId)
	require.Equal(t, permissions, res.Body.Data.Sessions[0].Permissions)
}

func TestBodylessListSessionsRejectsRootKeysAndInvalidBearers(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{Devices: clidevicetest.New(h)}
	h.Register(route)
	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "unkey:v1:"+workspace.ID+":rootKeys/*#read")
	for _, tc := range []struct {
		name   string
		bearer string
		status int
	}{
		{name: "root key", bearer: rootKey, status: http.StatusForbidden},
		{name: "invalid bearer", bearer: "invalid", status: http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(route.Method(), route.Path(), nil)
			req.Header.Set("Authorization", "Bearer "+tc.bearer)
			res := testutil.CallRaw[openapi.UnauthorizedErrorResponse](h, req)
			require.Equal(t, tc.status, res.Status, res.RawBody)
		})
	}
}
