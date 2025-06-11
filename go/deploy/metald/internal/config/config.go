package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/backend/types"
)

// Config holds the application configuration
type Config struct {
	// Server configuration
	Server ServerConfig

	// Backend configuration
	Backend BackendConfig

	// Process management configuration
	ProcessManager ProcessManagerConfig

	// Billing configuration
	Billing BillingConfig

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

	// Jailer configuration for production deployment
	Jailer JailerConfig
}

// JailerConfig holds Firecracker jailer configuration
type JailerConfig struct {
	// Enabled indicates if jailer should be used (required for production)
	Enabled bool

	// Path to jailer binary
	BinaryPath string

	// Path to Firecracker binary (must be statically linked)
	FirecrackerBinaryPath string

	// UID for jailer process isolation
	UID uint32

	// GID for jailer process isolation
	GID uint32

	// Chroot directory for jailer isolation
	ChrootBaseDir string

	// Enable network namespace isolation
	NetNS bool

	// Enable PID namespace isolation
	PIDNS bool

	// Resource limits
	ResourceLimits JailerResourceLimits
}

// JailerResourceLimits holds resource limits for jailer
type JailerResourceLimits struct {
	// Memory limit in bytes
	MemoryLimitBytes int64

	// CPU quota (percentage, e.g., 100 = 1 CPU core)
	CPUQuota int64

	// Number of file descriptors
	FileDescriptorLimit int64
}

// ProcessManagerConfig holds process manager configuration
type ProcessManagerConfig struct {
	// SocketDir is the directory for Unix socket files (secure location)
	SocketDir string

	// LogDir is the directory for process log files (secure location)
	LogDir string

	// MaxProcesses is the maximum number of concurrent Firecracker processes
	MaxProcesses int
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
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	return LoadConfigWithSocketPath("")
}

// LoadConfigWithSocketPath loads configuration with an optional socket path override
func LoadConfigWithSocketPath(socketPath string) (*Config, error) {
	// Determine endpoints based on socket path or environment
	chEndpoint := getEnvOrDefault("UNKEY_METALD_CH_ENDPOINT", "unix:///tmp/ch.sock")
	fcEndpoint := getEnvOrDefault("UNKEY_METALD_FC_ENDPOINT", "unix:///tmp/firecracker.sock")
	if socketPath != "" {
		// Only override Cloud Hypervisor endpoint for backward compatibility
		chEndpoint = formatSocketPath(socketPath)
	}

	// Parse sampling rate
	samplingRate := 1.0
	if samplingStr := os.Getenv("UNKEY_METALD_OTEL_SAMPLING_RATE"); samplingStr != "" {
		if parsed, err := strconv.ParseFloat(samplingStr, 64); err == nil {
			samplingRate = parsed
		}
	}

	// Parse enabled flag
	otelEnabled := false
	if enabledStr := os.Getenv("UNKEY_METALD_OTEL_ENABLED"); enabledStr != "" {
		if parsed, err := strconv.ParseBool(enabledStr); err == nil {
			otelEnabled = parsed
		}
	}

	// Parse Prometheus enabled flag
	prometheusEnabled := true // Default to true when OTEL is enabled
	if promStr := os.Getenv("UNKEY_METALD_OTEL_PROMETHEUS_ENABLED"); promStr != "" {
		if parsed, err := strconv.ParseBool(promStr); err == nil {
			prometheusEnabled = parsed
		}
	}

	// Parse jailer enabled flag
	jailerEnabled := false
	if jailerStr := os.Getenv("UNKEY_METALD_JAILER_ENABLED"); jailerStr != "" {
		if parsed, err := strconv.ParseBool(jailerStr); err == nil {
			jailerEnabled = parsed
		}
	}

	// Parse jailer UID/GID
	jailerUID := uint32(1000)
	if uidStr := os.Getenv("UNKEY_METALD_JAILER_UID"); uidStr != "" {
		if parsed, err := strconv.ParseUint(uidStr, 10, 32); err == nil {
			jailerUID = uint32(parsed)
		}
	}

	jailerGID := uint32(1000)
	if gidStr := os.Getenv("UNKEY_METALD_JAILER_GID"); gidStr != "" {
		if parsed, err := strconv.ParseUint(gidStr, 10, 32); err == nil {
			jailerGID = uint32(parsed)
		}
	}

	// Parse jailer namespace flags
	jailerNetNS := true
	if netnsStr := os.Getenv("UNKEY_METALD_JAILER_NETNS"); netnsStr != "" {
		if parsed, err := strconv.ParseBool(netnsStr); err == nil {
			jailerNetNS = parsed
		}
	}

	jailerPIDNS := true
	if pidnsStr := os.Getenv("UNKEY_METALD_JAILER_PIDNS"); pidnsStr != "" {
		if parsed, err := strconv.ParseBool(pidnsStr); err == nil {
			jailerPIDNS = parsed
		}
	}

	// Parse resource limits
	memLimit := int64(128 * 1024 * 1024) // 128MB default
	if memStr := os.Getenv("UNKEY_METALD_JAILER_MEMORY_LIMIT"); memStr != "" {
		if parsed, err := strconv.ParseInt(memStr, 10, 64); err == nil {
			memLimit = parsed
		}
	}

	cpuQuota := int64(100) // 1 CPU core default
	if cpuStr := os.Getenv("UNKEY_METALD_JAILER_CPU_QUOTA"); cpuStr != "" {
		if parsed, err := strconv.ParseInt(cpuStr, 10, 64); err == nil {
			cpuQuota = parsed
		}
	}

	fdLimit := int64(1024) // 1024 file descriptors default
	if fdStr := os.Getenv("UNKEY_METALD_JAILER_FD_LIMIT"); fdStr != "" {
		if parsed, err := strconv.ParseInt(fdStr, 10, 64); err == nil {
			fdLimit = parsed
		}
	}

	// Parse process manager configuration with secure defaults
	maxProcesses := 25 // Default based on 32-core host
	if maxProcStr := os.Getenv("UNKEY_METALD_MAX_PROCESSES"); maxProcStr != "" {
		if parsed, err := strconv.Atoi(maxProcStr); err == nil && parsed > 0 {
			maxProcesses = parsed
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

	cfg := &Config{
		Server: ServerConfig{
			Port:    getEnvOrDefault("UNKEY_METALD_PORT", "8080"),
			Address: getEnvOrDefault("UNKEY_METALD_ADDRESS", "0.0.0.0"),
		},
		Backend: BackendConfig{
			Type: types.BackendType(getEnvOrDefault("UNKEY_METALD_BACKEND", string(types.BackendTypeCloudHypervisor))),
			CloudHypervisor: CloudHypervisorConfig{
				Endpoint: chEndpoint,
			},
			Firecracker: FirecrackerConfig{
				Endpoint: fcEndpoint,
				Jailer: JailerConfig{
					Enabled:               jailerEnabled,
					BinaryPath:            getEnvOrDefault("UNKEY_METALD_JAILER_BINARY", "/usr/bin/jailer"),
					FirecrackerBinaryPath: getEnvOrDefault("UNKEY_METALD_FIRECRACKER_BINARY", "/usr/bin/firecracker"),
					UID:                   jailerUID,
					GID:                   jailerGID,
					ChrootBaseDir:         getEnvOrDefault("UNKEY_METALD_JAILER_CHROOT_DIR", "/srv/jailer"),
					NetNS:                 jailerNetNS,
					PIDNS:                 jailerPIDNS,
					ResourceLimits: JailerResourceLimits{
						MemoryLimitBytes:    memLimit,
						CPUQuota:            cpuQuota,
						FileDescriptorLimit: fdLimit,
					},
				},
			},
		},
		ProcessManager: ProcessManagerConfig{
			SocketDir:    getEnvOrDefault("UNKEY_METALD_SOCKET_DIR", "/opt/metald/sockets"),
			LogDir:       getEnvOrDefault("UNKEY_METALD_LOG_DIR", "/opt/metald/logs"),
			MaxProcesses: maxProcesses,
		},
		Billing: BillingConfig{
			Enabled:  billingEnabled,
			Endpoint: getEnvOrDefault("UNKEY_METALD_BILLING_ENDPOINT", "http://localhost:8081"),
			MockMode: billingMockMode,
		},
		OpenTelemetry: OpenTelemetryConfig{
			Enabled:             otelEnabled,
			ServiceName:         getEnvOrDefault("UNKEY_METALD_OTEL_SERVICE_NAME", "metald"),
			ServiceVersion:      getEnvOrDefault("UNKEY_METALD_OTEL_SERVICE_VERSION", "0.0.1"),
			TracingSamplingRate: samplingRate,
			OTLPEndpoint:        getEnvOrDefault("UNKEY_METALD_OTEL_ENDPOINT", "localhost:4318"),
			PrometheusEnabled:   prometheusEnabled,
			PrometheusPort:      getEnvOrDefault("UNKEY_METALD_OTEL_PROMETHEUS_PORT", "9464"),
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
