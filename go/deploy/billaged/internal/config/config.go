package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	// Server configuration
	Server ServerConfig

	// OpenTelemetry configuration
	OpenTelemetry OpenTelemetryConfig

	// Aggregation configuration
	Aggregation AggregationConfig

	// TLS configuration (optional, defaults to disabled)
	TLS *TLSConfig
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	// Port to listen on
	Port string

	// Address to bind to
	Address string
}

// AggregationConfig holds aggregation-specific configuration
type AggregationConfig struct {
	// Interval for usage summary aggregation
	Interval string
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

	// HighCardinalityLabelsEnabled allows high-cardinality labels like vm_id and customer_id
	// Set to false in production to reduce cardinality
	HighCardinalityLabelsEnabled bool

	// PrometheusInterface controls the binding interface for metrics endpoint
	// Default "127.0.0.1" for localhost only (secure)
	// Set to "0.0.0.0" if remote access needed (not recommended)
	PrometheusInterface string
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

// LoadConfigWithLogger loads configuration from environment variables with custom logger
func LoadConfigWithLogger(logger *slog.Logger) (*Config, error) {
	// Parse sampling rate
	samplingRate := 1.0
	if samplingStr := os.Getenv("UNKEY_BILLAGED_OTEL_SAMPLING_RATE"); samplingStr != "" {
		if parsed, err := strconv.ParseFloat(samplingStr, 64); err == nil {
			samplingRate = parsed
		} else {
			logger.Warn("invalid UNKEY_BILLAGED_OTEL_SAMPLING_RATE, using default 1.0",
				slog.String("value", samplingStr),
				slog.String("error", err.Error()),
			)
		}
	}

	// Parse enabled flag
	otelEnabled := false
	if enabledStr := os.Getenv("UNKEY_BILLAGED_OTEL_ENABLED"); enabledStr != "" {
		if parsed, err := strconv.ParseBool(enabledStr); err == nil {
			otelEnabled = parsed
		} else {
			logger.Warn("invalid UNKEY_BILLAGED_OTEL_ENABLED, using default false",
				slog.String("value", enabledStr),
				slog.String("error", err.Error()),
			)
		}
	}

	// Parse Prometheus enabled flag
	prometheusEnabled := true // Default to true when OTEL is enabled
	if promStr := os.Getenv("UNKEY_BILLAGED_OTEL_PROMETHEUS_ENABLED"); promStr != "" {
		if parsed, err := strconv.ParseBool(promStr); err == nil {
			prometheusEnabled = parsed
		} else {
			logger.Warn("invalid UNKEY_BILLAGED_OTEL_PROMETHEUS_ENABLED, using default true",
				slog.String("value", promStr),
				slog.String("error", err.Error()),
			)
		}
	}

	// Parse high cardinality labels flag
	highCardinalityLabelsEnabled := false // Default to false for production safety
	if highCardStr := os.Getenv("UNKEY_BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED"); highCardStr != "" {
		if parsed, err := strconv.ParseBool(highCardStr); err == nil {
			highCardinalityLabelsEnabled = parsed
		} else {
			logger.Warn("invalid UNKEY_BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED, using default false",
				slog.String("value", highCardStr),
				slog.String("error", err.Error()),
			)
		}
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:    getEnvOrDefault("UNKEY_BILLAGED_PORT", "8081"),
			Address: getEnvOrDefault("UNKEY_BILLAGED_ADDRESS", "0.0.0.0"),
		},
		Aggregation: AggregationConfig{
			Interval: getEnvOrDefault("UNKEY_BILLAGED_AGGREGATION_INTERVAL", "60s"),
		},
		OpenTelemetry: OpenTelemetryConfig{
			Enabled:                      otelEnabled,
			ServiceName:                  getEnvOrDefault("UNKEY_BILLAGED_OTEL_SERVICE_NAME", "billaged"),
			ServiceVersion:               getEnvOrDefault("UNKEY_BILLAGED_OTEL_SERVICE_VERSION", "0.1.0"),
			TracingSamplingRate:          samplingRate,
			OTLPEndpoint:                 getEnvOrDefault("UNKEY_BILLAGED_OTEL_ENDPOINT", "localhost:4318"),
			PrometheusEnabled:            prometheusEnabled,
			PrometheusPort:               getEnvOrDefault("UNKEY_BILLAGED_OTEL_PROMETHEUS_PORT", "9465"),
			PrometheusInterface:          getEnvOrDefault("UNKEY_BILLAGED_OTEL_PROMETHEUS_INTERFACE", "127.0.0.1"),
			HighCardinalityLabelsEnabled: highCardinalityLabelsEnabled,
		},
		TLS: &TLSConfig{
			Mode:             getEnvOrDefault("UNKEY_BILLAGED_TLS_MODE", "spiffe"),
			CertFile:         getEnvOrDefault("UNKEY_BILLAGED_TLS_CERT_FILE", ""),
			KeyFile:          getEnvOrDefault("UNKEY_BILLAGED_TLS_KEY_FILE", ""),
			CAFile:           getEnvOrDefault("UNKEY_BILLAGED_TLS_CA_FILE", ""),
			SPIFFESocketPath: getEnvOrDefault("UNKEY_BILLAGED_SPIFFE_SOCKET", "/var/lib/spire/agent/agent.sock"),
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
