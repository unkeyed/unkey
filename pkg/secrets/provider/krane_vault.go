package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	kranev1 "github.com/unkeyed/unkey/gen/proto/krane/v1"
	"github.com/unkeyed/unkey/gen/proto/krane/v1/kranev1connect"
	"github.com/unkeyed/unkey/pkg/assert"
)

// KraneVaultProvider fetches secrets via Krane's SecretsService.
// Krane handles token validation and calls Vault for decryption.
type KraneVaultProvider struct {
	client kranev1connect.SecretsServiceClient
}

// NewKraneVaultProvider creates a new Krane-Vault secrets provider.
func NewKraneVaultProvider(cfg Config) (*KraneVaultProvider, error) {
	if err := assert.NotEmpty(cfg.Endpoint, "krane-vault provider requires endpoint"); err != nil {
		return nil, err
	}

	client := kranev1connect.NewSecretsServiceClient(
		&http.Client{Timeout: 30 * time.Second},
		cfg.Endpoint,
	)

	return &KraneVaultProvider{client: client}, nil
}

// Name returns the provider name.
func (p *KraneVaultProvider) Name() string {
	return string(KraneVault)
}

// FetchSecrets retrieves secrets from Krane (which decrypts via Vault).
func (p *KraneVaultProvider) FetchSecrets(ctx context.Context, opts FetchOptions) (map[string]string, error) {
	if len(opts.Encrypted) == 0 {
		return make(map[string]string), nil
	}

	token, err := resolveToken(opts)
	if err != nil {
		return nil, err
	}

	if err := assert.All(
		assert.NotEmpty(opts.DeploymentID, "deployment_id is required"),
		assert.NotEmpty(opts.EnvironmentID, "environment_id is required for blob decryption"),
	); err != nil {
		return nil, err
	}

	resp, err := p.client.DecryptSecretsBlob(ctx, connect.NewRequest(&kranev1.DecryptSecretsBlobRequest{
		EncryptedBlob: opts.Encrypted,
		EnvironmentId: opts.EnvironmentID,
		Token:         token,
		DeploymentId:  opts.DeploymentID,
	}))
	if err != nil {
		return nil, err
	}

	return resp.Msg.GetEnvVars(), nil
}

func resolveToken(opts FetchOptions) (string, error) {
	if opts.TokenPath != "" {
		tokenBytes, err := os.ReadFile(opts.TokenPath)
		if err != nil {
			return "", fmt.Errorf("failed to read token file: %w", err)
		}
		token := strings.TrimSpace(string(tokenBytes))
		if err := assert.NotEmpty(token, "token file exists but is empty"); err != nil {
			return "", err
		}
		return token, nil
	}

	if err := assert.NotEmpty(opts.Token, "token is required"); err != nil {
		return "", err
	}

	return opts.Token, nil
}
