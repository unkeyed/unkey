package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

// Config holds the complete builderd configuration
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Builder       BuilderConfig       `yaml:"builder"`
	Storage       StorageConfig       `yaml:"storage"`
	Docker        DockerConfig        `yaml:"docker"`
	Tenant        TenantConfig        `yaml:"tenant"`
	Database      DatabaseConfig      `yaml:"database"`
	OpenTelemetry OpenTelemetryConfig `yaml:"opentelemetry"`
	TLS           *TLSConfig          `yaml:"tls,omitempty"`
	AssetManager  AssetManagerConfig  `yaml:"assetmanager"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Address         string        `yaml:"address"`
	Port            string        `yaml:"port"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	RateLimit       int           `yaml:"rate_limit"` // Requests per second for health endpoint
}

// BuilderConfig holds build execution configuration
type BuilderConfig struct {
	MaxConcurrentBuilds int           `yaml:"max_concurrent_builds"`
	BuildTimeout        time.Duration `yaml:"build_timeout"`
	ScratchDir          string        `yaml:"scratch_dir"`
	RootfsOutputDir     string        `yaml:"rootfs_output_dir"`
	WorkspaceDir        string        `yaml:"workspace_dir"`
	CleanupInterval     time.Duration `yaml:"cleanup_interval"`
	UsePipelineExecutor bool          `yaml:"use_pipeline_executor"` // Feature flag for step-based execution
}

// StorageConfig holds storage backend configuration
type StorageConfig struct {
	Backend        string    `yaml:"backend"` // "local", "s3", "gcs"
	RetentionDays  int       `yaml:"retention_days"`
	MaxSizeGB      int       `yaml:"max_size_gb"`
	CacheEnabled   bool      `yaml:"cache_enabled"`
	CacheMaxSizeGB int       `yaml:"cache_max_size_gb"`
	S3Config       S3Config  `yaml:"s3,omitempty"`
	GCSConfig      GCSConfig `yaml:"gcs,omitempty"`
}

// S3Config holds S3 storage configuration
type S3Config struct {
	Bucket    string `yaml:"bucket"`
	Region    string `yaml:"region"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Endpoint  string `yaml:"endpoint,omitempty"` // For S3-compatible services
}

// GCSConfig holds Google Cloud Storage configuration
type GCSConfig struct {
	Bucket          string `yaml:"bucket"`
	Project         string `yaml:"project"`
	CredentialsPath string `yaml:"credentials_path"`
}

// DockerConfig holds Docker-related configuration
type DockerConfig struct {
	RegistryAuth       bool          `yaml:"registry_auth"`
	MaxImageSizeGB     int           `yaml:"max_image_size_gb"`
	AllowedRegistries  []string      `yaml:"allowed_registries"`
	PullTimeout        time.Duration `yaml:"pull_timeout"`
	RegistryMirror     string        `yaml:"registry_mirror,omitempty"`
	InsecureRegistries []string      `yaml:"insecure_registries,omitempty"`
}

// TenantConfig holds multi-tenancy configuration
type TenantConfig struct {
	DefaultTier           string         `yaml:"default_tier"`
	IsolationEnabled      bool           `yaml:"isolation_enabled"`
	QuotaCheckInterval    time.Duration  `yaml:"quota_check_interval"`
	DefaultResourceLimits ResourceLimits `yaml:"default_resource_limits"`
}

// ResourceLimits defines default resource limits per tenant tier
type ResourceLimits struct {
	MaxMemoryBytes      int64 `yaml:"max_memory_bytes"`
	MaxCPUCores         int32 `yaml:"max_cpu_cores"`
	MaxDiskBytes        int64 `yaml:"max_disk_bytes"`
	TimeoutSeconds      int32 `yaml:"timeout_seconds"`
	MaxConcurrentBuilds int32 `yaml:"max_concurrent_builds"`
	MaxDailyBuilds      int32 `yaml:"max_daily_builds"`
	MaxStorageBytes     int64 `yaml:"max_storage_bytes"`
	MaxBuildTimeMinutes int32 `yaml:"max_build_time_minutes"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	DataDir string `yaml:"data_dir"`
	Type    string `yaml:"type"` // "sqlite" (recommended), "postgres"

	// PostgreSQL specific (optional)
	Host     string `yaml:"host,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	Database string `yaml:"database,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	SSLMode  string `yaml:"ssl_mode,omitempty"`
}

// OpenTelemetryConfig holds observability configuration
type OpenTelemetryConfig struct {
	Enabled                      bool    `yaml:"enabled"`
	ServiceName                  string  `yaml:"service_name"`
	ServiceVersion               string  `yaml:"service_version"`
	TracingSamplingRate          float64 `yaml:"tracing_sampling_rate"`
	OTLPEndpoint                 string  `yaml:"otlp_endpoint"`
	PrometheusEnabled            bool    `yaml:"prometheus_enabled"`
	PrometheusPort               string  `yaml:"prometheus_port"`
	PrometheusInterface          string  `yaml:"prometheus_interface"`
	HighCardinalityLabelsEnabled bool    `yaml:"high_cardinality_labels_enabled"`
}

// AssetManagerConfig holds assetmanagerd client configuration
type AssetManagerConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Endpoint string `yaml:"endpoint"`
}

// TLSConfig holds TLS configuration
// AIDEV-BUSINESS_RULE: SPIFFE/mTLS is required by default for security - no fallback to disabled mode
type TLSConfig struct {
	// Mode can be "disabled", "file", or "spiffe"
	Mode string `json:"mode,omitempty"`

	// File-based TLS options
	CertFile string `json:"cert_file,omitempty"`
	KeyFile  string `json:"-"` // AIDEV-NOTE: Never serialize private key paths
	CAFile   string `json:"ca_file,omitempty"`

	// SPIFFE options
	SPIFFESocketPath string `json:"spiffe_socket_path,omitempty"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	return LoadConfigWithLogger(slog.Default())
}

// LoadConfigWithLogger loads configuration with a custom logger
func LoadConfigWithLogger(logger *slog.Logger) (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Address:         getEnvOrDefault("UNKEY_BUILDERD_ADDRESS", "0.0.0.0"),
			Port:            getEnvOrDefault("UNKEY_BUILDERD_PORT", "8082"),
			ShutdownTimeout: getEnvDurationOrDefault("UNKEY_BUILDERD_SHUTDOWN_TIMEOUT", 15*time.Second),
			RateLimit:       getEnvIntOrDefault("UNKEY_BUILDERD_RATE_LIMIT", 100),
		},
		Builder: BuilderConfig{
			MaxConcurrentBuilds: getEnvIntOrDefault("UNKEY_BUILDERD_MAX_CONCURRENT_BUILDS", 5),
			BuildTimeout:        getEnvDurationOrDefault("UNKEY_BUILDERD_BUILD_TIMEOUT", 15*time.Minute),
			ScratchDir:          getEnvOrDefault("UNKEY_BUILDERD_SCRATCH_DIR", "/tmp/builderd"),
			RootfsOutputDir:     getEnvOrDefault("UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR", "/opt/builderd/rootfs"),
			WorkspaceDir:        getEnvOrDefault("UNKEY_BUILDERD_WORKSPACE_DIR", "/opt/builderd/workspace"),
			CleanupInterval:     getEnvDurationOrDefault("UNKEY_BUILDERD_CLEANUP_INTERVAL", 1*time.Hour),
			UsePipelineExecutor: getEnvBoolOrDefault("UNKEY_BUILDERD_USE_PIPELINE_EXECUTOR", false),
		},
		Storage: StorageConfig{ //nolint:exhaustruct // S3Config and GCSConfig are optional backend-specific configs
			Backend:        getEnvOrDefault("UNKEY_BUILDERD_STORAGE_BACKEND", "local"),
			RetentionDays:  getEnvIntOrDefault("UNKEY_BUILDERD_STORAGE_RETENTION_DAYS", 30),
			MaxSizeGB:      getEnvIntOrDefault("UNKEY_BUILDERD_STORAGE_MAX_SIZE_GB", 100),
			CacheEnabled:   getEnvBoolOrDefault("UNKEY_BUILDERD_STORAGE_CACHE_ENABLED", true),
			CacheMaxSizeGB: getEnvIntOrDefault("UNKEY_BUILDERD_STORAGE_CACHE_MAX_SIZE_GB", 50),
		},
		Docker: DockerConfig{
			RegistryAuth:       getEnvBoolOrDefault("UNKEY_BUILDERD_DOCKER_REGISTRY_AUTH", true),
			MaxImageSizeGB:     getEnvIntOrDefault("UNKEY_BUILDERD_DOCKER_MAX_IMAGE_SIZE_GB", 5),
			AllowedRegistries:  getEnvSliceOrDefault("UNKEY_BUILDERD_DOCKER_ALLOWED_REGISTRIES", []string{}),
			PullTimeout:        getEnvDurationOrDefault("UNKEY_BUILDERD_DOCKER_PULL_TIMEOUT", 10*time.Minute),
			RegistryMirror:     getEnvOrDefault("UNKEY_BUILDERD_DOCKER_REGISTRY_MIRROR", ""),
			InsecureRegistries: getEnvSliceOrDefault("UNKEY_BUILDERD_DOCKER_INSECURE_REGISTRIES", []string{}),
		},
		Tenant: TenantConfig{
			DefaultTier:        getEnvOrDefault("UNKEY_BUILDERD_TENANT_DEFAULT_TIER", "free"),
			IsolationEnabled:   getEnvBoolOrDefault("UNKEY_BUILDERD_TENANT_ISOLATION_ENABLED", true),
			QuotaCheckInterval: getEnvDurationOrDefault("UNKEY_BUILDERD_TENANT_QUOTA_CHECK_INTERVAL", 5*time.Minute),
			DefaultResourceLimits: ResourceLimits{
				MaxMemoryBytes:      getEnvInt64OrDefault("UNKEY_BUILDERD_TENANT_DEFAULT_MAX_MEMORY_BYTES", 2<<30), // 2GB
				MaxCPUCores:         getEnvInt32OrDefault("UNKEY_BUILDERD_TENANT_DEFAULT_MAX_CPU_CORES", 2),
				MaxDiskBytes:        getEnvInt64OrDefault("UNKEY_BUILDERD_TENANT_DEFAULT_MAX_DISK_BYTES", 10<<30), // 10GB
				TimeoutSeconds:      getEnvInt32OrDefault("UNKEY_BUILDERD_TENANT_DEFAULT_TIMEOUT_SECONDS", 900),   // 15min
				MaxConcurrentBuilds: getEnvInt32OrDefault("UNKEY_BUILDERD_TENANT_DEFAULT_MAX_CONCURRENT_BUILDS", 3),
				MaxDailyBuilds:      getEnvInt32OrDefault("UNKEY_BUILDERD_TENANT_DEFAULT_MAX_DAILY_BUILDS", 100),
				MaxStorageBytes:     getEnvInt64OrDefault("UNKEY_BUILDERD_TENANT_DEFAULT_MAX_STORAGE_BYTES", 50<<30), // 50GB
				MaxBuildTimeMinutes: getEnvInt32OrDefault("UNKEY_BUILDERD_TENANT_DEFAULT_MAX_BUILD_TIME_MINUTES", 30),
			},
		},
		Database: DatabaseConfig{
			DataDir:  getEnvOrDefault("UNKEY_BUILDERD_DATABASE_DATA_DIR", "/opt/builderd/data"),
			Type:     getEnvOrDefault("UNKEY_BUILDERD_DATABASE_TYPE", "sqlite"),
			Host:     getEnvOrDefault("UNKEY_BUILDERD_DATABASE_HOST", "localhost"),
			Port:     getEnvIntOrDefault("UNKEY_BUILDERD_DATABASE_PORT", 5432),
			Database: getEnvOrDefault("UNKEY_BUILDERD_DATABASE_NAME", "builderd"),
			Username: getEnvOrDefault("UNKEY_BUILDERD_DATABASE_USERNAME", "builderd"),
			Password: getEnvOrDefault("UNKEY_BUILDERD_DATABASE_PASSWORD", ""),
			SSLMode:  getEnvOrDefault("UNKEY_BUILDERD_DATABASE_SSL_MODE", "disable"),
		},
		OpenTelemetry: OpenTelemetryConfig{
			Enabled:                      getEnvBoolOrDefault("UNKEY_BUILDERD_OTEL_ENABLED", false),
			ServiceName:                  getEnvOrDefault("UNKEY_BUILDERD_OTEL_SERVICE_NAME", "builderd"),
			ServiceVersion:               getEnvOrDefault("UNKEY_BUILDERD_OTEL_SERVICE_VERSION", "0.1.0"),
			TracingSamplingRate:          getEnvFloat64OrDefault("UNKEY_BUILDERD_OTEL_SAMPLING_RATE", 1.0),
			OTLPEndpoint:                 getEnvOrDefault("UNKEY_BUILDERD_OTEL_ENDPOINT", "localhost:4318"),
			PrometheusEnabled:            getEnvBoolOrDefault("UNKEY_BUILDERD_OTEL_PROMETHEUS_ENABLED", true),
			PrometheusPort:               getEnvOrDefault("UNKEY_BUILDERD_OTEL_PROMETHEUS_PORT", "9466"),
			PrometheusInterface:          getEnvOrDefault("UNKEY_BUILDERD_OTEL_PROMETHEUS_INTERFACE", "127.0.0.1"),
			HighCardinalityLabelsEnabled: getEnvBoolOrDefault("UNKEY_BUILDERD_OTEL_HIGH_CARDINALITY_ENABLED", false),
		},
		AssetManager: AssetManagerConfig{
			Enabled:  getEnvBoolOrDefault("UNKEY_BUILDERD_ASSETMANAGER_ENABLED", true),
			Endpoint: getEnvOrDefault("UNKEY_BUILDERD_ASSETMANAGER_ENDPOINT", "https://localhost:8083"),
		},
		TLS: &TLSConfig{
			Mode:             getEnvOrDefault("UNKEY_BUILDERD_TLS_MODE", "spiffe"),
			CertFile:         getEnvOrDefault("UNKEY_BUILDERD_TLS_CERT_FILE", ""),
			KeyFile:          getEnvOrDefault("UNKEY_BUILDERD_TLS_KEY_FILE", ""),
			CAFile:           getEnvOrDefault("UNKEY_BUILDERD_TLS_CA_FILE", ""),
			SPIFFESocketPath: getEnvOrDefault("UNKEY_BUILDERD_SPIFFE_SOCKET", "/var/lib/spire/agent/agent.sock"),
		},
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	logger.Info("configuration loaded successfully",
		slog.String("server_address", config.Server.Address),
		slog.String("server_port", config.Server.Port),
		slog.String("storage_backend", config.Storage.Backend),
		slog.Bool("otel_enabled", config.OpenTelemetry.Enabled),
		slog.Bool("tenant_isolation", config.Tenant.IsolationEnabled),
		slog.Int("max_concurrent_builds", config.Builder.MaxConcurrentBuilds),
	)

	return config, nil
}

// validateConfig validates the loaded configuration
func validateConfig(config *Config) error {
	if config.Builder.MaxConcurrentBuilds <= 0 {
		return fmt.Errorf("max_concurrent_builds must be positive")
	}

	if config.Builder.BuildTimeout <= 0 {
		return fmt.Errorf("build_timeout must be positive")
	}

	if config.Storage.MaxSizeGB <= 0 {
		return fmt.Errorf("storage max_size_gb must be positive")
	}

	if config.Docker.MaxImageSizeGB <= 0 {
		return fmt.Errorf("docker max_image_size_gb must be positive")
	}

	if config.OpenTelemetry.TracingSamplingRate < 0 || config.OpenTelemetry.TracingSamplingRate > 1 {
		return fmt.Errorf("tracing_sampling_rate must be between 0.0 and 1.0")
	}

	return nil
}

// Helper functions for environment variable parsing
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt32OrDefault(key string, defaultValue int32) int32 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 32); err == nil {
			return int32(parsed)
		}
	}
	return defaultValue
}

func getEnvInt64OrDefault(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvFloat64OrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvSliceOrDefault(key string, defaultValue []string) []string {
	// For now, return default. In production, could parse comma-separated values
	return defaultValue
}
