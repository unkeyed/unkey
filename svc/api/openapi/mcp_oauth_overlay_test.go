package openapi_test

import (
	_ "embed"
	"slices"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/unkeyed/unkey/svc/api/openapi"
	"github.com/unkeyed/unkey/svc/api/routes/oauth"
)

//go:embed overlays/mcp-oauth.yaml
var mcpOAuthOverlay []byte

type securityRequirement map[string][]string

type mcpOAuthSpec struct {
	Components struct {
		SecuritySchemes struct {
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
		} `yaml:"securitySchemes"`
	} `yaml:"components"`
	Servers []struct {
		URL string `yaml:"url"`
	} `yaml:"servers"`
	Security []securityRequirement           `yaml:"security"`
	Paths    map[string]map[string]yaml.Node `yaml:"paths"`
}

// operationSecurity maps "METHOD path" to the operation's own security list.
// A nil entry means the operation has no security key and inherits the global
// list, which differs from an explicit empty list that disables authentication.
func (s mcpOAuthSpec) operationSecurity(t *testing.T) map[string]*[]securityRequirement {
	t.Helper()
	security := map[string]*[]securityRequirement{}
	for path, item := range s.Paths {
		for method, node := range item {
			if node.Kind != yaml.MappingNode {
				continue
			}
			var operation struct {
				Security *[]securityRequirement `yaml:"security"`
			}
			require.NoError(t, node.Decode(&operation))
			security[method+" "+path] = operation.Security
		}
	}
	return security
}

func decodeMCPOAuthSpec(t *testing.T, spec []byte) mcpOAuthSpec {
	t.Helper()
	var doc mcpOAuthSpec
	require.NoError(t, yaml.Unmarshal(spec, &doc))
	return doc
}

func applyMCPOAuthOverlay(t *testing.T) mcpOAuthSpec {
	t.Helper()
	result, err := libopenapi.ApplyOverlayFromBytesToSpecBytes(openapi.Spec, mcpOAuthOverlay)
	require.NoError(t, err)
	require.Empty(t, result.Warnings, "every overlay target must match the spec")
	return decodeMCPOAuthSpec(t, result.Bytes)
}

// The MCP build reads the OAuth endpoints from the overlay, not from the
// metadata document, so a renamed route would silently break sign-in.
// Permissions come from the consent page, so the flow requests no scopes.
func TestMCPOAuthOverlayPointsAtAuthorizationServer(t *testing.T) {
	t.Parallel()
	doc := applyMCPOAuthOverlay(t)

	require.NotEmpty(t, doc.Servers)
	issuer := doc.Servers[0].URL
	scheme := doc.Components.SecuritySchemes.OAuth2
	require.Equal(t, "oauth2", scheme.Type)
	flow := scheme.Flows.AuthorizationCode
	require.Equal(t, issuer+oauth.PathAuthorize, flow.AuthorizationURL)
	require.Equal(t, issuer+oauth.PathToken, flow.TokenURL)
	require.Equal(t, issuer+oauth.PathToken, flow.RefreshURL)
	require.NotNil(t, flow.Scopes, "OpenAPI requires the scopes map even when it is empty")
	require.Empty(t, flow.Scopes)
}

// OAuth must be a separate requirement. Merged into the root key requirement,
// OpenAPI reads it as "root key AND OAuth token", which locks out both root key
// and OAuth callers. Operations that do not take a root key keep their security.
func TestMCPOAuthOverlayAddsOAuthAsRootKeyAlternative(t *testing.T) {
	t.Parallel()
	original := decodeMCPOAuthSpec(t, openapi.Spec)
	patched := applyMCPOAuthOverlay(t)
	oauthRequirement := securityRequirement{"oauth2": {}}

	require.Equal(t, append(slices.Clone(original.Security), oauthRequirement), patched.Security)

	before := original.operationSecurity(t)
	after := patched.operationSecurity(t)
	require.Len(t, after, len(before))
	var rootKeyOperations int
	for operation, security := range before {
		takesRootKey := security != nil && slices.ContainsFunc(*security, func(r securityRequirement) bool {
			_, ok := r["bearer"]
			return ok
		})
		if !takesRootKey {
			require.Equal(t, security, after[operation], operation)
			continue
		}
		rootKeyOperations++
		require.NotNil(t, after[operation], operation)
		require.Equal(t, append(slices.Clone(*security), oauthRequirement), *after[operation], operation)
	}
	require.NotZero(t, rootKeyOperations)
}
