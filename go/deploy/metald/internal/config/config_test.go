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
					Type: types.BackendTypeFirecracker,
					Jailer: JailerConfig{
						UID:           1000,
						GID:           1000,
						ChrootBaseDir: "/srv/jailer",
					},
				},
				Billing: BillingConfig{
					Enabled:  true,
					Endpoint: "http://localhost:8081",
					MockMode: false,
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:                      false,
					ServiceName:                  "metald",
					ServiceVersion:               "0.1.0",
					TracingSamplingRate:          1.0,
					OTLPEndpoint:                 "localhost:4318",
					PrometheusEnabled:            true,
					PrometheusPort:               "9464",
					PrometheusInterface:          "127.0.0.1",
					HighCardinalityLabelsEnabled: false,
				},
				Database: DatabaseConfig{
					DataDir: "/opt/metald/data",
				},
				AssetManager: AssetManagerConfig{
					Enabled:  true,
					Endpoint: "http://localhost:8083",
					CacheDir: "/opt/metald/assets",
				},
				Network: NetworkConfig{
					Enabled:         true,
					EnableIPv4:      true,
					BridgeIPv4:      "10.100.0.1/16",
					VMSubnetIPv4:    "10.100.0.0/16",
					DNSServersIPv4:  []string{"8.8.8.8", "8.8.4.4"},
					EnableIPv6:      true,
					BridgeIPv6:      "fd00::1/64",
					VMSubnetIPv6:    "fd00::/64",
					DNSServersIPv6:  []string{"2606:4700:4700::1111", "2606:4700:4700::1001"},
					IPv6Mode:        "dual-stack",
					BridgeName:      "br-vms",
					EnableRateLimit: true,
					RateLimitMbps:   1000,
				},
				TLS: &TLSConfig{
					Mode:              "spiffe",
					CertFile:          "",
					KeyFile:           "",
					CAFile:            "",
					SPIFFESocketPath:  "/var/lib/spire/agent/agent.sock",
					EnableCertCaching: true,
					CertCacheTTL:      "5s",
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
					Type: types.BackendTypeFirecracker,
					Jailer: JailerConfig{
						UID:           1000,
						GID:           1000,
						ChrootBaseDir: "/srv/jailer",
					},
				},
				Billing: BillingConfig{
					Enabled:  true,
					Endpoint: "http://localhost:8081",
					MockMode: false,
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:                      false,
					ServiceName:                  "metald",
					ServiceVersion:               "0.1.0",
					TracingSamplingRate:          1.0,
					OTLPEndpoint:                 "localhost:4318",
					PrometheusEnabled:            true,
					PrometheusPort:               "9464",
					PrometheusInterface:          "127.0.0.1",
					HighCardinalityLabelsEnabled: false,
				},
				Database: DatabaseConfig{
					DataDir: "/opt/metald/data",
				},
				AssetManager: AssetManagerConfig{
					Enabled:  true,
					Endpoint: "http://localhost:8083",
					CacheDir: "/opt/metald/assets",
				},
				Network: NetworkConfig{
					Enabled:         true,
					EnableIPv4:      true,
					BridgeIPv4:      "10.100.0.1/16",
					VMSubnetIPv4:    "10.100.0.0/16",
					DNSServersIPv4:  []string{"8.8.8.8", "8.8.4.4"},
					EnableIPv6:      true,
					BridgeIPv6:      "fd00::1/64",
					VMSubnetIPv6:    "fd00::/64",
					DNSServersIPv6:  []string{"2606:4700:4700::1111", "2606:4700:4700::1001"},
					IPv6Mode:        "dual-stack",
					BridgeName:      "br-vms",
					EnableRateLimit: true,
					RateLimitMbps:   1000,
				},
				TLS: &TLSConfig{
					Mode:              "spiffe",
					CertFile:          "",
					KeyFile:           "",
					CAFile:            "",
					SPIFFESocketPath:  "/var/lib/spire/agent/agent.sock",
					EnableCertCaching: true,
					CertCacheTTL:      "5s",
				},
			},
			wantErr: false,
		},
		{
			name: "custom jailer configuration",
			envVars: map[string]string{
				"UNKEY_METALD_JAILER_UID":        "2000",
				"UNKEY_METALD_JAILER_GID":        "2000",
				"UNKEY_METALD_JAILER_CHROOT_DIR": "/var/lib/jailer",
			},
			want: &Config{
				Server: ServerConfig{
					Port:    "8080",
					Address: "0.0.0.0",
				},
				Backend: BackendConfig{
					Type: types.BackendTypeFirecracker,
					Jailer: JailerConfig{
						UID:           2000,
						GID:           2000,
						ChrootBaseDir: "/var/lib/jailer",
					},
				},
				Billing: BillingConfig{
					Enabled:  true,
					Endpoint: "http://localhost:8081",
					MockMode: false,
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:                      false,
					ServiceName:                  "metald",
					ServiceVersion:               "0.1.0",
					TracingSamplingRate:          1.0,
					OTLPEndpoint:                 "localhost:4318",
					PrometheusEnabled:            true,
					PrometheusPort:               "9464",
					PrometheusInterface:          "127.0.0.1",
					HighCardinalityLabelsEnabled: false,
				},
				Database: DatabaseConfig{
					DataDir: "/opt/metald/data",
				},
				AssetManager: AssetManagerConfig{
					Enabled:  true,
					Endpoint: "http://localhost:8083",
					CacheDir: "/opt/metald/assets",
				},
				Network: NetworkConfig{
					Enabled:         true,
					EnableIPv4:      true,
					BridgeIPv4:      "10.100.0.1/16",
					VMSubnetIPv4:    "10.100.0.0/16",
					DNSServersIPv4:  []string{"8.8.8.8", "8.8.4.4"},
					EnableIPv6:      true,
					BridgeIPv6:      "fd00::1/64",
					VMSubnetIPv6:    "fd00::/64",
					DNSServersIPv6:  []string{"2606:4700:4700::1111", "2606:4700:4700::1001"},
					IPv6Mode:        "dual-stack",
					BridgeName:      "br-vms",
					EnableRateLimit: true,
					RateLimitMbps:   1000,
				},
				TLS: &TLSConfig{
					Mode:              "spiffe",
					CertFile:          "",
					KeyFile:           "",
					CAFile:            "",
					SPIFFESocketPath:  "/var/lib/spire/agent/agent.sock",
					EnableCertCaching: true,
					CertCacheTTL:      "5s",
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
					Type: types.BackendTypeFirecracker,
					Jailer: JailerConfig{
						UID:           1000,
						GID:           1000,
						ChrootBaseDir: "/srv/jailer",
					},
				},
				Billing: BillingConfig{
					Enabled:  true,
					Endpoint: "http://localhost:8081",
					MockMode: false,
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:                      true,
					ServiceName:                  "test-service",
					ServiceVersion:               "2.0.0",
					TracingSamplingRate:          0.5,
					OTLPEndpoint:                 "otel-collector:4318",
					PrometheusEnabled:            false,
					PrometheusPort:               "8888",
					PrometheusInterface:          "127.0.0.1",
					HighCardinalityLabelsEnabled: false,
				},
				Database: DatabaseConfig{
					DataDir: "/opt/metald/data",
				},
				AssetManager: AssetManagerConfig{
					Enabled:  true,
					Endpoint: "http://localhost:8083",
					CacheDir: "/opt/metald/assets",
				},
				Network: NetworkConfig{
					Enabled:         true,
					EnableIPv4:      true,
					BridgeIPv4:      "10.100.0.1/16",
					VMSubnetIPv4:    "10.100.0.0/16",
					DNSServersIPv4:  []string{"8.8.8.8", "8.8.4.4"},
					EnableIPv6:      true,
					BridgeIPv6:      "fd00::1/64",
					VMSubnetIPv6:    "fd00::/64",
					DNSServersIPv6:  []string{"2606:4700:4700::1111", "2606:4700:4700::1001"},
					IPv6Mode:        "dual-stack",
					BridgeName:      "br-vms",
					EnableRateLimit: true,
					RateLimitMbps:   1000,
				},
				TLS: &TLSConfig{
					Mode:              "spiffe",
					CertFile:          "",
					KeyFile:           "",
					CAFile:            "",
					SPIFFESocketPath:  "/var/lib/spire/agent/agent.sock",
					EnableCertCaching: true,
					CertCacheTTL:      "5s",
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
			name: "invalid sampling rate negative",
			envVars: map[string]string{
				"UNKEY_METALD_OTEL_ENABLED":       "true",
				"UNKEY_METALD_OTEL_SAMPLING_RATE": "-0.5",
			},
			wantErr: true,
			errMsg:  "tracing sampling rate must be between 0.0 and 1.0",
		},
		{
			name: "invalid sampling rate too high",
			envVars: map[string]string{
				"UNKEY_METALD_OTEL_ENABLED":       "true",
				"UNKEY_METALD_OTEL_SAMPLING_RATE": "1.5",
			},
			wantErr: true,
			errMsg:  "tracing sampling rate must be between 0.0 and 1.0",
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

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("LoadConfig() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid firecracker backend",
			config: &Config{
				Backend: BackendConfig{
					Type: types.BackendTypeFirecracker,
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled: false,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid backend type",
			config: &Config{
				Backend: BackendConfig{
					Type: types.BackendTypeCloudHypervisor,
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled: false,
				},
			},
			wantErr: true,
			errMsg:  "only firecracker backend is supported",
		},
		{
			name: "otel enabled with valid config",
			config: &Config{
				Backend: BackendConfig{
					Type: types.BackendTypeFirecracker,
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:             true,
					TracingSamplingRate: 0.5,
					OTLPEndpoint:        "localhost:4318",
					ServiceName:         "test-service",
				},
			},
			wantErr: false,
		},
		{
			name: "otel enabled without service name",
			config: &Config{
				Backend: BackendConfig{
					Type: types.BackendTypeFirecracker,
				},
				OpenTelemetry: OpenTelemetryConfig{
					Enabled:             true,
					TracingSamplingRate: 0.5,
					OTLPEndpoint:        "localhost:4318",
					ServiceName:         "",
				},
			},
			wantErr: true,
			errMsg:  "service name is required when OpenTelemetry is enabled",
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
					t.Errorf("Config.Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// Helper functions

func clearEnv() {
	// Clear all UNKEY_METALD_* environment variables
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "UNKEY_METALD_") {
			key := strings.Split(env, "=")[0]
			os.Unsetenv(key)
		}
	}
}

func compareConfigs(a, b *Config) bool {
	// Compare server config
	if a.Server != b.Server {
		return false
	}

	// Compare backend config
	if a.Backend.Type != b.Backend.Type {
		return false
	}
	if a.Backend.Jailer != b.Backend.Jailer {
		return false
	}

	// Compare process manager config

	// Compare billing config
	if a.Billing != b.Billing {
		return false
	}

	// Compare OpenTelemetry config
	if a.OpenTelemetry != b.OpenTelemetry {
		return false
	}

	// Compare database config
	if a.Database != b.Database {
		return false
	}

	// Compare AssetManager config
	if a.AssetManager != b.AssetManager {
		return false
	}

	// Compare Network config
	if a.Network.Enabled != b.Network.Enabled ||
		a.Network.EnableIPv4 != b.Network.EnableIPv4 ||
		a.Network.BridgeIPv4 != b.Network.BridgeIPv4 ||
		a.Network.VMSubnetIPv4 != b.Network.VMSubnetIPv4 ||
		!stringSlicesEqual(a.Network.DNSServersIPv4, b.Network.DNSServersIPv4) ||
		a.Network.EnableIPv6 != b.Network.EnableIPv6 ||
		a.Network.BridgeIPv6 != b.Network.BridgeIPv6 ||
		a.Network.VMSubnetIPv6 != b.Network.VMSubnetIPv6 ||
		!stringSlicesEqual(a.Network.DNSServersIPv6, b.Network.DNSServersIPv6) ||
		a.Network.IPv6Mode != b.Network.IPv6Mode ||
		a.Network.BridgeName != b.Network.BridgeName ||
		a.Network.EnableRateLimit != b.Network.EnableRateLimit ||
		a.Network.RateLimitMbps != b.Network.RateLimitMbps {
		return false
	}

	// Compare TLS config
	if (a.TLS == nil) != (b.TLS == nil) {
		return false
	}
	if a.TLS != nil && b.TLS != nil {
		if a.TLS.Mode != b.TLS.Mode ||
			a.TLS.CertFile != b.TLS.CertFile ||
			a.TLS.KeyFile != b.TLS.KeyFile ||
			a.TLS.CAFile != b.TLS.CAFile ||
			a.TLS.SPIFFESocketPath != b.TLS.SPIFFESocketPath ||
			a.TLS.EnableCertCaching != b.TLS.EnableCertCaching ||
			a.TLS.CertCacheTTL != b.TLS.CertCacheTTL {
			return false
		}
	}

	return true
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
