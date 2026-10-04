package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

func TestRBAC_EvaluatePermissions(t *testing.T) {
	tests := []struct {
		name        string
		query       PermissionQuery
		permissions []string
		wantValid   bool
	}{
		{
			name:        "Simple permission check (Pass)",
			query:       S("documents.read"),
			permissions: []string{"documents.read", "documents.write", "roles.read"},
			wantValid:   true,
		},
		{
			name:        "Simple permission check (Fail)",
			query:       S("invoices.read"),
			permissions: []string{"documents.read", "documents.write", "roles.read"},
			wantValid:   false,
		},
		{
			name: "AND of two permissions (Pass)",
			query: And(
				S("documents.read"),
				S("documents.write"),
			),
			permissions: []string{"documents.read", "documents.write", "roles.read"},
			wantValid:   true,
		},
		{
			name: "OR of two permissions (Pass)",
			query: Or(
				S("documents.read"),
				S("invoices.read"),
			),
			permissions: []string{"documents.read", "documents.write", "roles.read"},
			wantValid:   true,
		},
		{
			name: "Complex combination (Pass)",
			query: And(
				S("documents.read"),
				Or(
					S("documents.write"),
					S("roles.read"),
				),
			),
			permissions: []string{"documents.read", "documents.write", "roles.read"},
			wantValid:   true,
		},
		{
			name:        "Asterisk permission literal match (Pass)",
			query:       S("documents.*"),
			permissions: []string{"documents.*", "documents.read", "documents.write"},
			wantValid:   true,
		},
		{
			name:        "Asterisk permission NOT wildcard (Fail)",
			query:       S("documents.*"),
			permissions: []string{"documents.read", "documents.write", "documents.delete"},
			wantValid:   false,
		},
		{
			name:        "Asterisk within a permission remains literal (Fail)",
			query:       S("documents.*.read"),
			permissions: []string{"documents.contract.read"},
			wantValid:   false,
		},
		{
			name: "Complex query with asterisk permissions",
			query: Or(
				S("documents.*"),
				S("documents.read"),
			),
			permissions: []string{"documents.read"},
			wantValid:   true,
		},
		{
			name:        "Permission with colon namespace (Pass)",
			query:       S("system:admin:read"),
			permissions: []string{"system:admin:read", "system:admin:write"},
			wantValid:   true,
		},
		{
			name:        "Permission with colon namespace (Fail)",
			query:       S("system:admin:write"),
			permissions: []string{"system:admin:read", "user:basic:read"},
			wantValid:   false,
		},
		{
			name: "Complex query with colons and other characters",
			query: And(
				S("system:admin:*"),
				Or(
					S("api_v2:read"),
					S("api-v2:write"),
				),
			),
			permissions: []string{"system:admin:*", "api_v2:read", "user:basic:read"},
			wantValid:   true,
		},
	}

	rbac := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := rbac.EvaluatePermissions(tt.query, tt.permissions)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result.Valid != tt.wantValid {
				t.Errorf("want valid=%v, got valid=%v, message=%s",
					tt.wantValid, result.Valid, result.Message)
			}
		})
	}
}

func TestRBAC_ORFailureMessageDoesNotRevealGrantedPermissions(t *testing.T) {
	t.Parallel()

	query := Or(
		S("documents.read"),
		S("documents.write"),
	)
	granted := []string{"billing.admin"}

	result, err := New().EvaluatePermissions(query, granted)
	require.NoError(t, err)
	require.False(t, result.Valid)
	require.Equal(
		t,
		"Missing one of these permissions: documents.read or documents.write",
		result.Message,
	)
	require.NotContains(t, result.Message, "have:")
	require.NotContains(t, result.Message, "billing.admin")
	require.NotContains(t, result.Message, "{")
	require.NotContains(t, result.Message, "}")
}

// TestRBAC_ProjectUrnPermissions guarantees the canonical URN form scopes
// permissions to one project, all projects in a workspace, and never across
// workspaces.
func TestRBAC_ProjectUrnPermissions(t *testing.T) {
	t.Parallel()

	query := U(
		urn.New().Workspace("ws_1").Project("proj_abc"),
		permissions.Write,
	)
	require.Equal(t, "unkey:v1:ws_1:projects/proj_abc#write", query.Value)

	tests := []struct {
		name        string
		permissions []string
		wantValid   bool
	}{
		{
			name:        "exact project grant",
			permissions: []string{"unkey:v1:ws_1:projects/proj_abc#write"},
			wantValid:   true,
		},
		{
			name:        "workspace-wide project grant",
			permissions: []string{"unkey:v1:ws_1:projects/*#write"},
			wantValid:   true,
		},
		{
			name:        "admin grant",
			permissions: []string{"unkey:v1:ws_1:**#*"},
			wantValid:   true,
		},
		{
			name:        "other project grant",
			permissions: []string{"unkey:v1:ws_1:projects/proj_xyz#write"},
			wantValid:   false,
		},
		{
			name:        "other workspace grant",
			permissions: []string{"unkey:v1:ws_2:projects/proj_abc#write"},
			wantValid:   false,
		},
		{
			name:        "read grant does not satisfy write",
			permissions: []string{"unkey:v1:ws_1:projects/*#read"},
			wantValid:   false,
		},
	}

	rbac := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := rbac.EvaluatePermissions(query, tt.permissions)
			require.NoError(t, err)
			require.Equal(t, tt.wantValid, result.Valid, result.Message)
		})
	}
}

// TestRBAC_ProjectUrnSubtreeGrantCoversKeys guarantees a project subtree grant
// reaches key resources below the project.
func TestRBAC_ProjectUrnSubtreeGrantCoversKeys(t *testing.T) {
	t.Parallel()

	required := UnkeyPermission{
		Resource: urn.V1{WorkspaceID: "ws_1", Resource: "projects/proj_abc/keyspaces/ks_1/keys/key_1"},
		Action:   ActionType(permissions.Read.String()),
	}
	granted := UnkeyPermission{
		Resource: urn.V1{WorkspaceID: "ws_1", Resource: "projects/proj_abc/**"},
		Action:   ActionType(permissions.Read.String()),
	}

	require.True(t, permissionCovers(required, granted))
}

// TestRBAC_ProjectActionWildcardRejected guarantees a project-scoped action
// wildcard cannot be granted; only the global "**" resource takes "*".
func TestRBAC_ProjectActionWildcardRejected(t *testing.T) {
	t.Parallel()

	_, err := parseUrnPermission("unkey:v1:ws_1:projects/*#*")
	require.ErrorIs(t, err, errInvalidURNPermission)
}

// TestRBAC_RecursivePermissionAppliesActionToResourceAndDescendants guarantees a
// recursive permission uses the same simple action for its base and descendants.
func TestRBAC_RecursivePermissionAppliesActionToResourceAndDescendants(t *testing.T) {
	t.Parallel()

	app := urn.New().Workspace("ws_1").Project("proj_1").App("app_1")
	environment := app.Environment("env_1")
	pattern := urn.V1{
		WorkspaceID: "ws_1",
		Resource:    "projects/proj_1/apps/app_1/**",
	}
	permission := U(pattern, permissions.Write).Value
	require.Equal(t, "unkey:v1:ws_1:projects/proj_1/apps/app_1/**#write", permission)

	evaluator := New()
	for _, query := range []PermissionQuery{
		U(app, permissions.Write),
		U(environment, permissions.Write),
	} {
		result, err := evaluator.EvaluatePermissions(
			query,
			[]string{permission},
		)
		require.NoError(t, err)
		require.True(t, result.Valid, "recursive app permission must cover %s", query.Value)
	}
}
