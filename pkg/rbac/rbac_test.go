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
			query:       T(Tuple{ResourceType: Api, ResourceID: "api1", Action: ReadAPI}),
			permissions: []string{"api.api1.read_api", "api.api1.update_api", "rbac.role1.read_role"},
			wantValid:   true,
		},
		{
			name:        "Simple permission check (Fail)",
			query:       T(Tuple{ResourceType: Api, ResourceID: "api2", Action: ReadAPI}),
			permissions: []string{"api.api1.read_api", "api.api1.update_api", "rbac.role1.read_role"},
			wantValid:   false,
		},
		{
			name: "AND of two permissions (Pass)",
			query: And(
				T(Tuple{ResourceType: Api, ResourceID: "api1", Action: ReadAPI}),
				T(Tuple{ResourceType: Api, ResourceID: "api1", Action: UpdateAPI}),
			),
			permissions: []string{"api.api1.read_api", "api.api1.update_api", "rbac.role1.read_role"},
			wantValid:   true,
		},
		{
			name: "OR of two permissions (Pass)",
			query: Or(
				T(Tuple{ResourceType: Api, ResourceID: "api1", Action: ReadAPI}),
				T(Tuple{ResourceType: Api, ResourceID: "api2", Action: ReadAPI}),
			),
			permissions: []string{"api.api1.read_api", "api.api1.update_api", "rbac.role1.read_role"},
			wantValid:   true,
		},
		{
			name: "Complex combination (Pass)",
			query: And(
				T(Tuple{ResourceType: Api, ResourceID: "api1", Action: ReadAPI}),
				Or(
					T(Tuple{ResourceType: Api, ResourceID: "api1", Action: UpdateAPI}),
					T(Tuple{ResourceType: Rbac, ResourceID: "role1", Action: ReadRole}),
				),
			),
			permissions: []string{"api.api1.read_api", "api.api1.update_api", "rbac.role1.read_role"},
			wantValid:   true,
		},
		{
			name:        "Asterisk permission literal match (Pass)",
			query:       S("api.*"),
			permissions: []string{"api.*", "api.read", "api.write"},
			wantValid:   true,
		},
		{
			name:        "Asterisk permission NOT wildcard (Fail)",
			query:       S("api.*"),
			permissions: []string{"api.read", "api.write", "api.delete"},
			wantValid:   false,
		},
		{
			name:        "Tuple wildcard remains literal (Fail)",
			query:       S("api.*.read_key"),
			permissions: []string{"api.key_123.read_key"},
			wantValid:   false,
		},
		{
			name: "Complex query with asterisk permissions",
			query: Or(
				S("api.*"),
				S("api.read"),
			),
			permissions: []string{"api.read"},
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
		S("api.*.verify_key"),
		S("api.api_requested.verify_key"),
	)
	granted := []string{"api.api_secret.read_api"}

	result, err := New().EvaluatePermissions(query, granted)
	require.NoError(t, err)
	require.False(t, result.Valid)
	require.Equal(
		t,
		"Missing one of these permissions: api.*.verify_key or api.api_requested.verify_key",
		result.Message,
	)
	require.NotContains(t, result.Message, "have:")
	require.NotContains(t, result.Message, "api.api_secret.read_api")
	require.NotContains(t, result.Message, "{")
	require.NotContains(t, result.Message, "}")
}

// TestRBAC_PortalTuplePermissions guarantees portal permissions work in the
// legacy tuple form, where strings match literally and "*" is not expanded.
func TestRBAC_PortalTuplePermissions(t *testing.T) {
	t.Parallel()

	tuple := Tuple{ResourceType: Portal, ResourceID: "pc_abc", Action: CreatePortalSession}
	require.Equal(t, "portal.pc_abc.create_portal_session", tuple.String())

	parsed, err := TupleFromString("portal.pc_abc.create_portal_session")
	require.NoError(t, err)
	require.Equal(t, tuple, parsed)

	tests := []struct {
		name        string
		query       PermissionQuery
		permissions []string
		wantValid   bool
	}{
		{
			name:        "exact tuple grant",
			query:       T(tuple),
			permissions: []string{"portal.pc_abc.create_portal_session"},
			wantValid:   true,
		},
		{
			name:        "wildcard tuple grant matches the literal wildcard query",
			query:       T(Tuple{ResourceType: Portal, ResourceID: "*", Action: CreatePortalSession}),
			permissions: []string{"portal.*.create_portal_session"},
			wantValid:   true,
		},
		{
			name:        "wildcard tuple grant does not expand to a concrete portal",
			query:       T(tuple),
			permissions: []string{"portal.*.create_portal_session"},
			wantValid:   false,
		},
		{
			name:        "management grant does not satisfy session minting",
			query:       T(tuple),
			permissions: []string{"portal.pc_abc.read_portal"},
			wantValid:   false,
		},
	}

	rbac := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := rbac.EvaluatePermissions(tt.query, tt.permissions)
			require.NoError(t, err)
			require.Equal(t, tt.wantValid, result.Valid, result.Message)
		})
	}
}

// TestRBAC_PortalUrnPermissions guarantees the canonical URN form scopes portal
// permissions to a single portal, to every portal in the workspace, and never
// across workspaces.
func TestRBAC_PortalUrnPermissions(t *testing.T) {
	t.Parallel()

	query := U(urn.New().Workspace("ws_1").Portal("pc_abc"), permissions.CreatePortalSession{})
	require.Equal(t, "unkey:v1:ws_1:portals/pc_abc#create_portal_session", query.Value)

	tests := []struct {
		name        string
		permissions []string
		wantValid   bool
	}{
		{
			name:        "exact portal grant",
			permissions: []string{"unkey:v1:ws_1:portals/pc_abc#create_portal_session"},
			wantValid:   true,
		},
		{
			name:        "workspace wide portal grant",
			permissions: []string{"unkey:v1:ws_1:portals/*#create_portal_session"},
			wantValid:   true,
		},
		{
			name:        "admin grant",
			permissions: []string{"unkey:v1:ws_1:**#*"},
			wantValid:   true,
		},
		{
			name:        "other portal grant",
			permissions: []string{"unkey:v1:ws_1:portals/pc_xyz#create_portal_session"},
			wantValid:   false,
		},
		{
			name:        "other workspace grant",
			permissions: []string{"unkey:v1:ws_2:portals/pc_abc#create_portal_session"},
			wantValid:   false,
		},
		{
			name:        "management grant does not satisfy session minting",
			permissions: []string{"unkey:v1:ws_1:portals/*#read_portal"},
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

// TestRBAC_PortalUrnSubtreeGrantCoversSessions guarantees a portal subtree grant
// reaches session subresources, which have no typed action of their own.
func TestRBAC_PortalUrnSubtreeGrantCoversSessions(t *testing.T) {
	t.Parallel()

	required := UnkeyPermission{
		Resource: urn.New().Workspace("ws_1").Portal("pc_abc").Session("ps_1"),
		Action:   ActionType(permissions.CreatePortalSession{}.String()),
	}
	granted := UnkeyPermission{
		Resource: urn.V1{WorkspaceID: "ws_1", Resource: "portals/**"},
		Action:   CreatePortalSession,
	}

	require.True(t, permissionCovers(required, granted))
}

// TestRBAC_PortalActionWildcardRejected guarantees a portal-scoped action
// wildcard cannot be granted; only the global "**" resource takes "*".
func TestRBAC_PortalActionWildcardRejected(t *testing.T) {
	t.Parallel()

	_, err := parseUrnPermission("unkey:v1:ws_1:portals/*#*")
	require.ErrorIs(t, err, errInvalidURNPermission)
}
