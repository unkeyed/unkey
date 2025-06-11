package config

import (
	"os"
	"strings"
	"testing"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/backend/types"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    *Config
		wantErr bool
	}{
		{
			name:    "default configuration",
			envVars: map[string]string{},
			want: &Config{
				Server: ServerConfig{
					Port:    "8080",
					Address: "0.0.0.0",
				},
				Backend: BackendConfig{
					Type: types.BackendTypeCloudHypervisor,
					CloudHypervisor: CloudHypervisorConfig{
						Endpoint: "unix:///tmp/ch.sock",
					},
					Firecracker: FirecrackerConfig{
						Endpoint: "unix:///tmp/firecracker.sock",
					},
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:             false,
					ServiceName:         "metald",
					ServiceVersion:      "0.0.1",
					TracingSamplingRate: 1.0,
					OTLPEndpoint:        "localhost:4318",
					PrometheusEnabled:   true,
					PrometheusPort:      "9464",
				},
			},
			wantErr: false,
		},
		{
			name: "custom server configuration",
			envVars: map[string]string{
				"UNKEY_METALD_PORT":    "9999",
				"UNKEY_METALD_ADDRESS": "127.0.0.1",
			},
			want: &Config{
				Server: ServerConfig{
					Port:    "9999",
					Address: "127.0.0.1",
				},
				Backend: BackendConfig{
					Type: types.BackendTypeCloudHypervisor,
					CloudHypervisor: CloudHypervisorConfig{
						Endpoint: "unix:///tmp/ch.sock",
					},
					Firecracker: FirecrackerConfig{
						Endpoint: "unix:///tmp/firecracker.sock",
					},
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:             false,
					ServiceName:         "metald",
					ServiceVersion:      "0.0.1",
					TracingSamplingRate: 1.0,
					OTLPEndpoint:        "localhost:4318",
					PrometheusEnabled:   true,
					PrometheusPort:      "9464",
				},
			},
			wantErr: false,
		},
		{
			name: "custom backend endpoint",
			envVars: map[string]string{
				"UNKEY_METALD_CH_ENDPOINT": "unix:///var/run/ch.sock",
			},
			want: &Config{
				Server: ServerConfig{
					Port:    "8080",
					Address: "0.0.0.0",
				},
				Backend: BackendConfig{
					Type: types.BackendTypeCloudHypervisor,
					CloudHypervisor: CloudHypervisorConfig{
						Endpoint: "unix:///var/run/ch.sock",
					},
					Firecracker: FirecrackerConfig{
						Endpoint: "unix:///tmp/firecracker.sock",
					},
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:             false,
					ServiceName:         "metald",
					ServiceVersion:      "0.0.1",
					TracingSamplingRate: 1.0,
					OTLPEndpoint:        "localhost:4318",
					PrometheusEnabled:   true,
					PrometheusPort:      "9464",
				},
			},
			wantErr: false,
		},
		{
			name: "opentelemetry enabled with custom config",
			envVars: map[string]string{
				"UNKEY_METALD_OTEL_ENABLED":            "true",
				"UNKEY_METALD_OTEL_SERVICE_NAME":       "test-service",
				"UNKEY_METALD_OTEL_SERVICE_VERSION":    "2.0.0",
				"UNKEY_METALD_OTEL_SAMPLING_RATE":      "0.5",
				"UNKEY_METALD_OTEL_ENDPOINT":           "otel-collector:4318",
				"UNKEY_METALD_OTEL_PROMETHEUS_ENABLED": "false",
				"UNKEY_METALD_OTEL_PROMETHEUS_PORT":    "8888",
			},
			want: &Config{
				Server: ServerConfig{
					Port:    "8080",
					Address: "0.0.0.0",
				},
				Backend: BackendConfig{
					Type: types.BackendTypeCloudHypervisor,
					CloudHypervisor: CloudHypervisorConfig{
						Endpoint: "unix:///tmp/ch.sock",
					},
					Firecracker: FirecrackerConfig{
						Endpoint: "unix:///tmp/firecracker.sock",
					},
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:             true,
					ServiceName:         "test-service",
					ServiceVersion:      "2.0.0",
					TracingSamplingRate: 0.5,
					OTLPEndpoint:        "otel-collector:4318",
					PrometheusEnabled:   false,
					PrometheusPort:      "8888",
				},
			},
			wantErr: false,
		},
		{
			name: "firecracker backend configuration",
			envVars: map[string]string{
				"UNKEY_METALD_BACKEND": "firecracker",
			},
			want: &Config{
				Server: ServerConfig{
					Port:    "8080",
					Address: "0.0.0.0",
				},
				Backend: BackendConfig{
					Type: types.BackendTypeFirecracker,
					CloudHypervisor: CloudHypervisorConfig{
						Endpoint: "unix:///tmp/ch.sock",
					},
					Firecracker: FirecrackerConfig{
						Endpoint: "unix:///tmp/firecracker.sock",
					},
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:             false,
					ServiceName:         "metald",
					ServiceVersion:      "0.0.1",
					TracingSamplingRate: 1.0,
					OTLPEndpoint:        "localhost:4318",
					PrometheusEnabled:   true,
					PrometheusPort:      "9464",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment before test
			clearEnv()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer clearEnv() // Clean up after test

			got, err := LoadConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return // Don't check config if we expected an error
			}

			if !compareConfigs(got, tt.want) {
				t.Errorf("LoadConfig() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLoadConfigWithSocketPath(t *testing.T) {
	tests := []struct {
		name       string
		socketPath string
		envVars    map[string]string
		want       string // expected endpoint
		wantErr    bool
	}{
		{
			name:       "socket path override",
			socketPath: "/custom/path/ch.sock",
			envVars:    map[string]string{},
			want:       "unix:///custom/path/ch.sock",
			wantErr:    false,
		},
		{
			name:       "socket path with unix:// prefix",
			socketPath: "unix:///already/prefixed.sock",
			envVars:    map[string]string{},
			want:       "unix:///already/prefixed.sock",
			wantErr:    false,
		},
		{
			name:       "empty socket path uses env var",
			socketPath: "",
			envVars: map[string]string{
				"UNKEY_METALD_CH_ENDPOINT": "unix:///env/path.sock",
			},
			want:    "unix:///env/path.sock",
			wantErr: false,
		},
		{
			name:       "empty socket path uses default",
			socketPath: "",
			envVars:    map[string]string{},
			want:       "unix:///tmp/ch.sock",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment before test
			clearEnv()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer clearEnv()

			got, err := LoadConfigWithSocketPath(tt.socketPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfigWithSocketPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if got.Backend.CloudHypervisor.Endpoint != tt.want {
				t.Errorf("LoadConfigWithSocketPath() endpoint = %v, want %v", got.Backend.CloudHypervisor.Endpoint, tt.want)
			}
		})
	}
}

func TestOpenTelemetryConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid sampling rate 0.0",
			envVars: map[string]string{
				"UNKEY_METALD_OTEL_ENABLED":       "true",
				"UNKEY_METALD_OTEL_SAMPLING_RATE": "0.0",
			},
			wantErr: false,
		},
		{
			name: "valid sampling rate 1.0",
			envVars: map[string]string{
				"UNKEY_METALD_OTEL_ENABLED":       "true",
				"UNKEY_METALD_OTEL_SAMPLING_RATE": "1.0",
			},
			wantErr: false,
		},
		{
			name: "valid sampling rate 0.5",
			envVars: map[string]string{
				"UNKEY_METALD_OTEL_ENABLED":       "true",
				"UNKEY_METALD_OTEL_SAMPLING_RATE": "0.5",
			},
			wantErr: false,
		},
		{
			name: "invalid sampling rate -0.1",
			envVars: map[string]string{
				"UNKEY_METALD_OTEL_ENABLED":       "true",
				"UNKEY_METALD_OTEL_SAMPLING_RATE": "-0.1",
			},
			wantErr: true,
			errMsg:  "tracing sampling rate must be between 0.0 and 1.0",
		},
		{
			name: "invalid sampling rate 1.1",
			envVars: map[string]string{
				"UNKEY_METALD_OTEL_ENABLED":       "true",
				"UNKEY_METALD_OTEL_SAMPLING_RATE": "1.1",
			},
			wantErr: true,
			errMsg:  "tracing sampling rate must be between 0.0 and 1.0",
		},
		// Note: We can't easily test empty OTLP endpoint and service name because
		// getEnvOrDefault() will return the default value when env var is empty string.
		// These would need to be tested by temporarily modifying the validation logic
		// or using dependency injection for the defaults.
		{
			name: "OTEL disabled - validation should pass even with invalid values",
			envVars: map[string]string{
				"UNKEY_METALD_OTEL_ENABLED":       "false",
				"UNKEY_METALD_OTEL_SAMPLING_RATE": "5.0", // Invalid but should be ignored
				"UNKEY_METALD_OTEL_ENDPOINT":      "",    // Empty but should be ignored
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment before test
			clearEnv()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer clearEnv()

			_, err := LoadConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					// Check if error message contains the expected substring
					if len(tt.errMsg) > 0 && !strings.Contains(err.Error(), tt.errMsg) {
						t.Errorf("LoadConfig() error = %v, want error containing %v", err.Error(), tt.errMsg)
					}
				}
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid cloud hypervisor config",
			config: &Config{
				Backend: BackendConfig{
					Type: types.BackendTypeCloudHypervisor,
					CloudHypervisor: CloudHypervisorConfig{
						Endpoint: "unix:///tmp/ch.sock",
					},
					Firecracker: FirecrackerConfig{
						Endpoint: "unix:///tmp/firecracker.sock",
					},
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled: false,
				},
			},
			wantErr: false,
		},
		{
			name: "valid OTEL config",
			config: &Config{
				Backend: BackendConfig{
					Type: types.BackendTypeCloudHypervisor,
					CloudHypervisor: CloudHypervisorConfig{
						Endpoint: "unix:///tmp/ch.sock",
					},
					Firecracker: FirecrackerConfig{
						Endpoint: "unix:///tmp/firecracker.sock",
					},
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:             true,
					ServiceName:         "test-service",
					TracingSamplingRate: 0.5,
					OTLPEndpoint:        "localhost:4318",
				},
			},
			wantErr: false,
		},
		{
			name: "missing cloud hypervisor endpoint",
			config: &Config{
				Backend: BackendConfig{
					Type: types.BackendTypeCloudHypervisor,
					CloudHypervisor: CloudHypervisorConfig{
						Endpoint: "",
					},
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled: false,
				},
			},
			wantErr: true,
			errMsg:  "cloud hypervisor endpoint is required",
		},
		{
			name: "OTEL enabled with invalid sampling rate low",
			config: &Config{
				Backend: BackendConfig{
					Type: types.BackendTypeCloudHypervisor,
					CloudHypervisor: CloudHypervisorConfig{
						Endpoint: "unix:///tmp/ch.sock",
					},
					Firecracker: FirecrackerConfig{
						Endpoint: "unix:///tmp/firecracker.sock",
					},
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:             true,
					ServiceName:         "test-service",
					TracingSamplingRate: -0.1,
					OTLPEndpoint:        "localhost:4318",
				},
			},
			wantErr: true,
			errMsg:  "tracing sampling rate must be between 0.0 and 1.0",
		},
		{
			name: "OTEL enabled with invalid sampling rate high",
			config: &Config{
				Backend: BackendConfig{
					Type: types.BackendTypeCloudHypervisor,
					CloudHypervisor: CloudHypervisorConfig{
						Endpoint: "unix:///tmp/ch.sock",
					},
					Firecracker: FirecrackerConfig{
						Endpoint: "unix:///tmp/firecracker.sock",
					},
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:             true,
					ServiceName:         "test-service",
					TracingSamplingRate: 1.1,
					OTLPEndpoint:        "localhost:4318",
				},
			},
			wantErr: true,
			errMsg:  "tracing sampling rate must be between 0.0 and 1.0",
		},
		{
			name: "OTEL enabled with missing OTLP endpoint",
			config: &Config{
				Backend: BackendConfig{
					Type: types.BackendTypeCloudHypervisor,
					CloudHypervisor: CloudHypervisorConfig{
						Endpoint: "unix:///tmp/ch.sock",
					},
					Firecracker: FirecrackerConfig{
						Endpoint: "unix:///tmp/firecracker.sock",
					},
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:             true,
					ServiceName:         "test-service",
					TracingSamplingRate: 0.5,
					OTLPEndpoint:        "",
				},
			},
			wantErr: true,
			errMsg:  "OTLP endpoint is required when OpenTelemetry is enabled",
		},
		{
			name: "OTEL enabled with missing service name",
			config: &Config{
				Backend: BackendConfig{
					Type: types.BackendTypeCloudHypervisor,
					CloudHypervisor: CloudHypervisorConfig{
						Endpoint: "unix:///tmp/ch.sock",
					},
					Firecracker: FirecrackerConfig{
						Endpoint: "unix:///tmp/firecracker.sock",
					},
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:             true,
					ServiceName:         "",
					TracingSamplingRate: 0.5,
					OTLPEndpoint:        "localhost:4318",
				},
			},
			wantErr: true,
			errMsg:  "service name is required when OpenTelemetry is enabled",
		},
		{
			name: "missing firecracker endpoint",
			config: &Config{
				Backend: BackendConfig{
					Type: types.BackendTypeFirecracker,
					Firecracker: FirecrackerConfig{
						Endpoint: "",
					},
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled: false,
				},
			},
			wantErr: true,
			errMsg:  "firecracker endpoint is required",
		},
		{
			name: "unsupported backend type",
			config: &Config{
				Backend: BackendConfig{
					Type: types.BackendType("unknown"),
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled: false,
				},
			},
			wantErr: true,
			errMsg:  "unsupported backend type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Config.Validate() error = %v, want error containing %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestFormatSocketPath(t *testing.T) {
	tests := []struct {
		name       string
		socketPath string
		want       string
	}{
		{
			name:       "path without unix:// prefix",
			socketPath: "/tmp/ch.sock",
			want:       "unix:///tmp/ch.sock",
		},
		{
			name:       "path with unix:// prefix",
			socketPath: "unix:///tmp/ch.sock",
			want:       "unix:///tmp/ch.sock",
		},
		{
			name:       "relative path",
			socketPath: "./ch.sock",
			want:       "unix://./ch.sock",
		},
		{
			name:       "empty path",
			socketPath: "",
			want:       "unix://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSocketPath(tt.socketPath)
			if got != tt.want {
				t.Errorf("formatSocketPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions

func clearEnv() {
	envVars := []string{
		"UNKEY_METALD_PORT", "UNKEY_METALD_ADDRESS", "UNKEY_METALD_BACKEND", "UNKEY_METALD_CH_ENDPOINT", "UNKEY_METALD_FC_ENDPOINT",
		"UNKEY_METALD_OTEL_ENABLED", "UNKEY_METALD_OTEL_SERVICE_NAME", "UNKEY_METALD_OTEL_SERVICE_VERSION",
		"UNKEY_METALD_OTEL_SAMPLING_RATE", "UNKEY_METALD_OTEL_ENDPOINT",
		"UNKEY_METALD_OTEL_PROMETHEUS_ENABLED", "UNKEY_METALD_OTEL_PROMETHEUS_PORT",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

func compareConfigs(got, want *Config) bool {
	if got == nil || want == nil {
		return got == want
	}

	// Compare Server config
	if got.Server.Port != want.Server.Port ||
		got.Server.Address != want.Server.Address {
		return false
	}

	// Compare Backend config
	if got.Backend.Type != want.Backend.Type ||
		got.Backend.CloudHypervisor.Endpoint != want.Backend.CloudHypervisor.Endpoint ||
		got.Backend.Firecracker.Endpoint != want.Backend.Firecracker.Endpoint {
		return false
	}

	// Compare OpenTelemetry config
	otelGot := got.OpenTelemetry
	otelWant := want.OpenTelemetry
	if otelGot.Enabled != otelWant.Enabled ||
		otelGot.ServiceName != otelWant.ServiceName ||
		otelGot.ServiceVersion != otelWant.ServiceVersion ||
		otelGot.TracingSamplingRate != otelWant.TracingSamplingRate ||
		otelGot.OTLPEndpoint != otelWant.OTLPEndpoint ||
		otelGot.PrometheusEnabled != otelWant.PrometheusEnabled ||
		otelGot.PrometheusPort != otelWant.PrometheusPort {
		return false
	}

	return true
}
