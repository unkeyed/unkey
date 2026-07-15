package vault

import (
	"errors"

	"github.com/unkeyed/unkey/pkg/config"
)

// EncryptionConfig holds the master keys used for encryption and decryption.
type EncryptionConfig struct {
	// MasterKey is the current key used for encrypting new data.
	MasterKey string `toml:"master_key" config:"required,nonempty"`

	// PreviousMasterKey is an optional old key retained for decrypting
	// existing data during key rotation.
	PreviousMasterKey *string `toml:"previous_master_key"`
}

// StorageConfig selects the backend used to persist encrypted secrets.
// Exactly one of S3 or Disk must be set.
type StorageConfig struct {
	// S3 configures an S3-compatible object storage backend. See [S3Config].
	S3 *S3Config `toml:"s3"`

	// Disk configures a local filesystem backend, intended for local
	// development. See [DiskConfig].
	Disk *DiskConfig `toml:"disk"`
}

// S3Config configures the S3-compatible object storage backend used by vault to
// persist encrypted secrets. All fields are required.
type S3Config struct {
	// URL is the S3-compatible endpoint URL.
	// Example: "http://s3:3902"
	URL string `toml:"url" config:"required,nonempty"`

	// Bucket is the S3 bucket name for storing encrypted secrets.
	Bucket string `toml:"bucket" config:"required,nonempty"`

	// AccessKeyID is the access key ID for authenticating with S3.
	AccessKeyID string `toml:"access_key_id" config:"required,nonempty"`

	// AccessKeySecret is the secret access key for authenticating with S3.
	AccessKeySecret string `toml:"access_key_secret" config:"required,nonempty"`
}

// DiskConfig configures a local filesystem storage backend. Encrypted secrets
// are written under Path using the same key layout as the S3 backend. Use
// this for local development to avoid running a minio container; do not use
// in production.
type DiskConfig struct {
	// Path is the directory where encrypted secrets are persisted. Created
	// on startup if it does not already exist.
	Path string `toml:"path" config:"required,nonempty"`
}

// Config holds the complete configuration for the vault service. It is designed
// to be loaded from a TOML file using [config.Load]:
//
//	cfg, err := config.Load[vault.Config]("/etc/unkey/vault.toml")
//
// Environment variables are expanded in file values using ${VAR}
// syntax before parsing.
type Config struct {
	// InstanceID identifies this particular vault instance.
	InstanceID string `toml:"instance_id"`

	// HttpPort is the TCP port the vault server binds to.
	HttpPort int `toml:"http_port" config:"default=8060,min=1,max=65535"`

	// Region is the geographic region identifier (e.g. "us-east-1").
	// Included in structured logs and OpenTelemetry attributes.
	Region string `toml:"region"`

	// BearerToken is the authentication token for securing vault operations.
	BearerToken string `toml:"bearer_token" config:"required,nonempty"`

	// Encryption holds the master keys for encrypting and decrypting data.
	Encryption EncryptionConfig `toml:"encryption"`

	// Storage selects the persistence backend. Exactly one of [storage.s3]
	// or [storage.disk] must be set.
	Storage StorageConfig `toml:"storage"`

	// Observability configures tracing, logging, and metrics. See [config.Observability].
	Observability config.Observability `toml:"observability"`
}

// Validate implements [config.Validator] so that [config.Load] calls it
// automatically after tag-level validation.
func (c *Config) Validate() error {
	if c.Storage.S3 == nil && c.Storage.Disk == nil {
		return errors.New("storage: must set either [storage.s3] or [storage.disk]")
	}
	if c.Storage.S3 != nil && c.Storage.Disk != nil {
		return errors.New("storage: set only one of [storage.s3] or [storage.disk]")
	}
	return nil
}
