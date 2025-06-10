package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"vmm-controlplane/internal/backend/types"
)

// Config holds the application configuration
type Config struct {
	// Server configuration
	Server ServerConfig

	// Backend configuration
	Backend BackendConfig

	// OpenTelemetry configuration
	OpenTelemetry OpenTelemetryConfig
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
	// Type of backend (cloudhypervisor or firecracker)
	Type types.BackendType

	// CloudHypervisor specific config
	CloudHypervisor CloudHypervisorConfig

	// Firecracker specific config
	Firecracker FirecrackerConfig
}

// CloudHypervisorConfig holds Cloud Hypervisor specific configuration
type CloudHypervisorConfig struct {
	// API endpoint (unix:///path/to/socket)
	Endpoint string
}

// FirecrackerConfig holds Firecracker specific configuration
type FirecrackerConfig struct {
	// API endpoint (unix:///path/to/socket)
	Endpoint string
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
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	return LoadConfigWithSocketPath("")
}

// LoadConfigWithSocketPath loads configuration with an optional socket path override
func LoadConfigWithSocketPath(socketPath string) (*Config, error) {
	// Determine endpoints based on socket path or environment
	chEndpoint := getEnvOrDefault("UNKEY_VMCP_CH_ENDPOINT", "unix:///tmp/ch.sock")
	fcEndpoint := getEnvOrDefault("UNKEY_VMCP_FC_ENDPOINT", "unix:///tmp/firecracker.sock")
	if socketPath != "" {
		// Only override Cloud Hypervisor endpoint for backward compatibility
		chEndpoint = formatSocketPath(socketPath)
	}

	// Parse sampling rate
	samplingRate := 1.0
	if samplingStr := os.Getenv("UNKEY_VMCP_OTEL_SAMPLING_RATE"); samplingStr != "" {
		if parsed, err := strconv.ParseFloat(samplingStr, 64); err == nil {
			samplingRate = parsed
		}
	}

	// Parse enabled flag
	otelEnabled := false
	if enabledStr := os.Getenv("UNKEY_VMCP_OTEL_ENABLED"); enabledStr != "" {
		if parsed, err := strconv.ParseBool(enabledStr); err == nil {
			otelEnabled = parsed
		}
	}

	// Parse Prometheus enabled flag
	prometheusEnabled := true // Default to true when OTEL is enabled
	if promStr := os.Getenv("UNKEY_VMCP_OTEL_PROMETHEUS_ENABLED"); promStr != "" {
		if parsed, err := strconv.ParseBool(promStr); err == nil {
			prometheusEnabled = parsed
		}
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:    getEnvOrDefault("UNKEY_VMCP_PORT", "8080"),
			Address: getEnvOrDefault("UNKEY_VMCP_ADDRESS", "0.0.0.0"),
		},
		Backend: BackendConfig{
			Type: types.BackendType(getEnvOrDefault("UNKEY_VMCP_BACKEND", string(types.BackendTypeCloudHypervisor))),
			CloudHypervisor: CloudHypervisorConfig{
				Endpoint: chEndpoint,
			},
			Firecracker: FirecrackerConfig{
				Endpoint: fcEndpoint,
			},
		},
		OpenTelemetry: OpenTelemetryConfig{
			Enabled:             otelEnabled,
			ServiceName:         getEnvOrDefault("UNKEY_VMCP_OTEL_SERVICE_NAME", "vmm-controlplane"),
			ServiceVersion:      getEnvOrDefault("UNKEY_VMCP_OTEL_SERVICE_VERSION", "0.0.1"),
			TracingSamplingRate: samplingRate,
			OTLPEndpoint:        getEnvOrDefault("UNKEY_VMCP_OTEL_ENDPOINT", "localhost:4318"),
			PrometheusEnabled:   prometheusEnabled,
			PrometheusPort:      getEnvOrDefault("UNKEY_VMCP_OTEL_PROMETHEUS_PORT", "9464"),
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
	// AIDEV-BUSINESS_RULE: Backend type must be supported
	switch c.Backend.Type {
	case types.BackendTypeCloudHypervisor:
		if c.Backend.CloudHypervisor.Endpoint == "" {
			return fmt.Errorf("cloud hypervisor endpoint is required")
		}
	case types.BackendTypeFirecracker:
		if c.Backend.Firecracker.Endpoint == "" {
			return fmt.Errorf("firecracker endpoint is required")
		}
	default:
		return fmt.Errorf("unsupported backend type: %s", c.Backend.Type)
	}

	// AIDEV-NOTE: Comprehensive unit tests implemented in config_test.go
	// Tests cover: parsing, validation, edge cases, default values, and error conditions
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

// formatSocketPath formats the socket path into a unix:// URL
func formatSocketPath(socketPath string) string {
	// AIDEV-NOTE: Cloud Hypervisor only supports Unix sockets
	if strings.HasPrefix(socketPath, "unix://") {
		return socketPath
	}

	// Add unix:// scheme to socket paths
	return "unix://" + socketPath
}
