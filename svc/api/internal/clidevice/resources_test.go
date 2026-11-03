package clidevice

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResourcePolicyAllowsAnyPathOnAnAllowedOrigin(t *testing.T) {
	policy, err := NewResourcePolicy([]string{"https://mcp.unkey.com", "http://localhost:7071/"}, "https://mcp.unkey.com")
	require.NoError(t, err)

	for requested, want := range map[string]string{
		"https://mcp.unkey.com":             "https://mcp.unkey.com",
		"https://mcp.unkey.com/":            "https://mcp.unkey.com",
		"https://MCP.unkey.com/compute":     "https://mcp.unkey.com/compute",
		"https://mcp.unkey.com/mcp/v2/":     "https://mcp.unkey.com/mcp/v2",
		"https://mcp.unkey.com/mcp/compute": "https://mcp.unkey.com/mcp/compute",
		"http://localhost:7071/compute":     "http://localhost:7071/compute",
	} {
		resource, oauthErr := policy.resolve([]string{requested})
		require.Nil(t, oauthErr, requested)
		require.Equal(t, want, resource, requested)
	}

	resource, oauthErr := policy.resolve(nil)
	require.Nil(t, oauthErr)
	require.Equal(t, "https://mcp.unkey.com", resource)
}

func TestResourcePolicyRejectsResourcesItDoesNotServe(t *testing.T) {
	policy, err := NewResourcePolicy([]string{"https://mcp.unkey.com"}, "https://mcp.unkey.com")
	require.NoError(t, err)

	for name, requested := range map[string][]string{
		"other origin":            {"https://mcp.unkey.com.evil.example/mcp"},
		"other scheme":            {"http://mcp.unkey.com"},
		"other port":              {"https://mcp.unkey.com:8443"},
		"relative":                {"/compute"},
		"query":                   {"https://mcp.unkey.com/?toolset=v2"},
		"fragment":                {"https://mcp.unkey.com/#v2"},
		"userinfo":                {"https://user@mcp.unkey.com"},
		"longer than an audience": {"https://mcp.unkey.com/" + strings.Repeat("a", 256)},
		"two resources":           {"https://mcp.unkey.com", "https://mcp.unkey.com/compute"},
	} {
		_, oauthErr := policy.resolve(requested)
		require.NotNil(t, oauthErr, name)
		require.Equal(t, OAuthInvalidTarget, oauthErr.Code, name)
	}
}

func TestNewResourcePolicyRejectsUnusableConfig(t *testing.T) {
	for name, tc := range map[string]struct {
		origins  []string
		fallback string
	}{
		"no origins":              {origins: nil, fallback: "https://mcp.unkey.com"},
		"origin with a path":      {origins: []string{"https://mcp.unkey.com/mcp"}, fallback: "https://mcp.unkey.com/mcp"},
		"origin without a scheme": {origins: []string{"mcp.unkey.com"}, fallback: "https://mcp.unkey.com"},
		"default off origin":      {origins: []string{"https://mcp.unkey.com"}, fallback: "https://api.unkey.com"},
		"missing default":         {origins: []string{"https://mcp.unkey.com"}, fallback: ""},
	} {
		_, err := NewResourcePolicy(tc.origins, tc.fallback)
		require.Error(t, err, name)
	}
}
