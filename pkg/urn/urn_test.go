package urn

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// TestParseV1 guarantees concrete v1 resource names round-trip
// through parsing.
func TestParseV1(t *testing.T) {
	t.Parallel()

	resource, err := ParseV1("unkey:v1:ws_123:projects/proj_123/apps/app_456")
	require.NoError(t, err)
	require.Equal(t, V1{
		WorkspaceID: "ws_123",
		Resource:    "projects/proj_123/apps/app_456",
	}, resource)
	require.Equal(t, "unkey:v1:ws_123:projects/proj_123/apps/app_456", resource.String())
}

// TestParseV1_AllowsResourcePatterns guarantees RBAC grants can use resource
// wildcards in the shared resource-name grammar.
func TestParseV1_AllowsResourcePatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  V1
	}{
		{
			name:  "segment wildcard",
			value: "unkey:v1:ws_123:ratelimits/namespaces/*/overrides/*",
			want: V1{
				WorkspaceID: "ws_123",
				Resource:    "ratelimits/namespaces/*/overrides/*",
			},
		},
		{
			name:  "nested segment wildcards",
			value: "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*",
			want: V1{
				WorkspaceID: "ws_123",
				Resource:    "projects/*/apps/*/environments/*/deployments/*",
			},
		},
		{
			name:  "concrete app below wildcard project",
			value: "unkey:v1:ws_123:projects/*/apps/app_123",
			want: V1{
				WorkspaceID: "ws_123",
				Resource:    "projects/*/apps/app_123",
			},
		},
		{
			name:  "concrete environment below wildcard app",
			value: "unkey:v1:ws_123:projects/proj_123/apps/*/environments/env_123",
			want: V1{
				WorkspaceID: "ws_123",
				Resource:    "projects/proj_123/apps/*/environments/env_123",
			},
		},
		{
			name:  "concrete deployment below nested wildcard hierarchy",
			value: "unkey:v1:ws_123:projects/proj_123/apps/*/environments/*/deployments/dep_123",
			want: V1{
				WorkspaceID: "ws_123",
				Resource:    "projects/proj_123/apps/*/environments/*/deployments/dep_123",
			},
		},
		{
			name:  "concrete override below wildcard namespace",
			value: "unkey:v1:ws_123:ratelimits/namespaces/*/overrides/override_123",
			want: V1{
				WorkspaceID: "ws_123",
				Resource:    "ratelimits/namespaces/*/overrides/override_123",
			},
		},
		{
			name:  "project ratelimit namespaces",
			value: "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*",
			want: V1{
				WorkspaceID: "ws_123",
				Resource:    "projects/*/ratelimits/namespaces/*",
			},
		},
		{
			name:  "project RBAC roles",
			value: "unkey:v1:ws_123:projects/*/rbac/roles/*",
			want: V1{
				WorkspaceID: "ws_123",
				Resource:    "projects/*/rbac/roles/*",
			},
		},
		{
			name:  "project app environment gateway policies",
			value: "unkey:v1:ws_123:projects/*/apps/*/environments/*/gateway/policies/*",
			want: V1{
				WorkspaceID: "ws_123",
				Resource:    "projects/*/apps/*/environments/*/gateway/policies/*",
			},
		},
		{
			name:  "wildcard project apps descendant scope",
			value: "unkey:v1:ws_123:projects/*/apps/**",
			want: V1{
				WorkspaceID: "ws_123",
				Resource:    "projects/*/apps/**",
			},
		},
		{
			name:  "wildcard keyspace keys descendant scope",
			value: "unkey:v1:ws_123:keyspaces/*/keys/**",
			want: V1{
				WorkspaceID: "ws_123",
				Resource:    "keyspaces/*/keys/**",
			},
		},
		{
			name:  "wildcard ratelimit overrides descendant scope",
			value: "unkey:v1:ws_123:ratelimits/namespaces/*/overrides/**",
			want: V1{
				WorkspaceID: "ws_123",
				Resource:    "ratelimits/namespaces/*/overrides/**",
			},
		},
		{
			name:  "descendant wildcard",
			value: "unkey:v1:ws_123:ratelimits/**",
			want: V1{
				WorkspaceID: "ws_123",
				Resource:    "ratelimits/**",
			},
		},
		{
			name:  "single segment wildcard",
			value: "unkey:v1:ws_123:*",
			want: V1{
				WorkspaceID: "ws_123",
				Resource:    "*",
			},
		},
		{
			name:  "global wildcard",
			value: "unkey:v1:ws_123:**",
			want: V1{
				WorkspaceID: "ws_123",
				Resource:    "**",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseV1(tt.value)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestParseV1RejectsInvalidValues guarantees malformed values cannot be
// treated as resource names.
func TestParseV1RejectsInvalidValues(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"",
		"urn:unkey:v1:ws_123:keyspaces/ks_123",
		"unkey:v1:ws_123",
		"unkey:v2:ws_123:keyspaces/ks_123",
		"unkey:v1::keyspaces/ks_123",
		"unkey:v1:ws_123:keyspaces/ks_123#read_keyspace",
		"unkey:v1:ws_123:/keyspaces/ks_123",
		"unkey:v1:ws_123:keyspaces/ks_123/",
		"unkey:v1:ws_123:keyspaces//ks_123",
	} {
		_, err := ParseV1(value)
		require.ErrorIs(t, err, ErrInvalidResourceName, value)
	}
}

// TestParseV1_RejectsAmbiguousPatterns guarantees "*" only expands as
// a full path segment and "**" only appears at the end of a resource path.
func TestParseV1_RejectsAmbiguousPatterns(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"unkey:v1:ws_123:ratelimits/**/overrides/*",
		"unkey:v1:ws_123:ratelimits/namespaces/ns_*",
		"unkey:v1:ws_123:/ratelimits/*",
		"unkey:v1:ws_123:ratelimits//namespaces/*",
		"unkey:v1:ws_123:ratelimits/*/",
	} {
		_, err := ParseV1(value)
		require.ErrorIs(t, err, ErrInvalidResourceName, value)
	}
}

// TestResourceCatalogHelpers guarantees the typed builders produce the resource
// paths documented for API handlers and permission grants.
func TestResourceCatalogHelpers(t *testing.T) {
	t.Parallel()

	workspace := New().Workspace("ws_123")
	require.Equal(t, "unkey:v1:ws_123:team/memberships/mbr_123", workspace.Team.Membership("mbr_123").String())
	require.Equal(t, "unkey:v1:ws_123:team/invitations/inv_123", workspace.Team.Invitation("inv_123").String())
	require.Equal(t, "unkey:v1:ws_123:billing/invoices/inv_123", workspace.Billing().Invoice("inv_123").String())
	require.Equal(t, "unkey:v1:ws_123:billing/quotas", workspace.Billing().Quotas().String())
	require.Equal(t, "unkey:v1:ws_123:keyspaces/ks_123", workspace.Keyspace("ks_123").String())
	require.Equal(t, "unkey:v1:ws_123:keyspaces/ks_123/keys/key_123", workspace.Keyspace("ks_123").Key("key_123").String())
	require.Equal(t, "unkey:v1:ws_123:keyspaces/ks_123/**", workspace.Keyspace("ks_123").Any().String())
	require.Equal(t, "unkey:v1:ws_123:ratelimits/namespaces/ns_123/overrides/ov_123", workspace.RatelimitNamespace("ns_123").Override("ov_123").String())
	require.Equal(t, "unkey:v1:ws_123:ratelimits/namespaces/ns_123", workspace.RatelimitNamespace("ns_123").String())
	require.Equal(t, "unkey:v1:ws_123:ratelimits/namespaces/ns_123/**", workspace.RatelimitNamespace("ns_123").Any().String())
	require.Equal(t, "unkey:v1:ws_123:rbac/roles/role_123", workspace.RBAC.Role("role_123").String())
	require.Equal(t, "unkey:v1:ws_123:rbac/permissions/perm_123", workspace.RBAC.Permission("perm_123").String())
	require.Equal(t, "unkey:v1:ws_123:projects/proj_123", workspace.Project("proj_123").String())
	require.Equal(t, "unkey:v1:ws_123:projects/proj_123/**", workspace.Project("proj_123").Any().String())
	require.Equal(t, "unkey:v1:ws_123:projects/proj_123/identities/id_123", workspace.Project("proj_123").Identity("id_123").String())
	require.Equal(t, "unkey:v1:ws_123:projects/proj_123/apps/app_123", workspace.Project("proj_123").App("app_123").String())
	require.Equal(t, "unkey:v1:ws_123:projects/proj_123/apps/app_123/**", workspace.Project("proj_123").App("app_123").Any().String())
	require.Equal(t, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123", workspace.Project("proj_123").App("app_123").Environment("env_123").String())
	require.Equal(t, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/**", workspace.Project("proj_123").App("app_123").Environment("env_123").Any().String())
	require.Equal(t, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123/instances/inst_123", workspace.Project("proj_123").App("app_123").Environment("env_123").Deployment("dep_123").Instance("inst_123").String())
	require.Equal(t, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123", workspace.Project("proj_123").App("app_123").Environment("env_123").Deployment("dep_123").String())
	require.Equal(t, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123/**", workspace.Project("proj_123").App("app_123").Environment("env_123").Deployment("dep_123").Any().String())
	require.Equal(t, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/domains/dom_123", workspace.Project("proj_123").App("app_123").Environment("env_123").Domain("dom_123").String())
	require.Equal(t, "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/variables/var_123", workspace.Project("proj_123").App("app_123").Environment("env_123").Variable("var_123").String())
	require.Equal(t, "unkey:v1:ws_123:portals/portal_123/session_tokens/token_123", workspace.Portal("portal_123").SessionToken("token_123").String())
	require.Equal(t, "unkey:v1:ws_123:portals/portal_123/sessions/session_123", workspace.Portal("portal_123").Session("session_123").String())
	require.Equal(t, "unkey:v1:ws_123:portals/portal_123/branding", workspace.Portal("portal_123").Branding().String())
	require.Equal(t, "unkey:v1:ws_123:portals/portal_123", workspace.Portal("portal_123").String())
	require.Equal(t, "unkey:v1:ws_123:portals/portal_123/**", workspace.Portal("portal_123").Any().String())
}

// TestV1Covers_OnlySupportedWildcardsExpandScope guarantees Covers performs
// directional scope containment for concrete, collection, and descendant
// resource scopes.
func TestV1Covers_OnlySupportedWildcardsExpandScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		target  string
		want    bool
	}{
		{name: "global wildcard", pattern: "**", target: "ratelimits/namespaces/ns_123", want: true},
		{name: "global wildcard covers single segment", pattern: "**", target: "settings", want: true},
		{name: "wildcard descendant alias covers global wildcard", pattern: "*/**", target: "**", want: true},
		{name: "global wildcard covers wildcard descendant alias", pattern: "**", target: "*/**", want: true},
		{name: "two segment descendant scope does not cover global wildcard", pattern: "*/*/**", target: "**", want: false},
		{name: "single segment wildcard matches one segment", pattern: "*", target: "settings", want: true},
		{name: "single segment wildcard does not match nested paths", pattern: "*", target: "ratelimits/namespaces/ns_123", want: false},
		{name: "exact", pattern: "ratelimits/namespaces/ns_123", target: "ratelimits/namespaces/ns_123", want: true},
		{name: "segment wildcard", pattern: "ratelimits/namespaces/*", target: "ratelimits/namespaces/ns_123", want: true},
		{name: "equal segment wildcard scopes", pattern: "ratelimits/namespaces/*", target: "ratelimits/namespaces/*", want: true},
		{name: "concrete does not cover segment wildcard scope", pattern: "ratelimits/namespaces/ns_123", target: "ratelimits/namespaces/*", want: false},
		{name: "descendant wildcard", pattern: "ratelimits/**", target: "ratelimits/namespaces/ns_123", want: true},
		{name: "descendant wildcard covers narrower wildcard scope", pattern: "ratelimits/**", target: "ratelimits/namespaces/*", want: true},
		{name: "descendant wildcard covers narrower descendant scope", pattern: "ratelimits/**", target: "ratelimits/namespaces/**", want: true},
		{name: "segment wildcard does not cover descendant scope", pattern: "ratelimits/*", target: "ratelimits/**", want: false},
		{name: "narrow descendant scope does not cover broader descendant scope", pattern: "ratelimits/namespaces/**", target: "ratelimits/**", want: false},
		{name: "descendant wildcard covers base", pattern: "ratelimits/**", target: "ratelimits", want: true},
		{name: "descendant wildcard target shorter than base", pattern: "ratelimits/namespaces/**", target: "ratelimits", want: false},
		{name: "descendant wildcard wrong prefix", pattern: "identities/**", target: "ratelimits/namespaces/ns_123", want: false},
		{name: "segment wildcard does not cross segments", pattern: "ratelimits/*", target: "ratelimits/namespaces/ns_123", want: false},
		{name: "exact shorter", pattern: "ratelimits", target: "ratelimits/namespaces/ns_123", want: false},
		{name: "exact longer", pattern: "ratelimits/namespaces/ns_123", target: "ratelimits", want: false},
		{name: "descendant covers sibling collection", pattern: "a/b/**", target: "a/b/*", want: true},
		{name: "sibling collection does not cover descendant", pattern: "a/b/*", target: "a/b/**", want: false},
		{name: "wildcard parent descendant covers concrete parent descendant", pattern: "a/*/**", target: "a/b/**", want: true},
		{name: "concrete parent descendant does not cover wildcard parent descendant", pattern: "a/b/**", target: "a/*/**", want: false},
		{name: "wildcard parent descendant covers parent collection", pattern: "a/*/**", target: "a/*", want: true},
		{name: "concrete parent descendant does not cover wildcard collection", pattern: "a/b/**", target: "a/*", want: false},
		{name: "nested wildcard descendant covers concrete nested collection", pattern: "a/*/c/**", target: "a/b/c/*", want: true},
		{name: "concrete nested descendant does not cover wildcard nested collection", pattern: "a/b/c/**", target: "a/*/c/*", want: false},
		{name: "equal nested wildcard collections", pattern: "a/*/c/*", target: "a/*/c/*", want: true},
		{name: "nested collection does not cover nested descendant", pattern: "a/*/c/*", target: "a/*/c/**", want: false},
		{name: "wrong prefix descendant scopes", pattern: "a/**", target: "b/**", want: false},
		{name: "reverse wrong prefix descendant scopes", pattern: "b/**", target: "a/**", want: false},
		{name: "single segment wildcard does not cover global descendant", pattern: "*", target: "**", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pattern := V1{WorkspaceID: "ws_123", Resource: tt.pattern}
			target := V1{WorkspaceID: "ws_123", Resource: tt.target}
			require.Equal(t, tt.want, pattern.Covers(target))
		})
	}
}

// TestV1Covers_RequiresMatchingWorkspace guarantees no resource pattern, even
// the global wildcard, crosses workspace boundaries.
func TestV1Covers_RequiresMatchingWorkspace(t *testing.T) {
	t.Parallel()

	pattern := V1{WorkspaceID: "ws_123", Resource: "**"}
	target := V1{WorkspaceID: "ws_456", Resource: "ratelimits/namespaces/ns_123"}
	require.False(t, pattern.Covers(target))
}

// FuzzV1Covers_ExactAndGlobalWildcardInvariants guarantees fuzzed matcher
// inputs preserve the two smallest invariants every caller relies on.
func FuzzV1Covers_ExactAndGlobalWildcardInvariants(f *testing.F) {
	fuzz.Seed(f)
	for _, seed := range []struct {
		pattern string
		target  string
	}{
		{pattern: "**", target: "ratelimits/namespaces/ns_123"},
		{pattern: "ratelimits/**", target: "ratelimits/namespaces/ns_123/overrides/ov_123"},
		{pattern: "ratelimits/namespaces/*", target: "ratelimits/namespaces/ns_123"},
		{pattern: "ratelimits/namespaces/ns_123", target: "ratelimits/namespaces/ns_123"},
	} {
		f.Add(fuzzStringSeed(seed.pattern, seed.target))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		pattern := V1{WorkspaceID: "ws_123", Resource: c.String()}
		target := V1{WorkspaceID: "ws_123", Resource: c.String()}

		got := pattern.Covers(target)
		if pattern.Resource == target.Resource {
			require.True(t, got)
		}
		if pattern.Resource == "**" {
			require.True(t, got)
		}
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
