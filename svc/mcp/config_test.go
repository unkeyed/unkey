package mcp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigFromEnv(t *testing.T) {
	env := func(values map[string]string) func(string) string {
		return func(name string) string { return values[name] }
	}

	cfg, err := ConfigFromEnv(env(map[string]string{
		"PUBLIC_URL": "https://mcp.unkey.com/",
		"API_URL":    "https://api.unkey.com/",
	}))
	require.NoError(t, err)
	require.Equal(t, Config{
		Port:        8080,
		PublicURL:   "https://mcp.unkey.com",
		APIURL:      "https://api.unkey.com",
		OAuthIssuer: "https://api.unkey.com",
	}, cfg)

	cfg, err = ConfigFromEnv(env(map[string]string{
		"PORT":         "7095",
		"PUBLIC_URL":   "http://localhost:7095",
		"API_URL":      "http://api:7070",
		"OAUTH_ISSUER": "http://localhost:7070",
	}))
	require.NoError(t, err)
	require.Equal(t, 7095, cfg.Port)
	require.Equal(t, "http://localhost:7070", cfg.OAuthIssuer)

	_, err = ConfigFromEnv(env(map[string]string{"PUBLIC_URL": "https://mcp.unkey.com"}))
	require.ErrorContains(t, err, "API_URL")

	_, err = ConfigFromEnv(env(map[string]string{"PORT": "http", "PUBLIC_URL": "https://mcp.unkey.com", "API_URL": "https://api.unkey.com"}))
	require.ErrorContains(t, err, "PORT")
}
