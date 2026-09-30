package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_create_key"
)

func TestCreateRequiresPermissionRegardlessOfPrincipalType(t *testing.T) {
	for _, p := range []principal.Principal{
		{Type: principal.TypeAPIKey, Source: principal.KeySource{}},
		{Type: principal.TypePortalSession, Source: principal.PortalSessionSource{}},
		{Type: principal.TypeJWT, Source: principal.JWTSource{Roles: []string{"admin"}}},
		{Type: principal.TypeJWT, Source: principal.JWTSource{Roles: []string{"developer"}}},
	} {
		t.Run(string(p.Type), func(t *testing.T) {
			p.AuthorizedWorkspaceID = "ws_customer"
			p.Subject = principal.Subject{ID: "user_admin", Type: principal.SubjectTypeUser}
			p.Permissions = []string{"unkey:v1:ws_customer:rootKeys/*#read"}
			s := &zen.Session{}
			s.SetPrincipal(&p)
			err := (&handler.Handler{}).Handle(context.Background(), s)
			require.Error(t, err)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.Auth.Authorization.InsufficientPermissions.URN(), code)
		})
	}
}

func TestCreateRejectsInvalidBearerThroughAuthenticationMiddleware(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{}
	h.Register(route)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer invalid"}, "Content-Type": {"application/json"},
	}, handler.Request{Permissions: []string{"*"}})
	require.Equal(t, http.StatusUnauthorized, res.Status)
}

func TestCreateRejectsLegacyCreationPermissions(t *testing.T) {
	for _, grant := range []string{"*", "workspace.*.create_root_key"} {
		p := &principal.Principal{
			Type: principal.TypeAPIKey, Subject: principal.Subject{ID: "key_caller", Type: principal.SubjectTypeRootKey},
			Source: principal.KeySource{}, AuthorizedWorkspaceID: "ws_customer", Permissions: []string{grant},
		}
		s := &zen.Session{}
		s.SetPrincipal(p)
		err := (&handler.Handler{}).Handle(t.Context(), s)
		require.Error(t, err)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, codes.Auth.Authorization.InsufficientPermissions.URN(), code)
	}
}
