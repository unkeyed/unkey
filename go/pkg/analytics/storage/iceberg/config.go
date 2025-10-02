package iceberg

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/vault"
)

// Config contains data lake-specific configuration for analytics.
type Config struct {
	// Endpoint is the S3-compatible storage endpoint URL (e.g., R2, S3, MinIO, etc.)
	Endpoint string `json:"endpoint"`

	// CatalogEndpoint is the optional Iceberg REST Catalog API endpoint (catalog URI)
	// Leave empty to skip catalog commits (files will still be uploaded to S3)
	CatalogEndpoint string `json:"catalog_endpoint,omitempty"`

	// CatalogToken is the authentication token for the Iceberg catalog
	// For R2, this is the R2 API token value (not encrypted - same as S3 credentials)
	CatalogToken string `json:"catalog_token,omitempty"`

	// Region for the data lake storage
	Region string `json:"region"`

	// Credentials for accessing the data lake
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`

	// Bucket is the full bucket name for this workspace
	Bucket string `json:"bucket"`

	// Format specifies the data format (e.g., "iceberg", "parquet")
	Format string `json:"format"`
}

// WorkspaceConfig contains the workspace-specific Iceberg configuration
// that is stored in the database analytics_config.config JSON field.
// Sensitive fields are vault-encrypted and prefixed with "encrypted" in the field name.
type WorkspaceConfig struct {
	// Bucket is the full bucket name for this workspace's analytics data
	Bucket string `json:"bucket"`

	// Endpoint is the S3-compatible storage endpoint URL (e.g., R2, S3, MinIO, etc.)
	Endpoint string `json:"endpoint,omitempty"`

	// CatalogEndpoint is the optional Iceberg REST Catalog API endpoint (catalog URI)
	// Leave empty for R2 Data Catalog or if using S3-only (no catalog commits)
	CatalogEndpoint string `json:"catalogEndpoint,omitempty"`

	// Region is the cloud region (optional, depends on provider)
	Region string `json:"region,omitempty"`

	// EncryptedAccessKeyId - vault-encrypted access key ID (stored in DB)
	EncryptedAccessKeyId string `json:"encryptedAccessKeyId,omitempty"`

	// EncryptedSecretAccessKey - vault-encrypted secret access key (stored in DB)
	EncryptedSecretAccessKey string `json:"encryptedSecretAccessKey,omitempty"`

	// EncryptedCatalogToken - vault-encrypted catalog token (R2 API token for catalog auth)
	EncryptedCatalogToken string `json:"encryptedCatalogToken,omitempty"`

	// AccessKeyId - decrypted access key ID (populated after Decrypt)
	AccessKeyId string `json:"-"`

	// SecretAccessKey - decrypted secret access key (populated after Decrypt)
	SecretAccessKey string `json:"-"`

	// CatalogToken - decrypted catalog token (populated after Decrypt)
	CatalogToken string `json:"-"`

	// Format is the data format (iceberg, parquet). Default: iceberg
	Format string `json:"format,omitempty"`

	// RetentionDays is the number of days to retain data before expiring snapshots.
	// Default: 30 days (1 month). Customers can configure longer retention (e.g., 90, 365 days).
	// This controls the 'history.expire.max-snapshot-age-ms' table property.
	RetentionDays int `json:"retentionDays,omitempty"`

	// CatalogID is the R2 Data Catalog ID (for R2 provider with Iceberg enabled)
	CatalogID string `json:"catalogId,omitempty"`
}

// Decrypt decrypts the vault-encrypted fields in the config.
// After calling this method, AccessKeyId, SecretAccessKey, and CatalogToken will be populated
// with the decrypted values.
func (c *WorkspaceConfig) Decrypt(ctx context.Context, workspaceId string, vault *vault.Service) error {
	if c.EncryptedAccessKeyId != "" {
		decrypted, err := vault.Decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   workspaceId,
			Encrypted: c.EncryptedAccessKeyId,
		})
		if err != nil {
			return fmt.Errorf("failed to decrypt access key id: %w", err)
		}

		c.AccessKeyId = decrypted.GetPlaintext()
	}

	if c.EncryptedSecretAccessKey != "" {
		decrypted, err := vault.Decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   workspaceId,
			Encrypted: c.EncryptedSecretAccessKey,
		})
		if err != nil {
			return fmt.Errorf("failed to decrypt secret access key: %w", err)
		}

		c.SecretAccessKey = decrypted.GetPlaintext()
	}

	if c.EncryptedCatalogToken != "" {
		decrypted, err := vault.Decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   workspaceId,
			Encrypted: c.EncryptedCatalogToken,
		})
		if err != nil {
			return fmt.Errorf("failed to decrypt catalog token: %w", err)
		}

		c.CatalogToken = decrypted.GetPlaintext()
	}

	return nil
}
