package agentsignup

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/zen"
)

type stubResolver struct {
	principal *principal.Principal
	err       error
}

func (s stubResolver) Resolve(context.Context, *zen.Session) (*principal.Principal, error) {
	return s.principal, s.err
}

func TestResolverMapsAgentSignupRoleOnly(t *testing.T) {
	t.Parallel()

	resolver := NewRoleMappingResolver(stubResolver{
		principal: &principal.Principal{
			Type:                  principal.TypeJWT,
			AuthorizedWorkspaceID: "ws_123",
			Source:                principal.JWTSource{Roles: []string{Role}},
			Permissions:           []string{"unkey:v1:ws_123:**#*"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	resolved, err := resolver.Resolve(context.Background(), sess)
	require.NoError(t, err)
	require.NotContains(t, resolved.Permissions, "unkey:v1:ws_123:**#*")
	require.Contains(t, resolved.Permissions, "unkey:v1:ws_123:projects/*/keyspaces/*#read")
	require.NotContains(t, strings.Join(resolved.Permissions, "\n"), "decrypt")
}

func TestResolverDropsAdminRole(t *testing.T) {
	t.Parallel()

	resolver := NewRoleMappingResolver(stubResolver{
		principal: &principal.Principal{
			Type:                  principal.TypeJWT,
			AuthorizedWorkspaceID: "ws_123",
			Source:                principal.JWTSource{Roles: []string{"admin"}},
			Permissions:           []string{"kept-only-if-unmapped"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))

	resolved, err := resolver.Resolve(context.Background(), sess)
	require.NoError(t, err)
	require.Empty(t, resolved.Permissions)
}
