package openapi_test

import (
	_ "embed"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/unkeyed/unkey/svc/api/internal/oauth"
)

//go:embed overlays/mcp-oauth.yaml
var mcpOAuthOverlayYAML []byte

type mcpOAuthOverlay struct {
	Actions []struct {
		Target string    `yaml:"target"`
		Update yaml.Node `yaml:"update"`
	} `yaml:"actions"`
}

type oauthRequirement struct {
	OAuth2 []string `yaml:"oauth2"`
}

func loadMCPOAuthOverlay(t *testing.T) map[string]yaml.Node {
	t.Helper()
	var overlay mcpOAuthOverlay
	require.NoError(t, yaml.Unmarshal(mcpOAuthOverlayYAML, &overlay))
	updates := make(map[string]yaml.Node, len(overlay.Actions))
	for _, action := range overlay.Actions {
		require.NotContains(t, updates, action.Target, "duplicate overlay target")
		updates[action.Target] = action.Update
	}
	return updates
}

// The MCP gateway reads these URLs from the overlay, not from the server's
// metadata document, so a renamed route would silently break sign-in.
func TestMCPOAuthOverlayMatchesAuthorizationServer(t *testing.T) {
	const issuer = "https://api.unkey.com"
	updates := loadMCPOAuthOverlay(t)

	node, ok := updates["$.components.securitySchemes"]
	require.True(t, ok)
	var schemes struct {
		OAuth2 struct {
			Type  string `yaml:"type"`
			Flows struct {
				AuthorizationCode struct {
					AuthorizationURL string            `yaml:"authorizationUrl"`
					TokenURL         string            `yaml:"tokenUrl"`
					RefreshURL       string            `yaml:"refreshUrl"`
					Scopes           map[string]string `yaml:"scopes"`
				} `yaml:"authorizationCode"`
			} `yaml:"flows"`
		} `yaml:"oauth2"`
	}
	require.NoError(t, node.Decode(&schemes))
	require.Equal(t, "oauth2", schemes.OAuth2.Type)
	flow := schemes.OAuth2.Flows.AuthorizationCode
	require.Equal(t, issuer+oauth.AuthorizePath, flow.AuthorizationURL)
	require.Equal(t, issuer+oauth.TokenPath, flow.TokenURL)
	require.Equal(t, issuer+oauth.TokenPath, flow.RefreshURL)

	scopes := make([]string, 0, len(flow.Scopes))
	for scope := range flow.Scopes {
		scopes = append(scopes, scope)
	}
	slices.Sort(scopes)
	supported := slices.Clone(oauth.ScopesSupported)
	slices.Sort(supported)
	require.Equal(t, supported, scopes)
}

// An update to a requirement object merges oauth2 into it, which OpenAPI reads
// as "root key AND OAuth token". Root-key callers would then fail, so both the
// global and per-operation updates must append a separate list item instead.
func TestMCPOAuthOverlayAddsAlternativeRequirement(t *testing.T) {
	updates := loadMCPOAuthOverlay(t)

	var perOperation []string
	for target := range updates {
		if strings.HasPrefix(target, "$.paths.") {
			perOperation = append(perOperation, target)
		}
	}
	require.Len(t, perOperation, 1)

	for _, target := range []string{"$.security", perOperation[0]} {
		node := updates[target]
		require.True(t, strings.HasSuffix(target, ".security"), "target %q must be the requirement list", target)
		require.Equal(t, yaml.SequenceNode, node.Kind, "update for %q must be a list item, not a merged key", target)

		var requirements []oauthRequirement
		require.NoError(t, node.Decode(&requirements))
		require.Equal(t, []oauthRequirement{{OAuth2: oauth.ScopesSupported}}, requirements, target)
	}
	require.Contains(t, perOperation[0], "security[0].bearer", "only operations that accept a root key gain OAuth")
}
