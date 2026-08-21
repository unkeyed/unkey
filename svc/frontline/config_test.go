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
frontline_meta_signing_key = "frontline-meta-signing-key"

[control]
url = "http://control:7091"
token = "control-token"

[database]
primary = "unkey:password@tcp(mysql:3306)/unkey"
`))

	require.NoError(t, err)
	require.Equal(t, "http://control:7091", cfg.Control.URL)
	require.Equal(t, "control-token", cfg.Control.Token)
	require.Equal(t, "frontline-meta-signing-key", cfg.FrontlineMetaSigningKey)
}
