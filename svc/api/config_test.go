package api

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	sharedconfig "github.com/unkeyed/unkey/pkg/config"
)

func jwtSecretsAuth(secrets []string) AuthConfig {
	return JWTAuthConfig{
		Issuer:  "app.unkey.com",
		Secrets: secrets,
	}
}

func jwtJWKSAuth(jwksURL string) AuthConfig {
	return JWTAuthConfig{
		Issuer:  "https://api.workos.com",
		JWKSURL: jwksURL,
	}
}

// TestConfig_ValidateJWTSecretMinimumLength verifies the API rejects JWT
// secrets shorter than the HS256 entropy requirement (256 bits). Accepting a
// shorter secret would weaken signature security across every token signed or
// verified by this node.
func TestConfig_ValidateJWTSecretMinimumLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		secrets []string
		wantErr bool
	}{
		{
			name:    "empty list is rejected for a jwt entry",
			secrets: nil,
			wantErr: true,
		},
		{
			name:    "32 byte secret meets the minimum",
			secrets: []string{strings.Repeat("a", 32)},
			wantErr: false,
		},
		{
			name:    "31 byte secret is rejected",
			secrets: []string{strings.Repeat("a", 31)},
			wantErr: true,
		},
		{
			name:    "second secret too short still rejected",
			secrets: []string{strings.Repeat("a", 32), strings.Repeat("b", 16)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{Auth: AuthConfigs{jwtSecretsAuth(tt.secrets)}}
			err := cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "secrets")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestConfig_ValidateJWTJWKSURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "empty URL is rejected for a jwt entry",
			url:     "",
			wantErr: true,
		},
		{
			name:    "https URL passes",
			url:     "https://auth.example.com/.well-known/jwks.json",
			wantErr: false,
		},
		{
			name:    "http URL passes for local development",
			url:     "http://auth.local/.well-known/jwks.json",
			wantErr: false,
		},
		{
			name:    "relative URL is rejected",
			url:     "/.well-known/jwks.json",
			wantErr: true,
		},
		{
			name:    "unsupported scheme is rejected",
			url:     "file:///tmp/jwks.json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{Auth: AuthConfigs{jwtJWKSAuth(tt.url)}}
			err := cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "jwks_url")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestConfig_ValidateJWTIssuer(t *testing.T) {
	t.Parallel()

	cfg := &Config{Auth: AuthConfigs{JWTAuthConfig{
		Issuer:  "",
		Secrets: []string{strings.Repeat("a", 32)},
	}}}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "issuer")
}

func TestConfig_ValidateRejectsMixedJWTKeySources(t *testing.T) {
	t.Parallel()

	cfg := &Config{Auth: AuthConfigs{JWTAuthConfig{
		Issuer:  "app.unkey.com",
		Secrets: []string{strings.Repeat("a", 32)},
		JWKSURL: "https://auth.example.com/.well-known/jwks.json",
	}}}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "exactly one of secrets or jwks_url")
}

func TestConfig_ValidateRejectsUnknownJWTFields(t *testing.T) {
	t.Parallel()

	_, err := sharedconfig.LoadBytes[Config]([]byte(`
redis_url = "redis://redis:6379"

[[auth]]
type = "jwt"
issuer = "app.unkey.com"
secrets = ["local-test-secret-with-at-least-32-bytes"]
unknown = "value"

[database]
primary = "unkey:password@tcp(mysql:3306)/unkey"

[control]
url = "http://control:7091"
token = "control-token"
`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown is not valid")
}

func TestConfig_ValidateRejectsDisabledAuthEntry(t *testing.T) {
	t.Parallel()

	disabled := false
	cfg := &Config{
		Auth: AuthConfigs{RootKeyAuthConfig{
			Enabled: &disabled,
		}},
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "enabled=false is invalid")
}

func TestConfig_LoadBytesParsesAuthConfig(t *testing.T) {
	t.Parallel()

	cfg, err := sharedconfig.LoadBytes[Config]([]byte(`
redis_url = "redis://redis:6379"

[[auth]]
type = "jwt"
issuer = "https://api.workos.com"
audience = "api.unkey.com"
jwks_url = "https://auth.example.com/.well-known/jwks.json"

[[auth]]
type = "portal_session"

[[auth]]
type = "root_key"
enabled = true

[database]
primary = "unkey:password@tcp(mysql:3306)/unkey"

[control]
url = "http://control:7091"
token = "control-token"
`))

	require.NoError(t, err)
	require.Len(t, cfg.Auth, 3)
	jwtAuth, ok := cfg.Auth[0].(JWTAuthConfig)
	require.True(t, ok)
	require.Equal(t, "https://api.workos.com", jwtAuth.Issuer)
	require.Equal(t, "api.unkey.com", jwtAuth.Audience)
	require.Equal(t, "https://auth.example.com/.well-known/jwks.json", jwtAuth.JWKSURL)
	_, ok = cfg.Auth[1].(PortalSessionAuthConfig)
	require.True(t, ok)
	rootKeyAuth, ok := cfg.Auth[2].(RootKeyAuthConfig)
	require.True(t, ok)
	require.True(t, rootKeyAuth.enabled())
}
