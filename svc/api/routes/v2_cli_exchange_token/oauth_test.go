package handler_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/internal/grant/granttest"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_cli_exchange_token"
)

// TestExchangeRejectsOAuthCredentials guarantees that the CLI endpoint cannot
// be used to redeem an OAuth app's authorization code or to rotate an OAuth
// app's refresh token, and that a refused rotation leaves the session alone.
func TestExchangeRejectsOAuthCredentials(t *testing.T) {
	h := testutil.NewHarness(t)
	grants := granttest.New(h)
	route := &handler.Handler{Grants: grants}
	h.Register(route, h.PublicMiddleware()...)
	approver := granttest.Approver(h)
	permissions := []string{"unkey:v1:" + h.Resources().UserWorkspace.ID + ":rootKeys/*#read"}

	pending := granttest.PendingOAuthRequest(t, grants)
	decided := granttest.MustApprove(t, grants, grant.ApproveRequest{Principal: approver, UserCode: pending.UserCode, Permissions: permissions})
	callback, err := url.Parse(decided.RedirectTo)
	require.NoError(t, err)
	row, err := db.Query.FindDeviceAuthorizationByUserCode(t.Context(), h.DB.RW(), pending.UserCode)
	require.NoError(t, err)
	for _, loginID := range []string{callback.Query().Get("code"), row.ID} {
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, jsonHeaders, handler.Request{LoginId: &loginID})
		require.Equal(t, http.StatusBadRequest, res.Status, res.RawBody)
	}

	tokens := granttest.OAuthSession(t, grants, granttest.PendingOAuthRequest(t, grants), approver, permissions)
	res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, jsonHeaders, handler.Request{RefreshToken: &tokens.RefreshToken})
	require.Equal(t, http.StatusUnauthorized, res.Status, res.RawBody)
	session, err := db.Query.FindCLISessionByID(t.Context(), h.DB.RW(), tokens.SessionID)
	require.NoError(t, err)
	require.False(t, session.RevokedAt.Valid)
	require.False(t, session.RotatedAt.Valid)
}
