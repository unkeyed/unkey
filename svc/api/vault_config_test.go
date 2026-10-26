package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
)

func TestVaultTokenSourceConfig(t *testing.T) {
	for _, tc := range []struct {
		name, vault string
		valid       bool
	}{
		{"disabled", "", true},
		{"static", "[vault]\nurl = 'http://vault'\ntoken = 'test'", true},
		{"file", "[vault]\nurl = 'http://vault'\ntoken_file = '/run/token'", true},
		{"both", "[vault]\nurl = 'http://vault'\ntoken = 'test'\ntoken_file = '/run/token'", false},
		{"neither", "[vault]\nurl = 'http://vault'", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := config.LoadBytes[Config]([]byte(`
redis_url = "redis://redis:6379"
[[auth]]
type = "root_key"
[database]
primary = "unkey:password@tcp(mysql:3306)/unkey"
[control]
url = "http://control:7091"
token = "control-token"
[restate]
url = "https://restate.example.com"
api_key = "restate-test-key"
` + tc.vault))
			if !tc.valid {
				require.ErrorContains(t, err, "exactly one of vault.token or vault.token_file")
				return
			}
			require.NoError(t, err)
			if tc.name == "file" {
				require.Equal(t, "http://vault", cfg.Vault.URL)
				require.Equal(t, "/run/token", cfg.Vault.TokenFile)
				require.Empty(t, cfg.Vault.Token)
			}
			if tc.name == "static" {
				require.Equal(t, "test", cfg.Vault.Token)
				require.Empty(t, cfg.Vault.TokenFile)
			}
		})
	}
}

func TestVaultTokenSourceValidationForProgrammaticConfig(t *testing.T) {
	cfg := configWithAuth(RootKeyAuthConfig{})
	require.NoError(t, cfg.Validate())
	cfg.Vault.URL = "http://vault"
	require.ErrorContains(t, cfg.Validate(), "exactly one of vault.token or vault.token_file")
	cfg.Vault.TokenFile = "/run/token"
	require.NoError(t, cfg.Validate())
	cfg.Vault.Token = "static-token"
	require.ErrorContains(t, cfg.Validate(), "exactly one of vault.token or vault.token_file")
	cfg.Vault.TokenFile = ""
	require.NoError(t, cfg.Validate())
}
