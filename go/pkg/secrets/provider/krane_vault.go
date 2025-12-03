package provider

import (
	"context"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	"github.com/unkeyed/unkey/go/gen/proto/krane/v1/kranev1connect"
	"github.com/unkeyed/unkey/go/pkg/assert"
)

// KraneVaultProvider fetches secrets via Krane's SecretsService.
// Krane handles token validation and calls Vault for decryption.
type KraneVaultProvider struct {
	client   kranev1connect.SecretsServiceClient
	endpoint string
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

	return &KraneVaultProvider{
		client:   client,
		endpoint: cfg.Endpoint,
	}, nil
}

// Name returns the provider name.
func (p *KraneVaultProvider) Name() string {
	return string(KraneVault)
}

// FetchSecrets retrieves secrets from Krane (which decrypts via Vault).
// If EncryptedBlob is provided, uses DecryptSecretsBlob RPC (no DB lookup).
// Otherwise falls back to GetDeploymentSecrets (requires DB lookup).
func (p *KraneVaultProvider) FetchSecrets(ctx context.Context, opts FetchOptions) (map[string]string, error) {
	token := opts.Token

	// If TokenPath is set, read token from file (e.g., K8s service account token)
	if opts.TokenPath != "" {
		tokenBytes, err := os.ReadFile(opts.TokenPath)
		if err != nil {
			return nil, err
		}
		token = string(tokenBytes)
	}

	if err := assert.All(
		assert.NotEmpty(token, "token is required"),
		assert.NotEmpty(opts.DeploymentID, "deployment_id is required"),
	); err != nil {
		return nil, err
	}

	// Use DecryptSecretsBlob if we have an encrypted blob (preferred - no DB lookup)
	if len(opts.EncryptedBlob) > 0 {
		if err := assert.NotEmpty(opts.EnvironmentID, "environment_id is required for blob decryption"); err != nil {
			return nil, err
		}

		resp, err := p.client.DecryptSecretsBlob(ctx, connect.NewRequest(&kranev1.DecryptSecretsBlobRequest{
			EncryptedBlob: opts.EncryptedBlob,
			EnvironmentId: opts.EnvironmentID,
			Token:         token,
			DeploymentId:  opts.DeploymentID,
		}))
		if err != nil {
			return nil, err
		}
		return resp.Msg.GetEnvVars(), nil
	}

	// Fallback to GetDeploymentSecrets (requires DB lookup in krane)
	resp, err := p.client.GetDeploymentSecrets(ctx, connect.NewRequest(&kranev1.GetDeploymentSecretsRequest{
		DeploymentId: opts.DeploymentID,
		Token:        token,
	}))
	if err != nil {
		return nil, err
	}

	return resp.Msg.GetEnvVars(), nil
}
