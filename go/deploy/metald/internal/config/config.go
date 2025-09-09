package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/backend/types"
)

// Config holds the application configuration
type Config struct {
	// Server configuration
	Server ServerConfig

	// Backend configuration
	Backend BackendConfig

	// Billing configuration
	Billing BillingConfig

	// OpenTelemetry configuration
	OpenTelemetry OpenTelemetryConfig

	// Database configuration
	Database DatabaseConfig

	// AssetManager configuration
	AssetManager AssetManagerConfig

	// TLS configuration
	TLS *TLSConfig
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	// Port to listen on
	Port string

	// Address to bind to
	Address string
}

// BackendConfig holds backend-specific configuration
type BackendConfig struct {
	// Type of backend
	Type types.BackendType

	// Jailer configuration
	Jailer JailerConfig
}

// JailerConfig holds Firecracker jailer configuration
type JailerConfig struct {
	// UID for jailer process isolation
	UID uint32

	// GID for jailer process isolation
	GID uint32

	// Chroot directory for jailer isolation
	ChrootBaseDir string
}

// BillingConfig holds billing service configuration
type BillingConfig struct {
	// Enabled indicates if billing integration is enabled
	Enabled bool

	// Endpoint is the billaged service endpoint (e.g., http://localhost:8081)
	Endpoint string

	// MockMode uses mock client instead of real ConnectRPC client
	MockMode bool
}

// OpenTelemetryConfig holds OpenTelemetry configuration
type OpenTelemetryConfig struct {
	// Enabled indicates if OpenTelemetry is enabled
	Enabled bool

	// ServiceName for resource attributes
	ServiceName string

	// ServiceVersion for resource attributes
	ServiceVersion string

	// TracingSamplingRate from 0.0 to 1.0
	TracingSamplingRate float64

	// OTLPEndpoint for sending traces and metrics
	OTLPEndpoint string

	// PrometheusEnabled enables Prometheus metrics endpoint
	PrometheusEnabled bool

	// PrometheusPort for scraping metrics
	PrometheusPort string

	// PrometheusInterface controls the binding interface for metrics endpoint
	// Default "127.0.0.1" for localhost only (secure)
	// Set to "0.0.0.0" if remote access needed (not recommended)
	PrometheusInterface string

	// HighCardinalityLabelsEnabled allows high-cardinality labels like vm_id and process_id
	// Set to false in production to reduce cardinality
	HighCardinalityLabelsEnabled bool
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	// DataDir is the directory where the SQLite database file is stored
	DataDir string
}

// AssetManagerConfig holds assetmanagerd service configuration
type AssetManagerConfig struct {
	// Enabled indicates if assetmanagerd integration is enabled
	Enabled bool

	// Endpoint is the assetmanagerd service endpoint (e.g., http://localhost:8082)
	Endpoint string

	// CacheDir is the local directory for caching assets
	CacheDir string
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	// Mode can be "file" or "spiffe" (default: "spiffe")
	Mode string `json:"mode,omitempty"`

	// File-based TLS options
	CertFile string `json:"cert_file,omitempty"`
	KeyFile  string `json:"-"` // AIDEV-NOTE: Never serialize private key paths
	CAFile   string `json:"ca_file,omitempty"`

	// SPIFFE options
	SPIFFESocketPath string `json:"spiffe_socket_path,omitempty"`

	// Performance options
	EnableCertCaching bool   `json:"enable_cert_caching,omitempty"`
	CertCacheTTL      string `json:"cert_cache_ttl,omitempty"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	return LoadConfigWithSocketPath("")
}

// LoadConfigWithSocketPath loads configuration with an optional socket path override
func LoadConfigWithSocketPath(socketPath string) (*Config, error) {
	// Use default logger for backward compatibility
	return LoadConfigWithSocketPathAndLogger(socketPath, slog.Default())
}

// LoadConfigWithSocketPathAndLogger loads configuration with optional socket path override and custom logger
func LoadConfigWithSocketPathAndLogger(socketPath string, logger *slog.Logger) (*Config, error) {
	// Parse sampling rate
	samplingRate := 1.0
	if samplingStr := os.Getenv("UNKEY_METALD_OTEL_SAMPLING_RATE"); samplingStr != "" {
		if parsed, err := strconv.ParseFloat(samplingStr, 64); err == nil {
			samplingRate = parsed
		} else {
			logger.Warn("invalid UNKEY_METALD_OTEL_SAMPLING_RATE, using default 1.0",
				slog.String("value", samplingStr),
				slog.String("error", err.Error()),
			)
		}
	}

	// Parse enabled flag
	otelEnabled := false
	if enabledStr := os.Getenv("UNKEY_METALD_OTEL_ENABLED"); enabledStr != "" {
		if parsed, err := strconv.ParseBool(enabledStr); err == nil {
			otelEnabled = parsed
		} else {
			logger.Warn("invalid UNKEY_METALD_OTEL_ENABLED, using default false",
				slog.String("value", enabledStr),
				slog.String("error", err.Error()),
			)
		}
	}

	// Parse Prometheus enabled flag
	prometheusEnabled := true // Default to true when OTEL is enabled
	if promStr := os.Getenv("UNKEY_METALD_OTEL_PROMETHEUS_ENABLED"); promStr != "" {
		if parsed, err := strconv.ParseBool(promStr); err == nil {
			prometheusEnabled = parsed
		} else {
			logger.Warn("invalid UNKEY_METALD_OTEL_PROMETHEUS_ENABLED, using default true",
				slog.String("value", promStr),
				slog.String("error", err.Error()),
			)
		}
	}

	// Parse high cardinality labels flag
	highCardinalityLabelsEnabled := false // Default to false for production safety
	if highCardStr := os.Getenv("UNKEY_METALD_OTEL_HIGH_CARDINALITY_ENABLED"); highCardStr != "" {
		if parsed, err := strconv.ParseBool(highCardStr); err == nil {
			highCardinalityLabelsEnabled = parsed
		} else {
			logger.Warn("invalid UNKEY_METALD_OTEL_HIGH_CARDINALITY_ENABLED, using default false",
				slog.String("value", highCardStr),
				slog.String("error", err.Error()),
			)
		}
	}

	// Parse jailer UID/GID
	jailerUID := uint32(1000)
	if uidStr := os.Getenv("UNKEY_METALD_JAILER_UID"); uidStr != "" {
		if parsed, err := strconv.ParseUint(uidStr, 10, 32); err == nil {
			jailerUID = uint32(parsed)
		} else {
			logger.Warn("invalid UNKEY_METALD_JAILER_UID, using default 1000",
				slog.String("value", uidStr),
				slog.String("error", err.Error()),
			)
		}
	}

	jailerGID := uint32(1000)
	if gidStr := os.Getenv("UNKEY_METALD_JAILER_GID"); gidStr != "" {
		if parsed, err := strconv.ParseUint(gidStr, 10, 32); err == nil {
			jailerGID = uint32(parsed)
		} else {
			logger.Warn("invalid UNKEY_METALD_JAILER_GID, using default 1000",
				slog.String("value", gidStr),
				slog.String("error", err.Error()),
			)
		}
	}

	// Parse billing configuration
	billingEnabled := true // Default to enabled
	if enabledStr := os.Getenv("UNKEY_METALD_BILLING_ENABLED"); enabledStr != "" {
		if parsed, err := strconv.ParseBool(enabledStr); err == nil {
			billingEnabled = parsed
		}
	}

	billingMockMode := false // Default to real client
	if mockStr := os.Getenv("UNKEY_METALD_BILLING_MOCK_MODE"); mockStr != "" {
		if parsed, err := strconv.ParseBool(mockStr); err == nil {
			billingMockMode = parsed
		}
	}

	// Parse assetmanager configuration
	assetManagerEnabled := true // Default to enabled
	if enabledStr := os.Getenv("UNKEY_METALD_ASSETMANAGER_ENABLED"); enabledStr != "" {
		if parsed, err := strconv.ParseBool(enabledStr); err == nil {
			assetManagerEnabled = parsed
		} else {
			logger.Warn("invalid UNKEY_METALD_ASSETMANAGER_ENABLED, using default true",
				slog.String("value", enabledStr),
				slog.String("error", err.Error()),
			)
		}
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:    getEnvOrDefault("UNKEY_METALD_PORT", "8080"),
			Address: getEnvOrDefault("UNKEY_METALD_ADDRESS", "0.0.0.0"),
		},
		Backend: BackendConfig{
			Type: types.BackendType(getEnvOrDefault("UNKEY_METALD_BACKEND", string(types.BackendTypeFirecracker))),
			Jailer: JailerConfig{
				UID:           jailerUID,
				GID:           jailerGID,
				ChrootBaseDir: getEnvOrDefault("UNKEY_METALD_JAILER_CHROOT_DIR", "/srv/jailer"),
			},
		},
		Billing: BillingConfig{
			Enabled:  billingEnabled,
			Endpoint: getEnvOrDefault("UNKEY_METALD_BILLING_ENDPOINT", "http://localhost:8081"),
			MockMode: billingMockMode,
		},
		OpenTelemetry: OpenTelemetryConfig{
			Enabled:                      otelEnabled,
			ServiceName:                  getEnvOrDefault("UNKEY_METALD_OTEL_SERVICE_NAME", "metald"),
			ServiceVersion:               getEnvOrDefault("UNKEY_METALD_OTEL_SERVICE_VERSION", "0.1.0"),
			TracingSamplingRate:          samplingRate,
			OTLPEndpoint:                 getEnvOrDefault("UNKEY_METALD_OTEL_ENDPOINT", "localhost:4318"),
			PrometheusEnabled:            prometheusEnabled,
			PrometheusPort:               getEnvOrDefault("UNKEY_METALD_OTEL_PROMETHEUS_PORT", "9464"),
			PrometheusInterface:          getEnvOrDefault("UNKEY_METALD_OTEL_PROMETHEUS_INTERFACE", "127.0.0.1"),
			HighCardinalityLabelsEnabled: highCardinalityLabelsEnabled,
		},
		Database: DatabaseConfig{
			DataDir: getEnvOrDefault("UNKEY_METALD_DATA_DIR", "/opt/metald/data"),
		},
		AssetManager: AssetManagerConfig{
			Enabled:  assetManagerEnabled,
			Endpoint: getEnvOrDefault("UNKEY_METALD_ASSETMANAGER_ENDPOINT", "http://localhost:8083"),
			CacheDir: getEnvOrDefault("UNKEY_METALD_ASSETMANAGER_CACHE_DIR", "/opt/metald/assets"),
		},
		TLS: &TLSConfig{
			// AIDEV-BUSINESS_RULE: mTLS/SPIFFE is required for production security
			Mode:              getEnvOrDefault("UNKEY_METALD_TLS_MODE", "spiffe"),
			CertFile:          getEnvOrDefault("UNKEY_METALD_TLS_CERT_FILE", ""),
			KeyFile:           getEnvOrDefault("UNKEY_METALD_TLS_KEY_FILE", ""),
			CAFile:            getEnvOrDefault("UNKEY_METALD_TLS_CA_FILE", ""),
			SPIFFESocketPath:  getEnvOrDefault("UNKEY_METALD_SPIFFE_SOCKET", "/var/lib/spire/agent/agent.sock"),
			EnableCertCaching: getEnvBoolOrDefault("UNKEY_METALD_TLS_ENABLE_CERT_CACHING"),
			CertCacheTTL:      getEnvOrDefault("UNKEY_METALD_TLS_CERT_CACHE_TTL", "5s"),
		},
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// AIDEV-BUSINESS_RULE: Support Firecracker and Docker backends
	if c.Backend.Type != types.BackendTypeFirecracker && c.Backend.Type != types.BackendTypeDocker {
		return fmt.Errorf("only firecracker and docker backends are supported, got: %s", c.Backend.Type)
	}

	if c.OpenTelemetry.Enabled {
		if c.OpenTelemetry.TracingSamplingRate < 0.0 || c.OpenTelemetry.TracingSamplingRate > 1.0 {
			return fmt.Errorf("tracing sampling rate must be between 0.0 and 1.0, got %f", c.OpenTelemetry.TracingSamplingRate)
		}
		if c.OpenTelemetry.OTLPEndpoint == "" {
			return fmt.Errorf("OTLP endpoint is required when OpenTelemetry is enabled")
		}
		if c.OpenTelemetry.ServiceName == "" {
			return fmt.Errorf("service name is required when OpenTelemetry is enabled")
		}
	}

	return nil
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBoolOrDefault(key string) bool {
	if value := os.Getenv(key); value != "" {
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return true
		}
		return boolValue
	}
	return true
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return defaultValue
		}
		return intValue
	}
	return defaultValue
}
