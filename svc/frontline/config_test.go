package frontline

import (
	"testing"

	"github.com/stretchr/testify/require"
	sharedconfig "github.com/unkeyed/unkey/pkg/config"
)

// TestConfig_LoadBytesParsesControlConfig guarantees Frontline reads the shared
// nested control-plane config shape used by other internal services.
func TestConfig_LoadBytesParsesControlConfig(t *testing.T) {
	t.Parallel()

	cfg, err := sharedconfig.LoadBytes[Config]([]byte(`
platform = "dev"
region = "local"
frontline_meta_signing_key = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

[control]
url = "http://control:7091"
token = "control-token"

[database]
primary = "unkey:password@tcp(mysql:3306)/unkey"
`))

	require.NoError(t, err)
	require.Equal(t, "http://control:7091", cfg.Control.URL)
	require.Equal(t, "control-token", cfg.Control.Token)
	require.Equal(t, "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", cfg.FrontlineMetaSigningKey)
}

// TestConfig_LoadBytesParsesTrustedProxyCIDRs guarantees the proxy networks
// reach zen. Without them Frontline ignores X-Forwarded-For and IP policies
// match the proxy address instead of the client.
func TestConfig_LoadBytesParsesTrustedProxyCIDRs(t *testing.T) {
	t.Parallel()

	cfg, err := sharedconfig.LoadBytes[Config]([]byte(`
platform = "dev"
region = "local"
frontline_meta_signing_key = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
trusted_proxy_cidrs = ["10.28.80.0/20", "192.0.2.0/24"]

[control]
url = "http://control:7091"
token = "control-token"

[database]
primary = "unkey:password@tcp(mysql:3306)/unkey"
`))

	require.NoError(t, err)
	require.Equal(t, []string{"10.28.80.0/20", "192.0.2.0/24"}, cfg.TrustedProxyCIDRs)
}
