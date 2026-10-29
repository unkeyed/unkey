package handler_test

import (
	"context"
	"net/http"
	"net/netip"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
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
	gethandler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_get_device_login"
)

func TestDecisionRejectsInvalidInputOverHTTP(t *testing.T) {
	h := testutil.NewHarness(t)
	workspaceID := h.Resources().UserWorkspace.ID
	route := &handler.Handler{Grants: granttest.New(h)}
	h.Register(route)
	rootKey := h.CreateRootKey(workspaceID)
	cases := []struct {
		name string
		req  handler.Request
	}{
		{name: "missing code", req: handler.Request{Decision: "deny"}},
		{name: "short code", req: handler.Request{UserCode: "ABC", Decision: "deny"}},
		{name: "long code", req: handler.Request{UserCode: strings.Repeat("A", 33), Decision: "deny"}},
		{name: "missing decision", req: handler.Request{UserCode: "ABCD-EFGH"}},
		{name: "unknown decision", req: handler.Request{UserCode: "ABCD-EFGH", Decision: "unknown"}},
		{name: "approve without permissions", req: handler.Request{UserCode: "ABCD-EFGH", Decision: "approve"}},
		{name: "approve empty permissions", req: handler.Request{UserCode: "ABCD-EFGH", Decision: "approve", Permissions: new([]string{})}},
		{name: "deny empty permissions", req: handler.Request{UserCode: "ABCD-EFGH", Decision: "deny", Permissions: new([]string{})}},
		{name: "deny with permissions", req: handler.Request{UserCode: "ABCD-EFGH", Decision: "deny", Permissions: new([]string{"unkey:v1:" + workspaceID + ":rootKeys/*#read"})}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer " + rootKey},
			}, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, res.RawBody)
			require.Equal(t, codes.App.Validation.InvalidInput.DocsURL(), res.Body.Error.Type)
		})
	}
}

func TestUserCodeRoutesShareTheUserBudget(t *testing.T) {
	h := testutil.NewHarness(t)
	devices := granttest.New(h)
	limiter := ratelimit.NewLocal(h.Clock)
	t.Cleanup(func() { require.NoError(t, limiter.Close()) })
	devices.Ratelimit = limiter
	workspace := h.Resources().UserWorkspace
	p := &principal.Principal{
		Version:               principal.Version,
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: uid.New(uid.TestPrefix), Type: principal.SubjectTypeUser},
		Source:                principal.JWTSource{Roles: []string{"admin"}},
		AuthorizedWorkspaceID: workspace.ID,
		Permissions:           workos.PermissionsForRoles(workspace.ID, []string{"admin"}),
	}
	clientIP := netip.MustParseAddr("203.0.113.70")
	middlewares := append(h.PublicMiddleware(), func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, session *zen.Session) error {
			session.SetPrincipal(p)
			session.SetClientIP(clientIP)
			return next(ctx, session)
		}
	})
	get := &gethandler.Handler{Grants: devices}
	decide := &handler.Handler{Grants: devices}
	h.Register(get, middlewares...)
	h.Register(decide, middlewares...)
	started, err := devices.Start(t.Context(), grant.StartRequest{RemoteIP: clientIP.String()})
	require.NoError(t, err)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + uid.New(uid.TestPrefix)},
	}

	for range 60 {
		res := testutil.CallRoute[gethandler.Request, gethandler.Response](h, get, headers, gethandler.Request{UserCode: started.UserCode})
		require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	}
	res := testutil.CallRoute[gethandler.Request, openapi.TooManyRequestsErrorResponse](h, get, headers, gethandler.Request{UserCode: started.UserCode})
	require.Equal(t, http.StatusTooManyRequests, res.Status, res.RawBody)
	require.Equal(t, codes.User.TooManyRequests.UserRateLimited.DocsURL(), res.Body.Error.Type)
	for _, decision := range []openapi.V2CliDecideDeviceLoginRequestBodyDecision{"approve", "deny"} {
		req := handler.Request{UserCode: started.UserCode, Decision: decision}
		if decision == "approve" {
			req.Permissions = new(p.Permissions)
		}
		res := testutil.CallRoute[handler.Request, openapi.TooManyRequestsErrorResponse](h, decide, headers, req)
		require.Equal(t, http.StatusTooManyRequests, res.Status, res.RawBody)
	}

	clientIP = netip.MustParseAddr("203.0.113.71")
	blocked := testutil.CallRoute[gethandler.Request, openapi.TooManyRequestsErrorResponse](h, get, headers, gethandler.Request{UserCode: started.UserCode})
	require.Equal(t, http.StatusTooManyRequests, blocked.Status, blocked.RawBody)
	p.Subject.ID = uid.New(uid.TestPrefix)
	allowed := testutil.CallRoute[gethandler.Request, gethandler.Response](h, get, headers, gethandler.Request{UserCode: started.UserCode})
	require.Equal(t, http.StatusOK, allowed.Status, allowed.RawBody)
	require.Equal(t, "pending", allowed.Body.Data.Status)
}

func TestDecisionApprovesOrDeniesTheLogin(t *testing.T) {
	for _, decision := range []openapi.V2CliDecideDeviceLoginRequestBodyDecision{"approve", "deny"} {
		t.Run(string(decision), func(t *testing.T) {
			h := testutil.NewHarness(t)
			devices := granttest.New(h)
			route := &handler.Handler{Grants: devices}
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
			started, err := devices.Start(t.Context(), grant.StartRequest{DeviceName: "laptop", RemoteIP: "203.0.113.60"})
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
			result, err := devices.Poll(t.Context(), grant.PollRequest{LoginID: started.LoginID, RemoteIP: "203.0.113.61"})
			require.NoError(t, err)
			if decision == "approve" {
				require.Equal(t, grant.PollComplete, result.Status)
				require.NotEmpty(t, result.AccessToken)
			} else {
				require.Equal(t, grant.PollDenied, result.Status)
				require.Empty(t, result.AccessToken)
			}
		})
	}
}
