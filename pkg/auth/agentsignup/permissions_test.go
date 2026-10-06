package agentsignup

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPermissionsForRolesGrantsOnlyTheAllowlist(t *testing.T) {
	t.Parallel()

	permissions := permissionsForRoles("ws_123", []string{Role})
	require.Len(t, permissions, 8)
	for _, permission := range permissions {
		require.True(t, strings.HasPrefix(permission, "unkey:v1:ws_123:"))
		require.NotContains(t, permission, "decrypt")
		require.NotContains(t, permission, "rootKeys")
		require.NotContains(t, permission, "**")
		require.True(t, PermissionAllowed("ws_123", permission))
	}
	require.False(t, PermissionAllowed("ws_other", permissions[0]))
	require.False(t, PermissionAllowed("ws_123", "unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#decrypt"))
	require.False(t, PermissionAllowed("ws_123", "unkey:v1:ws_123:**#*"))
	require.False(t, PermissionAllowed("ws_123", "unkey:v1:ws_123:rootKeys/*#write"))
}

func TestPermissionsForRolesIgnoresAdmin(t *testing.T) {
	t.Parallel()

	require.Empty(t, permissionsForRoles("ws_123", []string{"admin"}))
	require.Empty(t, permissionsForRoles("ws_123", []string{"developer", "viewer"}))
	require.Empty(t, permissionsForRoles("ws_123", nil))

	withAdmin := permissionsForRoles("ws_123", []string{"admin", Role})
	require.Equal(t, permissionsForRoles("ws_123", []string{Role}), withAdmin)
	for _, permission := range withAdmin {
		require.NotContains(t, permission, "**")
	}
}
