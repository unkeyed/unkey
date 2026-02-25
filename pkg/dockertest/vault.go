package dockertest

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	vaultkeys "github.com/unkeyed/unkey/pkg/vault/keys"
)

const (
	unkeyImage     = "unkey/unkey:latest"
	unkeyImageLoad = "//:unkey_image_load"
	vaultPort      = "8060/tcp"
	vaultBucket    = "vault-test"
	vaultToken     = "vault-test-token"
)

var (
	unkeyImageOnce sync.Once
	unkeyImageErr  error
)

// VaultConfig holds connection information for a vault test container.
type VaultConfig struct {
	// URL is the vault endpoint URL (e.g., "http://localhost:54321").
	URL string

	// Token is the bearer token used to authenticate requests.
	Token string
}

// Vault starts a vault service container configured to use the provided S3.
//
// The container image is built once on demand and cached locally. The service
// reads its config from a temp TOML file mounted into the container and expands
// env vars so each container can inject its own S3 credentials and master key at runtime.
func Vault(t *testing.T, s3 S3Config, network *Network) VaultConfig {
	t.Helper()

	_, masterKey, err := vaultkeys.GenerateMasterKey()
	require.NoError(t, err)

	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "vault.toml")
	require.NoError(t, os.WriteFile(configPath, []byte(vaultConfigTemplate), 0o644))

	ctr := startContainer(t, containerConfig{
		Image:        unkeyImage,
		ExposedPorts: []string{vaultPort},
		Env: map[string]string{
			"UNKEY_CONFIG":         "/etc/unkey/vault.toml",
			"VAULT_BEARER_TOKEN":   vaultToken,
			"VAULT_MASTER_KEY":     masterKey,
			"S3_URL":               s3.ContainerURL,
			"S3_BUCKET":            vaultBucket,
			"S3_ACCESS_KEY_ID":     s3.AccessKeyID,
			"S3_ACCESS_KEY_SECRET": s3.SecretAccessKey,
		},
		Cmd:          []string{"run", "vault"},
		WaitStrategy: NewHTTPWait(vaultPort, "/health/ready"),
		WaitTimeout:  30 * time.Second,
		Tmpfs:        nil,
		Binds:        []string{fmt.Sprintf("%s:/etc/unkey/vault.toml:ro", configPath)},
		Keep:         true,
		NetworkName:  networkName(network),
	})

	return VaultConfig{
		URL:   ctr.HostURL("http", vaultPort),
		Token: vaultToken,
	}
}

const vaultConfigTemplate = `instance_id = "vault-test"
http_port = 8060
bearer_token = "${VAULT_BEARER_TOKEN}"

[encryption]
master_key = "${VAULT_MASTER_KEY}"

[s3]
url = "${S3_URL}"
bucket = "${S3_BUCKET}"
access_key_id = "${S3_ACCESS_KEY_ID}"
access_key_secret = "${S3_ACCESS_KEY_SECRET}"

[observability]
`
