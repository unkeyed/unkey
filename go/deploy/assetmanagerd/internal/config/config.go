package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config represents the complete configuration for assetmanagerd
type Config struct {
	// Service configuration
	Port    int    `env:"UNKEY_ASSETMANAGERD_PORT" envDefault:"8083"`
	Address string `env:"UNKEY_ASSETMANAGERD_ADDRESS" envDefault:"0.0.0.0"`

	// Storage configuration
	StorageBackend   string `env:"UNKEY_ASSETMANAGERD_STORAGE_BACKEND" envDefault:"local"` // local, s3, nfs
	LocalStoragePath string `env:"UNKEY_ASSETMANAGERD_LOCAL_STORAGE_PATH" envDefault:"/opt/vm-assets"`
	DatabasePath     string `env:"UNKEY_ASSETMANAGERD_DATABASE_PATH" envDefault:"/opt/assetmanagerd/assets.db"`
	CacheDir         string `env:"UNKEY_ASSETMANAGERD_CACHE_DIR" envDefault:"/opt/assetmanagerd/cache"`

	// S3 configuration (if backend is s3)
	S3Bucket          string `env:"UNKEY_ASSETMANAGERD_S3_BUCKET"`
	S3Region          string `env:"UNKEY_ASSETMANAGERD_S3_REGION" envDefault:"us-east-1"`
	S3Endpoint        string `env:"UNKEY_ASSETMANAGERD_S3_ENDPOINT"` // For S3-compatible services
	S3AccessKeyID     string `env:"UNKEY_ASSETMANAGERD_S3_ACCESS_KEY_ID"`
	S3SecretAccessKey string `env:"UNKEY_ASSETMANAGERD_S3_SECRET_ACCESS_KEY"`

	// Garbage collection configuration
	GCEnabled       bool          `env:"UNKEY_ASSETMANAGERD_GC_ENABLED" envDefault:"true"`
	GCInterval      time.Duration `env:"UNKEY_ASSETMANAGERD_GC_INTERVAL" envDefault:"1h"`
	GCMaxAge        time.Duration `env:"UNKEY_ASSETMANAGERD_GC_MAX_AGE" envDefault:"168h"` // 7 days
	GCMinReferences int           `env:"UNKEY_ASSETMANAGERD_GC_MIN_REFERENCES" envDefault:"0"`

	// Asset limits
	MaxAssetSize int64         `env:"UNKEY_ASSETMANAGERD_MAX_ASSET_SIZE" envDefault:"10737418240"`  // 10GB
	MaxCacheSize int64         `env:"UNKEY_ASSETMANAGERD_MAX_CACHE_SIZE" envDefault:"107374182400"` // 100GB
	AssetTTL     time.Duration `env:"UNKEY_ASSETMANAGERD_ASSET_TTL" envDefault:"0"`                 // 0 = no TTL

	// Performance tuning
	DownloadConcurrency int           `env:"UNKEY_ASSETMANAGERD_DOWNLOAD_CONCURRENCY" envDefault:"4"`
	DownloadTimeout     time.Duration `env:"UNKEY_ASSETMANAGERD_DOWNLOAD_TIMEOUT" envDefault:"30m"`

	// OpenTelemetry configuration
	OTELEnabled             bool    `env:"UNKEY_ASSETMANAGERD_OTEL_ENABLED" envDefault:"true"`
	OTELServiceName         string  `env:"UNKEY_ASSETMANAGERD_OTEL_SERVICE_NAME" envDefault:"assetmanagerd"`
	OTELServiceVersion      string  `env:"UNKEY_ASSETMANAGERD_OTEL_SERVICE_VERSION" envDefault:"0.2.0"`
	OTELEndpoint            string  `env:"UNKEY_ASSETMANAGERD_OTEL_ENDPOINT" envDefault:"localhost:4318"`
	OTELSamplingRate        float64 `env:"UNKEY_ASSETMANAGERD_OTEL_SAMPLING_RATE" envDefault:"1.0"`
	OTELPrometheusPort      int     `env:"UNKEY_ASSETMANAGERD_OTEL_PROMETHEUS_PORT" envDefault:"9467"`
	OTELPrometheusEnabled   bool    `env:"UNKEY_ASSETMANAGERD_OTEL_PROMETHEUS_ENABLED" envDefault:"true"`
	OTELPrometheusInterface string  `env:"UNKEY_ASSETMANAGERD_OTEL_PROMETHEUS_INTERFACE" envDefault:"127.0.0.1"`

	// TLS configuration
	// AIDEV-BUSINESS_RULE: SPIFFE/mTLS is required by default for security - no fallback to disabled mode
	TLSMode             string `env:"UNKEY_ASSETMANAGERD_TLS_MODE" envDefault:"spiffe"`
	TLSCertFile         string `env:"UNKEY_ASSETMANAGERD_TLS_CERT_FILE"`
	TLSKeyFile          string `env:"UNKEY_ASSETMANAGERD_TLS_KEY_FILE"`
	TLSCAFile           string `env:"UNKEY_ASSETMANAGERD_TLS_CA_FILE"`
	TLSSPIFFESocketPath string `env:"UNKEY_ASSETMANAGERD_SPIFFE_SOCKET" envDefault:"/var/lib/spire/agent/agent.sock"`

	// Builderd integration configuration
	// AIDEV-NOTE: When enabled, assetmanagerd will automatically trigger builderd to create missing assets
	BuilderdEnabled      bool          `env:"UNKEY_ASSETMANAGERD_BUILDERD_ENABLED" envDefault:"true"`
	BuilderdEndpoint     string        `env:"UNKEY_ASSETMANAGERD_BUILDERD_ENDPOINT" envDefault:"https://localhost:8082"`
	BuilderdTimeout      time.Duration `env:"UNKEY_ASSETMANAGERD_BUILDERD_TIMEOUT" envDefault:"30m"`
	BuilderdAutoRegister bool          `env:"UNKEY_ASSETMANAGERD_BUILDERD_AUTO_REGISTER" envDefault:"true"`
	BuilderdMaxRetries   int           `env:"UNKEY_ASSETMANAGERD_BUILDERD_MAX_RETRIES" envDefault:"3"`
	BuilderdRetryDelay   time.Duration `env:"UNKEY_ASSETMANAGERD_BUILDERD_RETRY_DELAY" envDefault:"5s"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	//nolint:exhaustruct // Config fields will be populated by environment variables
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// AIDEV-NOTE: Comprehensive validation ensures service reliability from startup
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.OTELPrometheusPort < 1 || c.OTELPrometheusPort > 65535 {
		return fmt.Errorf("invalid prometheus port: %d", c.OTELPrometheusPort)
	}

	// Validate storage backend
	switch c.StorageBackend {
	case "local":
		if c.LocalStoragePath == "" {
			return fmt.Errorf("local storage path is required for local backend")
		}
	case "s3":
		if c.S3Bucket == "" {
			return fmt.Errorf("S3 bucket is required for s3 backend")
		}
		if c.S3AccessKeyID == "" || c.S3SecretAccessKey == "" {
			return fmt.Errorf("S3 credentials are required for s3 backend")
		}
	case "nfs":
		// NFS validation would go here
		return fmt.Errorf("NFS backend not yet implemented")
	default:
		return fmt.Errorf("unsupported storage backend: %s", c.StorageBackend)
	}

	// Validate GC settings
	if c.GCEnabled && c.GCInterval < time.Minute {
		return fmt.Errorf("GC interval must be at least 1 minute")
	}

	// Validate size limits
	if c.MaxAssetSize <= 0 {
		return fmt.Errorf("max asset size must be positive")
	}

	if c.MaxCacheSize < c.MaxAssetSize {
		return fmt.Errorf("max cache size must be at least as large as max asset size")
	}

	// Validate OTEL settings
	if c.OTELEnabled && c.OTELSamplingRate < 0 || c.OTELSamplingRate > 1 {
		return fmt.Errorf("OTEL sampling rate must be between 0 and 1")
	}

	// Validate builderd configuration
	if c.BuilderdEnabled {
		if c.BuilderdEndpoint == "" {
			return fmt.Errorf("builderd endpoint is required when builderd integration is enabled")
		}
		if c.BuilderdTimeout < time.Minute {
			return fmt.Errorf("builderd timeout must be at least 1 minute")
		}
		if c.BuilderdMaxRetries < 0 {
			return fmt.Errorf("builderd max retries must be non-negative")
		}
	}

	return nil
}
