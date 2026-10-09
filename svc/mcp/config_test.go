package mcp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigFromEnv_BuildsIssuerAndJWKS(t *testing.T) {
	cfg, err := ConfigFromEnv(func(key string) string {
		switch key {
		case "AUTHKIT_DOMAIN":
			return "login.example.test"
		case "PUBLIC_BASE_URL":
			return "https://mcp.example.test"
		case "LISTEN_ADDR":
			return "127.0.0.1:8787"
		case "MCP_SPIKE_LOG_CLAIMS":
			return "true"
		default:
			return ""
		}
	})
	require.NoError(t, err)
	require.Equal(t, "https://login.example.test", cfg.Issuer)
	require.Equal(t, "https://login.example.test/oauth2/jwks", cfg.JWKSURL)
	require.Equal(t, "https://mcp.example.test/api", cfg.ResourceURL(ResourceAPI))
	require.Equal(t, "https://mcp.example.test/compute", cfg.ResourceURL(ResourceCompute))
	require.Equal(t, "https://mcp.example.test/.well-known/oauth-protected-resource/api", cfg.MetadataURL(ResourceAPI))
	require.True(t, cfg.LogClaims)
}

func TestConfigFromEnv_AcceptsHTTPSOriginDomain(t *testing.T) {
	cfg, err := ConfigFromEnv(envWith("https://login.example.test", "https://mcp.example.test", "127.0.0.1:8787", ""))
	require.NoError(t, err)
	require.Equal(t, "https://login.example.test/oauth2/jwks", cfg.JWKSURL)
	require.False(t, cfg.LogClaims)
}

func TestConfigFromEnv_RejectsIncompleteOrPathedURLs(t *testing.T) {
	_, err := ConfigFromEnv(envWith("", "https://mcp.example.test", "127.0.0.1:8787", ""))
	require.ErrorContains(t, err, "AUTHKIT_DOMAIN")

	_, err = ConfigFromEnv(envWith("login.example.test", "https://mcp.example.test/", "127.0.0.1:8787", ""))
	require.ErrorContains(t, err, "PUBLIC_BASE_URL")

	_, err = ConfigFromEnv(envWith("login.example.test", "https://mcp.example.test/spike", "127.0.0.1:8787", ""))
	require.ErrorContains(t, err, "PUBLIC_BASE_URL")

	_, err = ConfigFromEnv(envWith("login.example.test", "https://mcp.example.test", "", ""))
	require.ErrorContains(t, err, "LISTEN_ADDR")
}

func envWith(domain string, public string, listen string, logClaims string) func(string) string {
	return func(key string) string {
		switch key {
		case "AUTHKIT_DOMAIN":
			return domain
		case "PUBLIC_BASE_URL":
			return public
		case "LISTEN_ADDR":
			return listen
		case "MCP_SPIKE_LOG_CLAIMS":
			return logClaims
		default:
			return ""
		}
	}
}
