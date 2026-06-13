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

// TestConfig_ValidateJWTJWKSURL guarantees JWKS auth accepts absolute HTTP(S)
// URLs and rejects missing, relative, or unsupported key sources.
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

// TestConfig_ValidateJWTIssuer guarantees shared-secret JWT auth cannot start
// without an issuer to bind accepted dashboard tokens.
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

// TestConfig_ValidateJWTJWKSAcceptsAnyIssuer guarantees the jwks_url key
// source is not tied to one identity provider; the provider field, not the
// issuer, selects provider-specific behavior at resolver wiring time.
func TestConfig_ValidateJWTJWKSAcceptsAnyIssuer(t *testing.T) {
	t.Parallel()

	cfg := &Config{Auth: AuthConfigs{JWTAuthConfig{
		Issuer:  "https://auth.acme.com",
		JWKSURL: "https://auth.acme.com/.well-known/jwks.json",
	}}}

	require.NoError(t, cfg.Validate())
}

// TestConfig_ValidateAcceptsWorkOSProvider guarantees a jwt entry can opt into
// WorkOS permission translation with any issuer, so custom auth domains and
// environment-scoped issuers select translation the same way.
func TestConfig_ValidateAcceptsWorkOSProvider(t *testing.T) {
	t.Parallel()

	cfg := &Config{Auth: AuthConfigs{JWTAuthConfig{
		Issuer:   "https://auth.acme.com/user_management/client_123",
		JWKSURL:  "https://auth.acme.com/sso/jwks/client_123",
		Provider: "workos",
	}}}

	require.NoError(t, cfg.Validate())
}

// TestConfig_ValidateRejectsUnknownProvider guarantees a misspelled or
// unsupported provider fails startup instead of silently skipping permission
// translation and 403-ing every request.
func TestConfig_ValidateRejectsUnknownProvider(t *testing.T) {
	t.Parallel()

	cfg := &Config{Auth: AuthConfigs{JWTAuthConfig{
		Issuer:   "https://api.workos.com",
		JWKSURL:  "https://api.workos.com/sso/jwks/test",
		Provider: "workoss",
	}}}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "provider")
}

// TestConfig_ValidateJWTJWKSRequiresIssuer guarantees JWKS auth entries bind
// accepted tokens to an explicit issuer instead of implying one.
func TestConfig_ValidateJWTJWKSRequiresIssuer(t *testing.T) {
	t.Parallel()

	cfg := &Config{Auth: AuthConfigs{JWTAuthConfig{
		JWKSURL: "https://api.workos.com/sso/jwks/test",
	}}}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "issuer")
}

// TestConfig_ValidateRequiresAuthEntries guarantees a config without [[auth]]
// tables fails startup instead of booting an API that rejects every request,
// which is what an un-migrated config (for example one still carrying the
// removed jwt_secrets key) would otherwise silently do.
func TestConfig_ValidateRequiresAuthEntries(t *testing.T) {
	t.Parallel()

	cfg := &Config{}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one [[auth]] entry is required")
}

// TestConfig_ValidateRejectsMixedJWTKeySources guarantees a JWT auth entry
// cannot accidentally enable both shared-secret and JWKS verification.
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

// TestConfig_ValidateRejectsUnknownJWTFields guarantees auth config typos fail
// startup instead of silently disabling or weakening authentication.
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

// TestConfig_ValidateRejectsDisabledAuthEntry guarantees auth entries are
// removed rather than disabled in-place, keeping startup behavior explicit.
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

// TestConfig_LoadBytesParsesAuthConfig guarantees typed auth entries round-trip
// from TOML into the resolver configuration consumed at API startup.
func TestConfig_LoadBytesParsesAuthConfig(t *testing.T) {
	t.Parallel()

	cfg, err := sharedconfig.LoadBytes[Config]([]byte(`
redis_url = "redis://redis:6379"

[[auth]]
type = "jwt"
issuer = "https://api.workos.com"
audience = "api.unkey.com"
jwks_url = "https://auth.example.com/.well-known/jwks.json"
provider = "workos"

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
	require.Equal(t, "workos", jwtAuth.Provider)
	_, ok = cfg.Auth[1].(PortalSessionAuthConfig)
	require.True(t, ok)
	rootKeyAuth, ok := cfg.Auth[2].(RootKeyAuthConfig)
	require.True(t, ok)
	require.True(t, rootKeyAuth.enabled())
}
