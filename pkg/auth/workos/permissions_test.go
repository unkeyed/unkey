package workos

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPermissionsForRoles guarantees roles expand into workspace-scoped
// permissions and unknown roles add nothing.
func TestPermissionsForRoles(t *testing.T) {
	t.Parallel()

	permissions := permissionsForRoles("ws_123", []string{
		"admin",
		"unknown",
		"",
	})

	require.Equal(t, []string{"unkey:v1:ws_123:**#*"}, permissions)
}

// TestPermissionsForRolesSupportsMultipleRoles guarantees role permissions are
// additive when a token contains multiple roles.
func TestPermissionsForRolesSupportsMultipleRoles(t *testing.T) {
	t.Parallel()

	permissions := permissionsForRoles("ws_123", []string{"admin", "developer", "developer"})

	require.Equal(t, "unkey:v1:ws_123:**#*", permissions[0])
	require.Contains(t, permissions, "unkey:v1:ws_123:projects/*#read")
	require.Contains(t, permissions, "unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#decrypt")
	require.Equal(t, 1, strings.Count(strings.Join(permissions, "\n"), "unkey:v1:ws_123:projects/*#read"))
}

// TestDeveloperRoleCoversProductPermissions guarantees the developer role
// includes normal product actions.
func TestDeveloperRoleCoversProductPermissions(t *testing.T) {
	t.Parallel()

	type catalogResource struct {
		resource string
		actions  []string
	}

	catalog := []catalogResource{
		{resource: "github/apps/*", actions: []string{"read", "write", "delete"}},
		{resource: "projects/*", actions: []string{"read", "write", "delete"}},
		{resource: "projects/*/portals/*", actions: []string{"read", "write", "delete"}},
		{resource: "projects/*/portals/*/sessions/*", actions: []string{"write"}},
		{resource: "projects/*/apps/*", actions: []string{"read", "write", "delete"}},
		{resource: "projects/*/apps/*/environments/*", actions: []string{"read", "write", "delete"}},
		{resource: "projects/*/apps/*/environments/*/deployments/*", actions: []string{"read", "write", "delete"}},
		{resource: "projects/*/apps/*/environments/*/deployments/*/logs", actions: []string{"read"}},
		{resource: "projects/*/apps/*/environments/*/domains/*", actions: []string{"read", "write", "delete"}},
		{resource: "projects/*/apps/*/environments/*/variables/*", actions: []string{"read", "write", "delete"}},
		{resource: "projects/*/apps/*/environments/*/gateway/logs", actions: []string{"read"}},
		{resource: "projects/*/apps/*/environments/*/gateway/policies/*", actions: []string{"read", "write", "delete"}},
		{resource: "projects/*/identities/*", actions: []string{"read", "write", "delete"}},
		{resource: "projects/*/keyspaces/*", actions: []string{"read", "write", "delete"}},
		{resource: "projects/*/keyspaces/*/logs", actions: []string{"read"}},
		{resource: "projects/*/keyspaces/*/keys/*", actions: []string{"read", "write", "delete", "decrypt", "verify"}},
		{resource: "projects/*/ratelimits/namespaces/*", actions: []string{"read", "write", "delete", "limit"}},
		{resource: "projects/*/ratelimits/namespaces/*/logs", actions: []string{"read"}},
		{resource: "projects/*/ratelimits/namespaces/*/overrides/*", actions: []string{"read", "write", "delete"}},
		{resource: "projects/*/rbac/roles/*", actions: []string{"read", "write", "delete"}},
		{resource: "projects/*/rbac/permissions/*", actions: []string{"read", "write", "delete"}},
	}

	want := make(map[string]struct{})
	for _, resource := range catalog {
		for _, action := range resource.actions {
			want[resource.resource+"#"+action] = struct{}{}
		}
	}

	got := make(map[string]struct{})
	for _, rolePermission := range rolePolicies["developer"] {
		permission := rolePermission.resource + "#" + string(rolePermission.action)
		require.NotContains(t, got, permission, "permission %q must occur once", permission)
		got[permission] = struct{}{}
	}

	require.Equal(t, want, got)
}

// TestRolePoliciesDefineDashboardRoles guarantees the API policy contains
// every dashboard role and the temporary migration alias.
func TestRolePoliciesDefineDashboardRoles(t *testing.T) {
	t.Parallel()

	require.Len(t, rolePolicies, 4)
	require.NotEmpty(t, rolePolicies["admin"])
	require.NotEmpty(t, rolePolicies["developer"])
	require.NotEmpty(t, rolePolicies["viewer"])
	require.NotEmpty(t, rolePolicies["basic_member"])
}

// TestDeveloperPermissionsGolden guarantees changes to the complete role
// expansion require an intentional golden-file update.
func TestDeveloperPermissionsGolden(t *testing.T) {
	t.Parallel()

	permissions := permissionsForRoles("ws_123", []string{"developer"})
	got := strings.Join(permissions, "\n") + "\n"

	want, err := os.ReadFile("testdata/permissions.golden")
	require.NoError(t, err)
	require.Equal(t, string(want), got)
}

// TestViewerPermissionsGolden guarantees the viewer role remains read-only and
// cannot decrypt API keys.
func TestViewerPermissionsGolden(t *testing.T) {
	t.Parallel()

	for _, permission := range rolePolicies["viewer"] {
		require.Equal(t, "read", string(permission.action))
	}

	permissions := permissionsForRoles("ws_123", []string{"viewer"})
	got := strings.Join(permissions, "\n") + "\n"

	want, err := os.ReadFile("testdata/viewer_permissions.golden")
	require.NoError(t, err)
	require.Equal(t, string(want), got)
}

// TestLegacyBasicMemberMatchesDeveloper keeps existing memberships valid during
// the role migration.
func TestLegacyBasicMemberMatchesDeveloper(t *testing.T) {
	t.Parallel()

	require.Equal(
		t,
		permissionsForRoles("ws_123", []string{"developer"}),
		permissionsForRoles("ws_123", []string{"basic_member"}),
	)
}
