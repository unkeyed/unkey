package dockertest

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
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
	HostURL   string
	DockerURL string
	Token     string
}

// Vault starts a vault service container configured to use the provided S3.
func (c *Cluster) Vault(s3 S3Config) VaultConfig {
	c.t.Helper()

	_, masterKey, err := vaultkeys.GenerateMasterKey()
	require.NoError(c.t, err)

	configDir := c.t.TempDir()
	configPath := filepath.Join(configDir, "vault.toml")
	require.NoError(c.t, os.WriteFile(configPath, []byte(vaultConfigTemplate), 0o644))

	ctr, cleanup, err := startContainer(c.cli, containerConfig{
		ContainerName: "",
		Image:         unkeyImage,
		ExposedPorts:  []string{vaultPort},
		Env: map[string]string{
			"UNKEY_CONFIG":         "/etc/unkey/vault.toml",
			"VAULT_BEARER_TOKEN":   vaultToken,
			"VAULT_MASTER_KEY":     masterKey,
			"S3_URL":               s3.DockerURL,
			"S3_BUCKET":            vaultBucket,
			"S3_ACCESS_KEY_ID":     s3.AccessKeyID,
			"S3_ACCESS_KEY_SECRET": s3.SecretAccessKey,
		},
		Cmd:         []string{"run", "vault"},
		Tmpfs:       nil,
		Binds:       []string{fmt.Sprintf("%s:/etc/unkey/vault.toml:ro", configPath)},
		Keep:        true,
		NetworkName: c.network.Name,
	}, c.t.Name())
	require.NoError(c.t, err)
	if cleanup != nil {
		c.t.Cleanup(func() { require.NoError(c.t, cleanup()) })
	}

	wait := NewHTTPWait(vaultPort, "/health/ready")
	wait.Wait(c.t, ctr, 30*time.Second)

	port := ctr.Port(vaultPort)
	require.NotEmpty(c.t, port, "vault port not mapped")

	return VaultConfig{
		HostURL:   fmt.Sprintf("http://%s:%s", ctr.Host, port),
		DockerURL: fmt.Sprintf("http://%s:%s", ctr.ContainerName, containerPortNumber(vaultPort)),
		Token:     vaultToken,
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
