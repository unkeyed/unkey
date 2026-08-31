package rbac

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
)

// TestUnkeyPermissionQuery_BuildsCanonicalPermission guarantees the typed RBAC
// helper emits the full permission string principals store.
func TestUnkeyPermissionQuery_BuildsCanonicalPermission(t *testing.T) {
	t.Parallel()

	resource := urn.New().Workspace("ws_123").Project("proj_123").RatelimitNamespace("ns_123").Override("ov_123")
	query := U(resource, permissions.Read)
	stringQuery := S(UnkeyPermission{
		Resource: urn.V1{
			WorkspaceID: "ws_123",
			Resource:    "projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123",
		},
		Action: ActionType(permissions.Read.String()),
	}.String())
	createQuery := U(
		urn.New().Workspace("ws_123").Project("proj_123").Keyspace("ks_123").Key("*"),
		permissions.Write,
	)

	require.Equal(t, "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123#read", query.Value)
	require.Equal(t, query.Value, stringQuery.Value)
	require.Equal(t, "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/keys/*#write", createQuery.Value)

	result, err := New().EvaluatePermissions(query, []string{
		"unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123#read",
	})
	require.NoError(t, err)
	require.True(t, result.Valid)
}

// TestStringQuery_DoesNotOptIntoUnkeyWildcardMatching guarantees callers must
// choose U() before canonical Unkey permission grants can expand wildcards.
func TestStringQuery_DoesNotOptIntoUnkeyWildcardMatching(t *testing.T) {
	t.Parallel()

	required := "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123#read"
	grants := []string{
		"unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/*#read",
	}

	stringResult, err := New().EvaluatePermissions(S(required), grants)
	require.NoError(t, err)
	require.False(t, stringResult.Valid)

	typedResult, err := New().EvaluatePermissions(
		U(
			urn.New().
				Workspace("ws_123").
				Project("proj_123").
				RatelimitNamespace("ns_123").
				Override("ov_123"),
			permissions.Read,
		),
		grants,
	)
	require.NoError(t, err)
	require.True(t, typedResult.Valid)
}

// TestParseUrnPermission_AcceptsOnlySupportedGrammar guarantees malformed
// permission strings cannot accidentally participate in wildcard matching.
func TestParseUrnPermission_AcceptsOnlySupportedGrammar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    UnkeyPermission
		wantErr bool
	}{
		{
			name:  "exact resource",
			value: "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123#read",
			want: UnkeyPermission{
				Resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123"},
				Action:   ActionType(permissions.Read.String()),
			},
		},
		{
			name:  "segment wildcard resource",
			value: "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/overrides/*#read",
			want: UnkeyPermission{
				Resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/*/ratelimits/namespaces/*/overrides/*"},
				Action:   ActionType(permissions.Read.String()),
			},
		},
		{
			name:  "descendant wildcard resource",
			value: "unkey:v1:ws_123:projects/proj_123/**#read",
			want: UnkeyPermission{
				Resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/proj_123/**"},
				Action:   ActionType(permissions.Read.String()),
			},
		},
		{
			name:  "admin permission",
			value: "unkey:v1:ws_123:**#*",
			want: UnkeyPermission{
				Resource: urn.V1{WorkspaceID: "ws_123", Resource: "**"},
				Action:   "*",
			},
		},
		{name: "write action", value: "unkey:v1:ws_123:projects/proj_123#write", want: UnkeyPermission{Resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/proj_123"}, Action: "write"}},
		{name: "delete action", value: "unkey:v1:ws_123:projects/proj_123#delete", want: UnkeyPermission{Resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/proj_123"}, Action: "delete"}},
		{name: "decrypt action", value: "unkey:v1:ws_123:projects/proj_123#decrypt", want: UnkeyPermission{Resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/proj_123"}, Action: "decrypt"}},
		{name: "verify action", value: "unkey:v1:ws_123:projects/proj_123#verify", want: UnkeyPermission{Resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/proj_123"}, Action: "verify"}},
		{name: "limit action", value: "unkey:v1:ws_123:projects/proj_123#limit", want: UnkeyPermission{Resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/proj_123"}, Action: "limit"}},
		{name: "old resource-qualified action", value: "unkey:v1:ws_123:projects/proj_123#write_project", wantErr: true},
		{name: "missing action separator", value: "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123", wantErr: true},
		{name: "empty resource", value: "unkey:v1:ws_123:#read", wantErr: true},
		{name: "empty action", value: "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#", wantErr: true},
		{name: "extra action separator", value: "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#read#override", wantErr: true},
		{name: "wrong prefix", value: "urn:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#read", wantErr: true},
		{name: "wrong version", value: "unkey:v2:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#read", wantErr: true},
		{name: "missing workspace resource separator", value: "unkey:v1:ws_123#read", wantErr: true},
		{name: "empty workspace", value: "unkey:v1::projects/proj_123/ratelimits/namespaces/ns_123#read", wantErr: true},
		{name: "workspace with slash", value: "unkey:v1:ws/123:projects/proj_123/ratelimits/namespaces/ns_123#read", wantErr: true},
		{name: "resource with colon", value: "unkey:v1:ws_123:projects/proj_123/ratelimits:namespaces/ns_123#read", wantErr: true},
		{name: "leading slash", value: "unkey:v1:ws_123:/projects/proj_123/ratelimits/namespaces/ns_123#read", wantErr: true},
		{name: "trailing slash", value: "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/#read", wantErr: true},
		{name: "empty segment", value: "unkey:v1:ws_123:projects/proj_123/ratelimits//namespaces/ns_123#read", wantErr: true},
		{name: "partial wildcard", value: "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_*#read", wantErr: true},
		{name: "descendant wildcard in middle", value: "unkey:v1:ws_123:projects/proj_123/ratelimits/**/overrides/*#read", wantErr: true},
		{name: "action with slash", value: "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#read/override", wantErr: true},
		{name: "action with colon", value: "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#read:override", wantErr: true},
		{name: "action wildcard without global resource", value: "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#*", wantErr: true},
		{name: "action wildcard with single segment wildcard resource", value: "unkey:v1:ws_123:*#*", wantErr: true},
		{name: "leading action separator", value: "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#_read", wantErr: true},
		{name: "arbitrary action", value: "unkey:v1:ws_123:projects/proj_123#publish", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseUrnPermission(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestIsUnkeyPermission_OnlyAcceptsCanonicalUnkeyPermissions guarantees the
// evaluator never applies Unkey wildcard semantics to legacy or customer grants.
func TestIsUnkeyPermission_OnlyAcceptsCanonicalUnkeyPermissions(t *testing.T) {
	t.Parallel()

	require.True(t, isUnkeyPermission("unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#read"))
	require.False(t, isUnkeyPermission("api.*.read_key"))
	require.False(t, isUnkeyPermission("ratelimit.ns_123.read_override"))
	require.False(t, isUnkeyPermission("unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123"))
}

// TestPermissionCovers_GrantedWildcardCoversExactRequiredResource guarantees
// handlers can query exact resources while principals hold broader grants.
func TestPermissionCovers_GrantedWildcardCoversExactRequiredResource(t *testing.T) {
	t.Parallel()

	required := UnkeyPermission{
		Resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123"},
		Action:   ActionType(permissions.Read.String()),
	}

	tests := []struct {
		name    string
		granted UnkeyPermission
		want    bool
	}{
		{
			name:    "exact permission",
			granted: required,
			want:    true,
		},
		{
			name: "segment wildcard permission",
			granted: UnkeyPermission{
				Resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/proj_123/ratelimits/namespaces/ns_123/overrides/*"},
				Action:   ActionType(permissions.Read.String()),
			},
			want: true,
		},
		{
			name: "workos wildcard permission",
			granted: UnkeyPermission{
				Resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/*/ratelimits/namespaces/*/overrides/*"},
				Action:   ActionType(permissions.Read.String()),
			},
			want: true,
		},
		{
			name: "descendant wildcard permission",
			granted: UnkeyPermission{
				Resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/proj_123/**"},
				Action:   ActionType(permissions.Read.String()),
			},
			want: true,
		},
		{
			name: "admin permission",
			granted: UnkeyPermission{
				Resource: urn.V1{WorkspaceID: "ws_123", Resource: "**"},
				Action:   "*",
			},
			want: true,
		},
		{
			name: "wrong workspace",
			granted: UnkeyPermission{
				Resource: urn.V1{WorkspaceID: "ws_456", Resource: "projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123"},
				Action:   ActionType(permissions.Read.String()),
			},
			want: false,
		},
		{
			name: "wrong action",
			granted: UnkeyPermission{
				Resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123"},
				Action:   ActionType(permissions.Delete.String()),
			},
			want: false,
		},
		{
			name: "sibling resource",
			granted: UnkeyPermission{
				Resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/proj_123/ratelimits/namespaces/ns_456/overrides/*"},
				Action:   ActionType(permissions.Read.String()),
			},
			want: false,
		},
		{
			name: "shorter exact resource",
			granted: UnkeyPermission{
				Resource: urn.V1{WorkspaceID: "ws_123", Resource: "projects/proj_123/ratelimits/namespaces/ns_123"},
				Action:   ActionType(permissions.Read.String()),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, permissionCovers(required, tt.granted))
		})
	}
}

// TestUrnPermissionEvaluation_MatchesThroughRBACEvaluator guarantees the public
// RBAC evaluator applies URN wildcard semantics, not just private helpers.
func TestUrnPermissionEvaluation_MatchesThroughRBACEvaluator(t *testing.T) {
	t.Parallel()

	query := U(
		urn.New().
			Workspace("ws_123").
			Project("proj_123").
			RatelimitNamespace("ns_123").
			Override("ov_123"),
		permissions.Read,
	)

	tests := []struct {
		name        string
		permissions []string
		wantValid   bool
	}{
		{
			name: "exact permission",
			permissions: []string{
				"unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123#read",
			},
			wantValid: true,
		},
		{
			name: "namespace override wildcard permission",
			permissions: []string{
				"unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/*#read",
			},
			wantValid: true,
		},
		{
			name: "workos wildcard namespace permission",
			permissions: []string{
				"unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/overrides/*#read",
			},
			wantValid: true,
		},
		{
			name: "project descendant permission",
			permissions: []string{
				"unkey:v1:ws_123:projects/proj_123/**#read",
			},
			wantValid: true,
		},
		{
			name: "admin permission",
			permissions: []string{
				"unkey:v1:ws_123:**#*",
			},
			wantValid: true,
		},
		{
			name: "malformed grants are ignored",
			permissions: []string{
				"unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/**/nested#read",
				"unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123#delete",
				"ratelimit.ns_123.read_override",
			},
			wantValid: false,
		},
		{
			name: "wrong workspace",
			permissions: []string{
				"unkey:v1:ws_456:projects/proj_123/ratelimits/namespaces/ns_123/overrides/*#read",
			},
			wantValid: false,
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

// FuzzParseUrnPermission_InvalidInputNeverProducesUnsafePermission guarantees
// arbitrary strings either fail parsing or produce self-covering valid permissions.
func FuzzParseUrnPermission_InvalidInputNeverProducesUnsafePermission(f *testing.F) {
	fuzz.Seed(f)
	for _, seed := range []string{
		"",
		"unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123#read",
		"unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/overrides/*#read",
		"unkey:v1:ws_123:projects/proj_123/**#read",
		"unkey:v1:ws_123:**#*",
		"unkey:v1:ws_123:projects/proj_123/ratelimits/**/overrides/*#read",
		"unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123#read#override",
	} {
		f.Add(fuzzStringSeed(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		value := c.String()

		permission, err := parseUrnPermission(value)
		if err != nil {
			return
		}

		require.NotEmpty(t, permission.Resource.WorkspaceID)
		require.NotEmpty(t, permission.Resource.Resource)
		require.NotEmpty(t, permission.Action)
		_, err = urn.ParseV1(permission.Resource.String())
		require.NoError(t, err)
		require.NoError(t, validatePermissionAction(string(permission.Action)))
		require.True(t, permissionCovers(permission, permission))
	})
}

func fuzzStringSeed(values ...string) []byte {
	out := make([]byte, 0)
	for _, value := range values {
		out = binary.BigEndian.AppendUint16(out, uint16(len(value)))
		out = append(out, value...)
	}
	return out
}
