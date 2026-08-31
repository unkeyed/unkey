package workos

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	rbacpermissions "github.com/unkeyed/unkey/pkg/rbac/permissions"
)

var workOSPermissionSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_:\.\*-]*[a-z0-9\*]$`)

// TestTranslatePermissions guarantees WorkOS slugs are translated at face value
// and unknown or malformed strings are skipped instead of normalized.
func TestTranslatePermissions(t *testing.T) {
	t.Parallel()

	result := translatePermissions("ws_123", []string{
		"keys:write",
		"keys:write",
		" keys:write ",
		"keys:read",
		"keyspaces:read",
		"keys:verify",
		"keys:delete",
		"identities:write",
		"identities:read",
		"identities:delete",
		"admin:*",
		"unknown:permission",
		"malformed",
		"",
	})
	require.Equal(t, []string{
		"unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#write",
		"unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#write",
		"unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#read",
		"unkey:v1:ws_123:projects/*/keyspaces/*#read",
		"unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#verify",
		"unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#delete",
		"unkey:v1:ws_123:projects/*/identities/*#write",
		"unkey:v1:ws_123:projects/*/identities/*#read",
		"unkey:v1:ws_123:projects/*/identities/*#delete",
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
			name: "key write",
			in:   "keys:write",
			want: "unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#write",
		},
		{
			name: "key read",
			in:   "keys:read",
			want: "unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#read",
		},
		{
			name: "key verify",
			in:   "keys:verify",
			want: "unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#verify",
		},
		{
			name: "key decrypt",
			in:   "keys:decrypt",
			want: "unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#decrypt",
		},
		{
			name: "key delete",
			in:   "keys:delete",
			want: "unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#delete",
		},
		{
			name: "identity write",
			in:   "identities:write",
			want: "unkey:v1:ws_123:projects/*/identities/*#write",
		},
		{
			name: "identity read",
			in:   "identities:read",
			want: "unkey:v1:ws_123:projects/*/identities/*#read",
		},
		{
			name: "identity delete",
			in:   "identities:delete",
			want: "unkey:v1:ws_123:projects/*/identities/*#delete",
		},
		{
			name: "domain write",
			in:   "domains:write",
			want: "unkey:v1:ws_123:projects/*/apps/*/environments/*/domains/*#write",
		},
		{
			name: "domain read",
			in:   "domains:read",
			want: "unkey:v1:ws_123:projects/*/apps/*/environments/*/domains/*#read",
		},
		{
			name: "domain delete",
			in:   "domains:delete",
			want: "unkey:v1:ws_123:projects/*/apps/*/environments/*/domains/*#delete",
		},
		{
			name: "gateway policies read",
			in:   "gateway_policies:read",
			want: "unkey:v1:ws_123:projects/*/apps/*/environments/*/gateway/policies/*#read",
		},
		{
			name: "gateway policies write",
			in:   "gateway_policies:write",
			want: "unkey:v1:ws_123:projects/*/apps/*/environments/*/gateway/policies/*#write",
		},
		{
			name: "gateway policies delete",
			in:   "gateway_policies:delete",
			want: "unkey:v1:ws_123:projects/*/apps/*/environments/*/gateway/policies/*#delete",
		},
		{
			name: "environment variables delete",
			in:   "environment_variables:delete",
			want: "unkey:v1:ws_123:projects/*/apps/*/environments/*/variables/*#delete",
		},
		{
			name: "admin",
			in:   "admin:*",
			want: "unkey:v1:ws_123:**#*",
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

// TestPermissionMappingsCoverCatalog guarantees WorkOS defines one slug for
// every resource and action in the public permission catalog.
func TestPermissionMappingsCoverCatalog(t *testing.T) {
	t.Parallel()

	type catalogResource struct {
		slugPrefix string
		resource   string
		actions    []string
	}

	catalog := []catalogResource{
		{slugPrefix: "github_apps", resource: "github/apps/*", actions: []string{"read", "write", "delete"}},
		{slugPrefix: "projects", resource: "projects/*", actions: []string{"read", "write", "delete"}},
		{slugPrefix: "apps", resource: "projects/*/apps/*", actions: []string{"read", "write", "delete"}},
		{slugPrefix: "environments", resource: "projects/*/apps/*/environments/*", actions: []string{"read", "write", "delete"}},
		{slugPrefix: "deployments", resource: "projects/*/apps/*/environments/*/deployments/*", actions: []string{"read", "write", "delete"}},
		{slugPrefix: "deployment_logs", resource: "projects/*/apps/*/environments/*/deployments/*/logs", actions: []string{"read"}},
		{slugPrefix: "domains", resource: "projects/*/apps/*/environments/*/domains/*", actions: []string{"read", "write", "delete"}},
		{slugPrefix: "environment_variables", resource: "projects/*/apps/*/environments/*/variables/*", actions: []string{"read", "write", "delete"}},
		{slugPrefix: "gateway_logs", resource: "projects/*/apps/*/environments/*/gateway/logs", actions: []string{"read"}},
		{slugPrefix: "gateway_policies", resource: "projects/*/apps/*/environments/*/gateway/policies/*", actions: []string{"read", "write", "delete"}},
		{slugPrefix: "identities", resource: "projects/*/identities/*", actions: []string{"read", "write", "delete"}},
		{slugPrefix: "keyspaces", resource: "projects/*/keyspaces/*", actions: []string{"read", "write", "delete"}},
		{slugPrefix: "keyspace_logs", resource: "projects/*/keyspaces/*/logs", actions: []string{"read"}},
		{slugPrefix: "keys", resource: "projects/*/keyspaces/*/keys/*", actions: []string{"read", "write", "delete", "decrypt", "verify"}},
		{slugPrefix: "ratelimit_namespaces", resource: "projects/*/ratelimits/namespaces/*", actions: []string{"read", "write", "delete", "limit"}},
		{slugPrefix: "ratelimit_logs", resource: "projects/*/ratelimits/namespaces/*/logs", actions: []string{"read"}},
		{slugPrefix: "ratelimit_overrides", resource: "projects/*/ratelimits/namespaces/*/overrides/*", actions: []string{"read", "write", "delete"}},
		{slugPrefix: "roles", resource: "projects/*/rbac/roles/*", actions: []string{"read", "write", "delete"}},
		{slugPrefix: "permissions", resource: "projects/*/rbac/permissions/*", actions: []string{"read", "write", "delete"}},
	}

	expected := make(map[string]string)
	for _, resource := range catalog {
		for _, action := range resource.actions {
			expected[resource.resource+"#"+action] = resource.slugPrefix + ":" + action
		}
	}

	actual := make(map[string]string)
	for slug, mapping := range permissionMappings {
		if slug == "admin:*" {
			continue
		}

		require.Len(t, mapping.permissions, 1, "catalog permission %q must grant exactly one resource action", slug)
		grant := mapping.permissions[0]
		key := grant.resource + "#" + string(grant.action)
		require.NotContains(t, actual, key, "resource action %q must have only one WorkOS slug", key)
		actual[key] = slug
	}

	require.Equal(t, expected, actual)
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
			{resource: "projects/*", action: action(rbacpermissions.Read)},
			{resource: "projects/*/apps/*", action: action(rbacpermissions.Read)},
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
		"unkey:v1:ws_123:projects/*#read",
		"unkey:v1:ws_123:projects/*/apps/*#read",
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
