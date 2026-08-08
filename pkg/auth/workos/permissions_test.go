package workos

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var workOSPermissionSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_:\.\*-]*[a-z0-9\*]$`)

// TestTranslatePermissions guarantees WorkOS slugs are translated at face value
// and unknown or malformed strings are skipped instead of normalized.
func TestTranslatePermissions(t *testing.T) {
	t.Parallel()

	result := translatePermissions("ws_123", []string{
		"keys:create",
		"keys:create",
		" keys:create ",
		"keys:read",
		"keys:update",
		"keys:verify",
		"keys:delete",
		"identities:create",
		"identities:read",
		"identities:update",
		"identities:delete",
		"admin:*",
		"unknown:permission",
		"malformed",
		"",
	})
	require.Equal(t, []string{
		"unkey:v1:ws_123:projects/*/keyspaces/**#create_key",
		"unkey:v1:ws_123:projects/*/keyspaces/**#create_key",
		"unkey:v1:ws_123:projects/*/keyspaces/**#read_key",
		"unkey:v1:ws_123:projects/*/keyspaces/**#read_keyspace",
		"unkey:v1:ws_123:projects/*/keyspaces/**#update_key",
		"unkey:v1:ws_123:projects/*/keyspaces/**#verify_key",
		"unkey:v1:ws_123:projects/*/keyspaces/**#delete_key",
		"unkey:v1:ws_123:projects/*/identities/*#create_identity",
		"unkey:v1:ws_123:projects/*/identities/*#read_identity",
		"unkey:v1:ws_123:projects/*/identities/*#update_identity",
		"unkey:v1:ws_123:projects/*/identities/*#delete_identity",
		"unkey:v1:ws_123:**#*",
	}, result)
}

// TestTranslatePermissionsKnownMappings guarantees representative WorkOS slugs
// map to the canonical Unkey resources and actions used by RBAC checks.
func TestTranslatePermissionsKnownMappings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "key create",
			in:   "keys:create",
			want: "unkey:v1:ws_123:projects/*/keyspaces/**#create_key",
		},
		{
			name: "key read",
			in:   "keys:read",
			want: "unkey:v1:ws_123:projects/*/keyspaces/**#read_key",
		},
		{
			name: "key update",
			in:   "keys:update",
			want: "unkey:v1:ws_123:projects/*/keyspaces/**#update_key",
		},
		{
			name: "key verify",
			in:   "keys:verify",
			want: "unkey:v1:ws_123:projects/*/keyspaces/**#verify_key",
		},
		{
			name: "key decrypt",
			in:   "keys:decrypt",
			want: "unkey:v1:ws_123:projects/*/keyspaces/**#decrypt_key",
		},
		{
			name: "key delete",
			in:   "keys:delete",
			want: "unkey:v1:ws_123:projects/*/keyspaces/**#delete_key",
		},
		{
			name: "identity create",
			in:   "identities:create",
			want: "unkey:v1:ws_123:projects/*/identities/*#create_identity",
		},
		{
			name: "identity read",
			in:   "identities:read",
			want: "unkey:v1:ws_123:projects/*/identities/*#read_identity",
		},
		{
			name: "identity update",
			in:   "identities:update",
			want: "unkey:v1:ws_123:projects/*/identities/*#update_identity",
		},
		{
			name: "identity delete",
			in:   "identities:delete",
			want: "unkey:v1:ws_123:projects/*/identities/*#delete_identity",
		},
		{
			name: "admin",
			in:   "admin:*",
			want: "unkey:v1:ws_123:**#*",
		},
		{
			name: "deployment promote",
			in:   "deployments:promote",
			want: "unkey:v1:ws_123:projects/*/apps/*/environments/*#promote_deployment",
		},
		{
			name: "deployment rollback",
			in:   "deployments:rollback",
			want: "unkey:v1:ws_123:projects/*/apps/*/environments/*#rollback_deployment",
		},
		{
			name: "environment variables",
			in:   "environments:create_variables",
			want: "unkey:v1:ws_123:projects/*/apps/*/environments/*#create_variables",
		},
		{
			name: "gateway policy update",
			in:   "gateway:update_policy",
			want: "unkey:v1:ws_123:projects/*/apps/*/environments/*/gateway/policies/*#update_policy",
		},
		{
			name: "rate limit override create",
			in:   "ratelimits:create_override",
			want: "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/overrides/*#create_override",
		},
		{
			name: "permission definition create",
			in:   "permissions:create",
			want: "unkey:v1:ws_123:projects/*/rbac/permissions/*#create_permission",
		},
		{
			name: "workspace GitHub installation",
			in:   "workspaces:install_github",
			want: "unkey:v1:ws_123:workspace#install_github",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := translatePermissions("ws_123", []string{tt.in})
			require.Contains(t, result, tt.want)
		})
	}
}

// TestTranslatePermissionsSupportsMultipleGrants guarantees one WorkOS slug can
// expand into multiple canonical Unkey permissions.
func TestTranslatePermissionsSupportsMultipleGrants(t *testing.T) {
	const slug = "test:multi"

	original, existed := permissionMappings[slug]
	permissionMappings[slug] = permissionMapping{
		name:        "Test multi grant",
		description: "Used by tests to prove one WorkOS slug can grant multiple Unkey permissions.",
		permissions: []permissionGrant{
			{resource: "projects/*", action: "read_project"},
			{resource: "projects/*/apps/*", action: "read_app"},
		},
	}
	t.Cleanup(func() {
		if existed {
			permissionMappings[slug] = original
			return
		}
		delete(permissionMappings, slug)
	})

	result := translatePermissions("ws_123", []string{slug})
	require.Equal(t, []string{
		"unkey:v1:ws_123:projects/*#read_project",
		"unkey:v1:ws_123:projects/*/apps/*#read_app",
	}, result)
}

// TestTranslatePermissionsGolden guarantees the full WorkOS slug mapping stays
// intentional when permissions are added, removed, or retargeted.
func TestTranslatePermissionsGolden(t *testing.T) {
	t.Parallel()

	permissions := sortedPermissionSlugs()

	translated := translatePermissions("ws_123", permissions)

	got := strings.Join(translated, "\n") + "\n"
	want, err := os.ReadFile("testdata/permissions.golden")
	require.NoError(t, err)
	require.Equal(t, string(want), got)
}

// TestSortedPermissionSlugs guarantees generated definitions consume a
// deterministic list from the same source of truth used by JWT permission
// translation.
func TestSortedPermissionSlugs(t *testing.T) {
	t.Parallel()

	permissions := sortedPermissionSlugs()
	require.Len(t, permissions, len(permissionMappings))
	require.True(t, slices.IsSorted(permissions))

	for _, permission := range permissions {
		_, ok := permissionMappings[permission]
		require.True(t, ok, "permission %q must exist in mapping table", permission)
	}
}

// TestPermissionDefinitionsCoverMigrationCatalog guarantees every product
// capability needed by the route migration can be assigned before routes move
// from tuple permissions to resource permissions.
func TestPermissionDefinitionsCoverMigrationCatalog(t *testing.T) {
	t.Parallel()

	want := []string{
		"admin:*",
		"apps:create",
		"apps:delete",
		"apps:read",
		"apps:update",
		"deployments:create",
		"deployments:promote",
		"deployments:read",
		"deployments:rollback",
		"deployments:start",
		"deployments:stop",
		"environments:create_variables",
		"environments:delete_variables",
		"environments:read",
		"environments:read_variables",
		"environments:update",
		"gateway:create_policies",
		"gateway:read_policies",
		"gateway:update_policy",
		"identities:create",
		"identities:delete",
		"identities:read",
		"identities:update",
		"keys:create",
		"keys:decrypt",
		"keys:delete",
		"keys:read",
		"keys:update",
		"keys:verify",
		"keyspaces:create",
		"keyspaces:delete",
		"keyspaces:read",
		"keyspaces:read_logs",
		"permissions:create",
		"permissions:delete",
		"permissions:read",
		"projects:create",
		"projects:delete",
		"projects:read",
		"projects:update",
		"ratelimits:create_namespace",
		"ratelimits:create_override",
		"ratelimits:delete_override",
		"ratelimits:limit",
		"ratelimits:read_logs",
		"ratelimits:read_overrides",
		"roles:create",
		"roles:delete",
		"roles:read",
		"workspaces:install_github",
	}

	require.Equal(t, want, sortedPermissionSlugs())
}

// TestPermissionDefinitions guarantees WorkOS display metadata is sourced from
// the same mapping as permission translation instead of being generated later.
func TestPermissionDefinitions(t *testing.T) {
	t.Parallel()

	definitions := PermissionDefinitions()
	require.Len(t, definitions, len(permissionMappings))

	slugs := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		slugs = append(slugs, definition.Slug)

		mapping, ok := permissionMappings[definition.Slug]
		require.True(t, ok, "permission %q must exist in mapping table", definition.Slug)
		require.NotEmpty(t, definition.Name, "permission %q must have a WorkOS name", definition.Slug)
		require.NotEmpty(t, definition.Description, "permission %q must have a WorkOS description", definition.Slug)
		require.Equal(t, mapping.name, definition.Name)
		require.Equal(t, mapping.description, definition.Description)
		require.NotEmpty(t, mapping.permissions, "permission %q must grant at least one Unkey permission", definition.Slug)
		for _, permission := range mapping.permissions {
			require.NotEmpty(t, permission.resource, "permission %q must not grant an empty resource", definition.Slug)
			require.NotEmpty(t, permission.action, "permission %q must not grant an empty action", definition.Slug)
		}
	}

	require.True(t, slices.IsSorted(slugs))
}

// TestWorkOSPermissionSlugs guarantees every configured slug satisfies WorkOS'
// character and length limits before it is created outside the UI.
func TestWorkOSPermissionSlugs(t *testing.T) {
	t.Parallel()

	for slug := range permissionMappings {
		require.LessOrEqual(t, len(slug), 48, "permission slug %q exceeds WorkOS length limit", slug)
		require.True(t, workOSPermissionSlugPattern.MatchString(slug), "permission slug %q violates WorkOS slug rules", slug)
	}
}
