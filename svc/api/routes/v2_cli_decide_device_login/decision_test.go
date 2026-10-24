package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice/clidevicetest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_decide_device_login"
)

func TestDecisionRejectsInvalidInputWithoutMiddleware(t *testing.T) {
	workspaceID := uid.New(uid.WorkspacePrefix)
	p := &principal.Principal{
		Version:               principal.Version,
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: uid.New(uid.TestPrefix), Type: principal.SubjectTypeUser},
		Source:                principal.JWTSource{Roles: []string{"admin"}},
		AuthorizedWorkspaceID: workspaceID,
		Permissions:           workos.PermissionsForRoles(workspaceID, []string{"admin"}),
	}
	cases := []struct {
		name string
		req  handler.Request
	}{
		{name: "missing decision", req: handler.Request{UserCode: "ABCD-EFGH"}},
		{name: "unknown decision", req: handler.Request{UserCode: "ABCD-EFGH", Decision: "unknown"}},
		{name: "approve without permissions", req: handler.Request{UserCode: "ABCD-EFGH", Decision: "approve"}},
		{name: "approve empty permissions", req: handler.Request{UserCode: "ABCD-EFGH", Decision: "approve", Permissions: new([]string{})}},
		{name: "deny empty permissions", req: handler.Request{UserCode: "ABCD-EFGH", Decision: "deny", Permissions: new([]string{})}},
		{name: "deny with permissions", req: handler.Request{UserCode: "ABCD-EFGH", Decision: "deny", Permissions: new([]string{"unkey:v1:" + workspaceID + ":rootKeys/*#read"})}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(tc.req)
			require.NoError(t, err)
			route := &handler.Handler{}
			req := httptest.NewRequest(route.Method(), route.Path(), bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			session := &zen.Session{}
			require.NoError(t, session.Init(httptest.NewRecorder(), req, 1<<20))
			session.SetPrincipal(p)
			err = route.Handle(t.Context(), session)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.App.Validation.InvalidInput.URN(), code)
		})
	}
}

func TestDecisionApprovesOrDeniesTheLogin(t *testing.T) {
	for _, decision := range []openapi.V2CliDecideDeviceLoginRequestBodyDecision{"approve", "deny"} {
		t.Run(string(decision), func(t *testing.T) {
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
			started, err := devices.Start(t.Context(), clidevice.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.60"})
			require.NoError(t, err)
			req := handler.Request{UserCode: started.UserCode, Decision: decision}
			if decision == "approve" {
				req.Permissions = new([]string{"unkey:v1:" + workspace.ID + ":rootKeys/*#read"})
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer " + uid.New(uid.TestPrefix)},
			}, req)
			require.Equal(t, http.StatusOK, res.Status, res.RawBody)
			result, err := devices.Poll(t.Context(), clidevice.PollRequest{LoginID: started.LoginID, RemoteIP: "203.0.113.61"})
			require.NoError(t, err)
			if decision == "approve" {
				require.Equal(t, clidevice.PollComplete, result.Status)
				require.NotEmpty(t, result.AccessToken)
			} else {
				require.Equal(t, clidevice.PollDenied, result.Status)
				require.Empty(t, result.AccessToken)
			}
		})
	}
}
