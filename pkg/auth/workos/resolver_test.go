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

func (s stubResolver) Resolve(context.Context, *zen.Session) (*principal.Principal, error) {
	return s.principal, s.err
}

// TestResolverWithPermissionsTranslatesPermissionStrings guarantees WorkOS JWT
// principals expose canonical Unkey permissions before handlers authorize them.
func TestResolverWithPermissionsTranslatesPermissionStrings(t *testing.T) {
	t.Parallel()

	resolver := resolverWithPermissions{
		resolver: stubResolver{
			principal: &principal.Principal{
				Type:        principal.TypeJWT,
				WorkspaceID: "ws_123",
				Permissions: []string{
					"keys:create",
					"keys:create",
					"keys:encrypt",
					"malformed",
					"",
				},
			},
		},
	}

	principal, err := resolver.Resolve(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, []string{
		"unkey:v1:ws_123:keyspaces/*/keys/*#create_key",
		"unkey:v1:ws_123:keyspaces/*/keys/*#create_key",
		"unkey:v1:ws_123:keyspaces/*/keys/*#encrypt_key",
	}, principal.Permissions)
}

// TestResolverWithPermissionsLeavesNonJWTPrincipalsAlone guarantees the WorkOS
// wrapper cannot rewrite root-key or portal-session permissions.
func TestResolverWithPermissionsLeavesNonJWTPrincipalsAlone(t *testing.T) {
	t.Parallel()

	resolver := resolverWithPermissions{
		resolver: stubResolver{
			principal: &principal.Principal{
				Type:        principal.TypeAPIKey,
				WorkspaceID: "ws_123",
				Permissions: []string{"keys:create"},
			},
		},
	}

	principal, err := resolver.Resolve(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, []string{"keys:create"}, principal.Permissions)
}

// TestResolverWithPermissionsYieldsAndPropagatesErrors guarantees the wrapper
// preserves resolver nil-principal and error behavior before translation.
func TestResolverWithPermissionsYieldsAndPropagatesErrors(t *testing.T) {
	t.Parallel()

	resolver := resolverWithPermissions{resolver: stubResolver{}}
	principal, err := resolver.Resolve(context.Background(), nil)
	require.NoError(t, err)
	require.Nil(t, principal)

	wantErr := errors.New("boom")
	resolver = resolverWithPermissions{resolver: stubResolver{err: wantErr}}
	principal, err = resolver.Resolve(context.Background(), nil)
	require.ErrorIs(t, err, wantErr)
	require.Nil(t, principal)
}
