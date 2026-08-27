package urn

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// TestParseV1 guarantees a concrete v1 resource name round-trips through parsing.
func TestParseV1(t *testing.T) {
	t.Parallel()

	value := "unkey:v1:ws_123:projects/proj_123/apps/app_456"
	resource, err := ParseV1(value)
	require.NoError(t, err)
	require.Equal(t, V1{
		WorkspaceID: "ws_123",
		Resource:    "projects/proj_123/apps/app_456",
	}, resource)
	require.Equal(t, value, resource.String())
}

// TestParseV1AllowsCanonicalPatterns guarantees canonical resource patterns use
// wildcards only in supported positions.
func TestParseV1AllowsCanonicalPatterns(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"unkey:v1:ws_123:github/apps/*",
		"unkey:v1:ws_123:projects/*",
		"unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*/logs",
		"unkey:v1:ws_123:projects/*/apps/*/environments/*/gateway/logs",
		"unkey:v1:ws_123:projects/*/keyspaces/*/logs",
		"unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/logs",
		"unkey:v1:ws_123:projects/*/rbac/permissions/*",
		"unkey:v1:ws_123:projects/proj_123/**",
		"unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/gateway/**",
		"unkey:v1:ws_123:projects/proj_123/rbac/**",
		"unkey:v1:ws_123:**",
	} {
		value := value
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			got, err := ParseV1(value)
			require.NoError(t, err)
			require.Equal(t, value, got.String())
		})
	}
}

// TestParseV1RejectsInvalidValues guarantees malformed and non-canonical values
// cannot be treated as resource names.
func TestParseV1RejectsInvalidValues(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"",
		"urn:unkey:v1:ws_123:projects/proj_123",
		"unkey:v1:ws_123",
		"unkey:v2:ws_123:projects/proj_123",
		"unkey:v1::projects/proj_123",
		"unkey:v1:ws_123:projects/proj_123#read_project",
		"unkey:v1:ws_123:/projects/proj_123",
		"unkey:v1:ws_123:projects/proj_123/",
		"unkey:v1:ws_123:projects//proj_123",
		"unkey:v1:ws_123:keyspaces/ks_123",
		"unkey:v1:ws_123:ratelimits/namespaces/ns_123",
		"unkey:v1:ws_123:rbac/roles/role_123",
		"unkey:v1:ws_123:portals/portal_123",
		"unkey:v1:ws_123:projects/proj_123/unknown/value",
		"unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/gateway",
		"unkey:v1:ws_123:projects/proj_123/rbac",
		"unkey:v1:ws_123:projects/*/apps/app_123",
		"unkey:v1:ws_123:projects/proj_123/apps/*/environments/env_123",
		"unkey:v1:ws_123:projects/proj_123/keyspaces/*/keys/key_123",
		"unkey:v1:ws_123:projects/proj_123/apps/**/environments/*",
		"unkey:v1:ws_123:projects/proj_*/apps/*",
		"unkey:v1:ws_123:projects/*/apps/**",
		"unkey:v1:ws_123:*",
	} {
		value := value
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			_, err := ParseV1(value)
			require.ErrorIs(t, err, ErrInvalidResourceName)
		})
	}
}

// TestResourceCatalogBuilders guarantees every typed builder produces a
// canonical resource path.
func TestResourceCatalogBuilders(t *testing.T) {
	t.Parallel()

	workspace := New().Workspace("ws_123")
	project := workspace.Project("proj_123")
	app := project.App("app_123")
	environment := app.Environment("env_123")
	deployment := environment.Deployment("dep_123")
	keyspace := project.Keyspace("ks_123")
	namespace := project.RatelimitNamespace("ns_123")

	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "GitHub app", got: workspace.GitHubApp("github_123").String(), want: "unkey:v1:ws_123:github/apps/github_123"},
		{name: "project", got: project.String(), want: "unkey:v1:ws_123:projects/proj_123"},
		{name: "app", got: app.String(), want: "unkey:v1:ws_123:projects/proj_123/apps/app_123"},
		{name: "environment", got: environment.String(), want: "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123"},
		{name: "deployment", got: deployment.String(), want: "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123"},
		{name: "deployment logs", got: deployment.Logs().String(), want: "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/deployments/dep_123/logs"},
		{name: "domain", got: environment.Domain("dom_123").String(), want: "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/domains/dom_123"},
		{name: "environment variable", got: environment.Variable("var_123").String(), want: "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/variables/var_123"},
		{name: "gateway logs", got: environment.Gateway().Logs().String(), want: "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/gateway/logs"},
		{name: "gateway policy", got: environment.Gateway().Policy("pol_123").String(), want: "unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/gateway/policies/pol_123"},
		{name: "identity", got: project.Identity("id_123").String(), want: "unkey:v1:ws_123:projects/proj_123/identities/id_123"},
		{name: "keyspace", got: keyspace.String(), want: "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123"},
		{name: "keyspace logs", got: keyspace.Logs().String(), want: "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/logs"},
		{name: "key", got: keyspace.Key("key_123").String(), want: "unkey:v1:ws_123:projects/proj_123/keyspaces/ks_123/keys/key_123"},
		{name: "rate limit namespace", got: namespace.String(), want: "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123"},
		{name: "rate limit logs", got: namespace.Logs().String(), want: "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/logs"},
		{name: "rate limit override", got: namespace.Override("ov_123").String(), want: "unkey:v1:ws_123:projects/proj_123/ratelimits/namespaces/ns_123/overrides/ov_123"},
		{name: "role", got: project.RBAC().Role("role_123").String(), want: "unkey:v1:ws_123:projects/proj_123/rbac/roles/role_123"},
		{name: "permission", got: project.RBAC().Permission("perm_123").String(), want: "unkey:v1:ws_123:projects/proj_123/rbac/permissions/perm_123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.got)
			parsed, err := ParseV1(tt.got)
			require.NoError(t, err)
			require.Equal(t, tt.got, parsed.String())
		})
	}
}

// TestResourceCatalogDescendantBuilders guarantees resource builders produce
// valid descendant patterns.
func TestResourceCatalogDescendantBuilders(t *testing.T) {
	t.Parallel()

	project := New().Workspace("ws_123").Project("proj_123")
	app := project.App("app_123")
	environment := app.Environment("env_123")

	for _, value := range []string{
		project.Any().String(),
		app.Any().String(),
		environment.Any().String(),
		environment.Deployment("dep_123").Any().String(),
		project.Keyspace("ks_123").Any().String(),
		project.RatelimitNamespace("ns_123").Any().String(),
	} {
		_, err := ParseV1(value)
		require.NoError(t, err, value)
	}
}

// TestV1Covers guarantees exact paths and supported wildcards expand scope as
// documented.
func TestV1Covers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		target  string
		want    bool
	}{
		{name: "global wildcard", pattern: "**", target: "projects/proj_123/keyspaces/ks_123", want: true},
		{name: "exact", pattern: "projects/proj_123/keyspaces/ks_123", target: "projects/proj_123/keyspaces/ks_123", want: true},
		{name: "segment wildcard", pattern: "projects/*/keyspaces/*", target: "projects/proj_123/keyspaces/ks_123", want: true},
		{name: "descendant wildcard", pattern: "projects/proj_123/**", target: "projects/proj_123/keyspaces/ks_123/keys/key_123", want: true},
		{name: "descendant wildcard covers base", pattern: "projects/proj_123/**", target: "projects/proj_123", want: true},
		{name: "descendant wildcard target shorter than base", pattern: "projects/proj_123/keyspaces/ks_123/**", target: "projects/proj_123", want: false},
		{name: "descendant wildcard wrong prefix", pattern: "projects/proj_123/keyspaces/**", target: "projects/proj_123/apps/app_123", want: false},
		{name: "segment wildcard does not cross segments", pattern: "projects/*", target: "projects/proj_123/keyspaces/ks_123", want: false},
		{name: "exact shorter", pattern: "projects/proj_123", target: "projects/proj_123/keyspaces/ks_123", want: false},
		{name: "exact longer", pattern: "projects/proj_123/keyspaces/ks_123", target: "projects/proj_123", want: false},
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

// TestV1CoversRequiresMatchingWorkspace guarantees no resource pattern crosses
// a workspace boundary.
func TestV1CoversRequiresMatchingWorkspace(t *testing.T) {
	t.Parallel()

	pattern := V1{WorkspaceID: "ws_123", Resource: "**"}
	target := V1{WorkspaceID: "ws_456", Resource: "projects/proj_123"}
	require.False(t, pattern.Covers(target))
}

// FuzzV1CoversExactAndGlobalWildcardInvariants guarantees fuzzed matcher inputs
// preserve exact-match and global-wildcard behavior.
func FuzzV1CoversExactAndGlobalWildcardInvariants(f *testing.F) {
	fuzz.Seed(f)
	for _, seed := range []struct {
		pattern string
		target  string
	}{
		{pattern: "**", target: "projects/proj_123/ratelimits/namespaces/ns_123"},
		{pattern: "projects/proj_123/**", target: "projects/proj_123/keyspaces/ks_123/keys/key_123"},
		{pattern: "projects/*/keyspaces/*", target: "projects/proj_123/keyspaces/ks_123"},
		{pattern: "projects/proj_123", target: "projects/proj_123"},
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
