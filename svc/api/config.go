package api

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/tls"
)

// ClickHouseConfig configures connections to ClickHouse for analytics storage.
// All fields are optional; when URL is empty, a no-op analytics backend is used.
type ClickHouseConfig struct {
	// URL is the ClickHouse connection string for the shared analytics cluster.
	// When empty, analytics writes are silently discarded.
	// Example: "clickhouse://default:password@clickhouse:9000?secure=false&skip_verify=true"
	URL string `toml:"url"`

	// AnalyticsURL is the base URL for workspace-specific analytics connections.
	// Unlike URL, this endpoint receives per-workspace credentials injected at
	// connection time by the analytics service. Only used when both this field
	// and a [VaultConfig] are configured.
	// Example: "http://clickhouse:8123/default"
	AnalyticsURL string `toml:"analytics_url"`
}

const (
	authTypeJWT           = "jwt"
	authTypePortalSession = "portal_session"
	authTypeRootKey       = "root_key"
)

// AuthConfig is a discriminated union for one authentication resolver.
// Implementations are selected from TOML auth entries by their type field.
type AuthConfig interface {
	authConfig()
	authConfigType() string
}

// AuthConfigs is the ordered authentication resolver chain.
type AuthConfigs []AuthConfig

// JWTAuthConfig configures JWT bearer authentication.
type JWTAuthConfig struct {
	// Issuer is the expected JWT iss claim for this auth entry.
	Issuer string `toml:"issuer"`

	// Audience is the optional JWT aud claim required for this auth entry.
	Audience string `toml:"audience"`

	// Secrets configures HS256 verification. Dashboard proxy routes sign tokens
	// with the first configured secret. The API verifies incoming tokens against
	// every secret in the ordered list so a new secret can be added before
	// removing an old one.
	Secrets []string `toml:"secrets"`

	// JWKSURL configures RS256 verification. The API fetches the JSON Web Key
	// Set from this URL during startup and verifies incoming tokens against the
	// returned RSA signing keys.
	JWKSURL string `toml:"jwks_url"`
}

func (JWTAuthConfig) authConfig() {}

func (JWTAuthConfig) authConfigType() string {
	return authTypeJWT
}

// PortalSessionAuthConfig configures portal browser-session authentication.
type PortalSessionAuthConfig struct{}

func (PortalSessionAuthConfig) authConfig() {}

func (PortalSessionAuthConfig) authConfigType() string {
	return authTypePortalSession
}

// RootKeyAuthConfig configures root-key bearer authentication.
type RootKeyAuthConfig struct {
	// Enabled may be set to true for explicitness. Setting it to false is a
	// configuration error; remove the auth entry instead.
	Enabled *bool `toml:"enabled"`
}

func (RootKeyAuthConfig) authConfig() {}

func (RootKeyAuthConfig) authConfigType() string {
	return authTypeRootKey
}

func (r RootKeyAuthConfig) enabled() bool {
	return r.Enabled == nil || *r.Enabled
}

// UnmarshalTOML decodes the auth array into concrete auth config entries.
func (a *AuthConfigs) UnmarshalTOML(v any) error {
	rawEntries, ok := v.([]map[string]any)
	if !ok {
		return fmt.Errorf("auth must be an array of tables")
	}

	entries := make(AuthConfigs, 0, len(rawEntries))
	for i, raw := range rawEntries {
		rawType, ok := raw["type"].(string)
		if !ok || strings.TrimSpace(rawType) == "" {
			return fmt.Errorf("auth[%d].type is required", i)
		}

		switch rawType {
		case authTypeJWT:
			auth, err := decodeJWTAuthConfig(i, raw)
			if err != nil {
				return err
			}
			entries = append(entries, auth)
		case authTypePortalSession:
			if err := rejectUnknownAuthFields(i, raw, "type"); err != nil {
				return err
			}
			entries = append(entries, PortalSessionAuthConfig{})
		case authTypeRootKey:
			auth, err := decodeRootKeyAuthConfig(i, raw)
			if err != nil {
				return err
			}
			entries = append(entries, auth)
		default:
			return fmt.Errorf("auth[%d].type must be one of jwt, portal_session, or root_key", i)
		}
	}

	*a = entries
	return nil
}

func decodeJWTAuthConfig(i int, raw map[string]any) (JWTAuthConfig, error) {
	if err := rejectUnknownAuthFields(i, raw, "type", "issuer", "audience", "secrets", "jwks_url"); err != nil {
		return JWTAuthConfig{}, err
	}

	auth := JWTAuthConfig{
		Issuer:   "",
		Audience: "",
		Secrets:  nil,
		JWKSURL:  "",
	}
	if rawIssuer, ok := raw["issuer"]; ok {
		issuer, ok := rawIssuer.(string)
		if !ok {
			return JWTAuthConfig{}, fmt.Errorf("auth[%d].issuer must be a string", i)
		}
		auth.Issuer = issuer
	}
	if rawAudience, ok := raw["audience"]; ok {
		audience, ok := rawAudience.(string)
		if !ok {
			return JWTAuthConfig{}, fmt.Errorf("auth[%d].audience must be a string", i)
		}
		auth.Audience = audience
	}
	if rawSecrets, ok := raw["secrets"]; ok {
		secrets, err := decodeStringSlice(rawSecrets)
		if err != nil {
			return JWTAuthConfig{}, fmt.Errorf("auth[%d].secrets must be a string array", i)
		}
		auth.Secrets = secrets
	}
	if rawJWKSURL, ok := raw["jwks_url"]; ok {
		jwksURL, ok := rawJWKSURL.(string)
		if !ok {
			return JWTAuthConfig{}, fmt.Errorf("auth[%d].jwks_url must be a string", i)
		}
		auth.JWKSURL = jwksURL
	}
	return auth, nil
}

func decodeRootKeyAuthConfig(i int, raw map[string]any) (RootKeyAuthConfig, error) {
	if err := rejectUnknownAuthFields(i, raw, "type", "enabled"); err != nil {
		return RootKeyAuthConfig{}, err
	}

	auth := RootKeyAuthConfig{Enabled: nil}
	if rawEnabled, ok := raw["enabled"]; ok {
		enabled, ok := rawEnabled.(bool)
		if !ok {
			return RootKeyAuthConfig{}, fmt.Errorf("auth[%d].enabled must be a boolean", i)
		}
		auth.Enabled = &enabled
	}
	return auth, nil
}

func rejectUnknownAuthFields(i int, raw map[string]any, allowed ...string) error {
	allowedFields := make(map[string]struct{}, len(allowed))
	for _, field := range allowed {
		allowedFields[field] = struct{}{}
	}
	for field := range raw {
		if _, ok := allowedFields[field]; !ok {
			return fmt.Errorf("auth[%d].%s is not valid for this auth type", i, field)
		}
	}
	return nil
}

func decodeStringSlice(v any) ([]string, error) {
	switch values := v.(type) {
	case []string:
		return values, nil
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			s, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("contains non-string value")
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("not a string array")
	}
}

// Config holds the complete configuration for the API server. It is designed to
// be loaded from a TOML file using [config.Load]:
//
//	cfg, err := config.Load[api.Config]("/etc/unkey/api.toml")
//
// Environment variables are expanded in file values using ${VAR}
// syntax before parsing. Struct tag defaults are applied to
// any field left at its zero value after parsing, and validation runs
// automatically via [Config.Validate].
//
// Several fields, Clock, TLSConfig, and the [TestConfig] group, are
// runtime-only and cannot be set through a config file. They are tagged
// toml:"-" and must be set programmatically after loading.
type Config struct {
	// InstanceID identifies this particular API server instance. Used in log
	// attribution, Kafka consumer group membership, and cache invalidation
	// messages so that a node can ignore its own broadcasts.
	InstanceID string `toml:"instance_id"`

	// Platform identifies the cloud platform where this node runs. Examples
	// include "aws", "gcp", "hetzner", and "kubernetes". Appears in structured
	// logs and metrics labels for filtering by infrastructure.
	Platform string `toml:"platform"`

	// Image is the container image identifier, such as "unkey/api:v1.2.3".
	// Logged at startup for correlating deployments with behavior changes.
	Image string `toml:"image"`

	// HttpPort is the TCP port the API server binds to. Ignored when
	// [TestConfig.Listener] is set, which is the case in test harnesses that
	// use ephemeral ports.
	HttpPort int `toml:"http_port" config:"default=7070,min=1,max=65535"`

	// Region is the geographic region identifier, such as "us-east-1" or "eu-west-1".
	// Included in structured logs and used by the key service when recording
	// which region served a verification request.
	Region string `toml:"region" config:"default=unknown"`

	// RedisURL is the connection string for the Redis instance backing
	// distributed rate limiting counters and usage tracking.
	// Example: "redis://redis:6379"
	RedisURL string `toml:"redis_url" config:"required,nonempty"`

	Observability config.Observability `toml:"observability"`

	// MaxRequestBodySize caps incoming request bodies at this many bytes.
	// The zen server rejects requests exceeding this limit with a 413 status.
	// Set to 0 or negative to disable the limit. Defaults to 10 MiB.
	MaxRequestBodySize int64 `toml:"max_request_body_size" config:"default=10485760"`

	// Database configures MySQL connections. See [config.DatabaseConfig].
	Database config.DatabaseConfig `toml:"database"`

	// ClickHouse configures analytics storage. See [ClickHouseConfig].
	ClickHouse ClickHouseConfig `toml:"clickhouse"`

	// TLS provides filesystem paths for HTTPS certificate and key.
	// See [config.TLSFiles].
	TLS config.TLS `toml:"tls"`

	// Vault configures the encryption/decryption service. See [config.VaultConfig].
	Vault config.VaultConfig `toml:"vault"`

	// Control configures the deployment management service. See [config.ControlConfig].
	Control config.ControlConfig `toml:"control"`

	// PortalBaseURL is the base URL for the customer portal.
	// Example: "https://portal.unkey.com"
	// Used to construct session redirect URLs in portal.createSession responses.
	// When a customer has a verified custom domain, that domain is used instead.
	PortalBaseURL string `toml:"portal_base_url" config:"default=https://portal.unkey.com"`

	// Auth configures the ordered authentication resolver chain.
	Auth AuthConfigs `toml:"auth"`

	// Pprof configures Go profiling endpoints. See [config.PprofConfig].
	// When nil (section omitted), pprof endpoints are not registered.
	Pprof *config.PprofConfig `toml:"pprof"`

	// Clock provides time operations and is injected for testability. Production
	// callers set this to [clock.New]; tests can substitute a fake clock to
	// control time progression.
	Clock clock.Clock `toml:"-"`

	// TLSConfig is the resolved [tls.Config] built from [TLSFiles.CertFile]
	// and [TLSFiles.KeyFile] at startup. This field is populated by the CLI
	// entrypoint after loading the config file and must not be set in TOML.
	TLSConfig *tls.Config `toml:"-"`

	// Test groups runtime-only overrides for integration tests. All fields are
	// zero in production and cannot be set from TOML.
	Test TestConfig `toml:"-"`
}

// TestConfig groups runtime-only flags and overrides used by integration
// tests. All fields are zero in production; setting any of them enables
// test-specific behavior that MUST NOT be reachable from a TOML config file.
type TestConfig struct {
	// Enabled relaxes certain security checks and trusts client-supplied
	// headers like X-Test-Time that would normally be rejected.
	Enabled bool

	// Counter overrides the distributed counter backend. Multi-node tests
	// share one in-memory counter across all nodes so replays sync in
	// microseconds rather than blocking on real Redis I/O.
	Counter counter.Counter

	// Listener is a pre-created net.Listener for the HTTP server. When set,
	// the server uses this listener instead of binding to HttpPort. Tests
	// use ephemeral ports (":0") to avoid conflicts when running in parallel.
	Listener net.Listener
}

// Validate checks cross-field constraints that cannot be expressed through
// struct tags alone. It implements [config.Validator] so that [config.Load]
// calls it automatically after tag-level validation.
//
// Validate rejects TLS configuration that provides only one certificate path.
func (c *Config) Validate() error {
	certFile := c.TLS.CertFile
	keyFile := c.TLS.KeyFile
	if (certFile == "") != (keyFile == "") {
		return fmt.Errorf("both tls.cert_file and tls.key_file must be provided to enable HTTPS")
	}

	for i, auth := range c.Auth {
		switch auth := auth.(type) {
		case JWTAuthConfig:
			if strings.TrimSpace(auth.Issuer) == "" {
				return fmt.Errorf("auth[%d] jwt requires issuer", i)
			}
			hasSecrets := len(auth.Secrets) > 0
			hasJWKSURL := auth.JWKSURL != ""
			if hasSecrets == hasJWKSURL {
				return fmt.Errorf("auth[%d] jwt requires exactly one of secrets or jwks_url", i)
			}
			// HS256 requires at least 256 bits of entropy in the shared secret.
			// Shorter secrets weaken signature security regardless of token lifetime.
			for j, secret := range auth.Secrets {
				if len(secret) < minJWTSecretBytes {
					return fmt.Errorf("auth[%d].secrets[%d] must be at least %d bytes, got %d", i, j, minJWTSecretBytes, len(secret))
				}
			}
			if hasJWKSURL {
				parsed, err := url.Parse(auth.JWKSURL)
				if err != nil || parsed.Scheme == "" || parsed.Host == "" {
					return fmt.Errorf("auth[%d].jwks_url must be an absolute URL", i)
				}
				if parsed.Scheme != "https" && parsed.Scheme != "http" {
					return fmt.Errorf("auth[%d].jwks_url must use http or https", i)
				}
			}
		case PortalSessionAuthConfig:
		case RootKeyAuthConfig:
			if auth.Enabled != nil && !*auth.Enabled {
				return fmt.Errorf("auth[%d].enabled=false is invalid; remove the auth entry instead", i)
			}
		default:
			return fmt.Errorf("auth[%d] has unsupported auth config type", i)
		}
	}
	return nil
}

// minJWTSecretBytes is the minimum entropy required for an HS256 signing key.
const minJWTSecretBytes = 32
