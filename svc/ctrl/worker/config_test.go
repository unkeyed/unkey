package worker

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
)

func TestConfigDatabase(t *testing.T) {
	t.Run("accepts flat database string", func(t *testing.T) {
		cfg, err := config.LoadBytes[Config]([]byte(`
cname_domain = "unkey.local"
database = "unkey:password@tcp(mysql:3306)/unkey?parseTime=true"

[vault]
url = "http://vault:8060"
token = "vault-token"
`))

		require.NoError(t, err)
		require.Equal(t, "unkey:password@tcp(mysql:3306)/unkey?parseTime=true", cfg.Database)
	})

	t.Run("rejects nested database table", func(t *testing.T) {
		_, err := config.LoadBytes[Config]([]byte(`
cname_domain = "unkey.local"

[database]
primary = "unkey:password@tcp(mysql:3306)/unkey?parseTime=true"

[vault]
url = "http://vault:8060"
token = "vault-token"
`))

		require.Error(t, err)
	})
}

func TestConfigBuildBackend(t *testing.T) {
	base := `
cname_domain = "unkey.local"
database = "unkey:password@tcp(mysql:3306)/unkey?parseTime=true"

[vault]
url = "http://vault:8060"
token = "vault-token"
`

	t.Run("defaults to depot", func(t *testing.T) {
		cfg, err := config.LoadBytes[Config]([]byte(base))
		require.NoError(t, err)
		require.Equal(t, "depot", cfg.Build.Backend)
	})

	t.Run("rejects unknown backend", func(t *testing.T) {
		_, err := config.LoadBytes[Config]([]byte(base + `
[build]
backend = "docker"
`))
		require.ErrorContains(t, err, "invalid build backend")
	})

	t.Run("kubernetes backend requires registry repository", func(t *testing.T) {
		_, err := config.LoadBytes[Config]([]byte(base + `
[build]
backend = "kubernetes"
`))
		require.ErrorContains(t, err, "registry repository is required")
	})

	t.Run("kubernetes backend accepts defaults with repository", func(t *testing.T) {
		cfg, err := config.LoadBytes[Config]([]byte(base + `
[build]
backend = "kubernetes"

[registry]
repository = "ctlptl-registry:5000/deployments"
insecure = true
`))
		require.NoError(t, err)
		require.Equal(t, "unkey", cfg.Build.Kubernetes.Namespace)
		require.Equal(t, "moby/buildkit:v0.26.3", cfg.Build.Kubernetes.Image)
		require.True(t, cfg.Registry.Insecure)
	})

	t.Run("depot backend keeps password-gated validation", func(t *testing.T) {
		_, err := config.LoadBytes[Config]([]byte(base + `
[registry]
repository = "registry.depot.dev/repo"
username = "x-token"
password = "depot-token"
`))
		require.ErrorContains(t, err, "Depot API URL is required")
	})

	t.Run("depot backend accepts nested build.depot config", func(t *testing.T) {
		cfg, err := config.LoadBytes[Config]([]byte(base + `
[build.depot]
api_url = "https://api.depot.dev"

[registry]
repository = "registry.depot.dev/repo"
username = "x-token"
password = "depot-token"
`))
		require.NoError(t, err)
		require.Equal(t, "https://api.depot.dev", cfg.Build.Depot.APIUrl)
		require.Equal(t, "us-east-1", cfg.Build.Depot.ProjectRegion)
	})
}
