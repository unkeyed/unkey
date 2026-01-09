// Package provider defines the interface for secrets providers.
// This abstraction allows inject to fetch secrets from different backends.
package provider

import (
	"context"
	"fmt"
)

// Type represents a secrets provider type.
type Type string

const (
	// KraneVault fetches secrets via Krane's SecretsService which decrypts via Vault.
	KraneVault Type = "krane-vault"
)

// Provider fetches secrets for a deployment.
type Provider interface {
	// Name returns the provider name for logging/debugging.
	Name() string

	// FetchSecrets retrieves decrypted secrets for a deployment.
	// Returns a map of environment variable names to their values.
	FetchSecrets(ctx context.Context, opts FetchOptions) (map[string]string, error)
}

// FetchOptions contains parameters for fetching secrets.
type FetchOptions struct {
	// DeploymentID is the deployment to fetch secrets for.
	DeploymentID string

	// EnvironmentID is the environment (keyring) for decryption.
	EnvironmentID string

	// Token is the authentication token.
	Token string

	// TokenPath is an optional path to read the token from (e.g., K8s service account token).
	// If set, Token is ignored and the token is read from this file.
	TokenPath string

	// Encrypted is the encrypted secrets blob from UNKEY_ENCRYPTED_ENV env var.
	// If set, uses DecryptSecretsBlob RPC instead of GetDeploymentSecrets.
	Encrypted []byte
}

// Config holds configuration for creating a provider.
type Config struct {
	// Type: KraneVault
	Type Type

	// Endpoint is the provider's API endpoint.
	Endpoint string
}

// ErrProviderNotFound is returned when an unknown provider type is requested.
var ErrProviderNotFound = fmt.Errorf("provider not found")

// New creates a provider based on the config type.
func New(cfg Config) (Provider, error) {
	switch cfg.Type {
	case KraneVault:
		return NewKraneVaultProvider(cfg)
	default:
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, cfg.Type)
	}
}
