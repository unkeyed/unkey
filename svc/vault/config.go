package vault

import (
	"github.com/unkeyed/unkey/pkg/config"
)

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

// Config holds the complete configuration for the vault service. It is designed
// to be loaded from a TOML file using [config.Load]:
//
//	cfg, err := config.Load[vault.Config]("/etc/unkey/vault.toml")
//
// Environment variables are expanded in file values using ${VAR} or
// ${VAR:-default} syntax before parsing.
type Config struct {
	// InstanceID identifies this particular vault instance. Used in log
	// attribution and observability labels.
	InstanceID string `toml:"instance_id" config:"required,nonempty"`

	// HttpPort is the TCP port the vault server binds to.
	HttpPort int `toml:"http_port" config:"default=8060,min=1,max=65535"`

	// Region is the geographic region identifier (e.g. "us-east-1").
	// Included in structured logs and OpenTelemetry attributes.
	Region string `toml:"region"`

	// BearerToken is the authentication token for securing vault operations.
	BearerToken string `toml:"bearer_token" config:"required,nonempty"`

	// MasterKeys holds encryption keys for the vault. The first key is used
	// for encryption; additional keys are retained for backwards-compatible
	// decryption. If multiple keys are provided, vault will start a rekey
	// process to migrate all secrets to the new key.
	MasterKeys []string `toml:"master_keys" config:"required,nonempty"`

	// S3 configures the S3-compatible storage backend. See [S3Config].
	S3 S3Config `toml:"s3"`

	Observability config.Observability `toml:"observability"`

	// Logging configures log sampling. See [config.LoggingConfig].
	Logging config.LoggingConfig `toml:"logging"`
}

// Validate implements [config.Validator] so that [config.Load] calls it
// automatically after tag-level validation. All constraints are expressed
// through struct tags, so this method has nothing additional to check.
func (c *Config) Validate() error {
	return nil
}
