package workos

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/zen"
)

type stubResolver struct {
	principal *principal.Principal
	err       error
}

// Resolve returns the configured principal and error.
func (s stubResolver) Resolve(context.Context, *zen.Session) (*principal.Principal, error) {
	return s.principal, s.err
}

// TestResolverWithRolesMapsRoles guarantees configured WorkOS JWT resolvers
// build API permissions only from the verified roles claim.
func TestResolverWithRolesMapsRoles(t *testing.T) {
	t.Parallel()

	resolver := resolverWithRoles{
		resolver: stubResolver{
			principal: &principal.Principal{
				Type:        principal.TypeJWT,
				WorkspaceID: "ws_123",
				Source: principal.JWTSource{
					Roles: []string{"viewer", "admin"},
				},
			},
		},
	}

	p, err := resolver.Resolve(context.Background(), nil)
	require.NoError(t, err)
	require.Contains(t, p.Permissions, "unkey:v1:ws_123:projects/*#read")
	require.Contains(t, p.Permissions, "unkey:v1:ws_123:**#*")
}

// TestResolverWithRolesLeavesNonJWTPrincipalsAlone guarantees provider role
// mapping cannot change root-key or portal-session permissions.
func TestResolverWithRolesLeavesNonJWTPrincipalsAlone(t *testing.T) {
	t.Parallel()

	resolver := resolverWithRoles{
		resolver: stubResolver{
			principal: &principal.Principal{
				Type:        principal.TypeAPIKey,
				WorkspaceID: "ws_123",
				Permissions: []string{"api.*.read_api"},
			},
		},
	}

	p, err := resolver.Resolve(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, []string{"api.*.read_api"}, p.Permissions)
}

// TestResolverWithRolesYieldsAndPropagatesErrors guarantees provider role
// mapping preserves nil-principal and error behavior from token verification.
func TestResolverWithRolesYieldsAndPropagatesErrors(t *testing.T) {
	t.Parallel()

	resolver := resolverWithRoles{resolver: stubResolver{}}
	p, err := resolver.Resolve(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, p)

	wantErr := errors.New("boom")
	resolver = resolverWithRoles{resolver: stubResolver{err: wantErr}}
	p, err = resolver.Resolve(context.Background(), nil)
	require.ErrorIs(t, err, wantErr)
	require.Nil(t, p)
}

// TestResolverWithRolesRejectsInvalidJWTPrincipal guarantees an internal JWT
// principal without JWT source data fails instead of receiving no permissions.
func TestResolverWithRolesRejectsInvalidJWTPrincipal(t *testing.T) {
	t.Parallel()

	resolver := resolverWithRoles{
		resolver: stubResolver{
			principal: &principal.Principal{Type: principal.TypeJWT},
		},
	}

	p, err := resolver.Resolve(context.Background(), nil)
	require.Error(t, err)
	require.Nil(t, p)
}
