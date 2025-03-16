package main

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Setup and cleanup for all tests
	setupEnv := func(t *testing.T, envs map[string]string) {
		t.Helper()
		for k, v := range envs {
			os.Setenv(k, v)
		}
	}

	cleanEnv := func(t *testing.T, keys []string) {
		t.Helper()
		for _, k := range keys {
			os.Unsetenv(k)
		}
	}

	// Required env vars that need to be cleaned up
	envKeys := []string{
		"CLICKHOUSE_URL",
		"BASIC_AUTH",
		"PORT",
		"OTEL_EXPORTER_LOG_DEBUG",
		"OTEL_TRACE_SAMPLE_RATE",
	}

	// Cleanup after all tests in this function
	t.Cleanup(func() {
		cleanEnv(t, envKeys)
	})

	// Test for valid config with default values
	t.Run("ValidConfig", func(t *testing.T) {
		// Set required environment variables
		setupEnv(t, map[string]string{
			"CLICKHOUSE_URL": "http://localhost:8123",
			"BASIC_AUTH":     "user:password",
		})

		config, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig() error = %v, wantErr = false", err)
		}

		// Use a table-driven check for expected values
		tests := []struct {
			name     string
			got      interface{}
			expected interface{}
		}{
			{"FlushInterval", config.FlushInterval, time.Second * 5},
			{"ListenerPort", config.ListenerPort, "7123"},
			{"LogDebug", config.LogDebug, false},
			{"ServiceName", config.ServiceName, "chproxy"},
			{"ServiceVersion", config.ServiceVersion, "1.3.1"},
			{"TraceMaxBatchSize", config.TraceMaxBatchSize, 512},
			{"TraceSampleRate", config.TraceSampleRate, 0.25},
			{"ClickhouseURL", config.ClickhouseURL, "http://localhost:8123"},
			{"BasicAuth", config.BasicAuth, "user:password"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if tc.got != tc.expected {
					t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.expected)
				}
			})
		}

		if config.Logger == nil {
			t.Error("Logger is nil, want non-nil")
		}
	})

	// Test for error cases
	t.Run("ErrorCases", func(t *testing.T) {
		tests := []struct {
			name    string
			envs    map[string]string
			wantErr string
		}{
			{
				name: "MissingClickhouseURL",
				envs: map[string]string{
					"BASIC_AUTH": "user:password",
				},
				wantErr: "CLICKHOUSE_URL must be defined",
			},
			{
				name: "MissingBasicAuth",
				envs: map[string]string{
					"CLICKHOUSE_URL": "http://localhost:8123",
				},
				wantErr: "BASIC_AUTH must be defined",
			},
			{
				name: "InvalidSampleRate",
				envs: map[string]string{
					"CLICKHOUSE_URL":         "http://localhost:8123",
					"BASIC_AUTH":             "user:password",
					"OTEL_TRACE_SAMPLE_RATE": "invalid",
				},
				wantErr: "invalid TRACE_SAMPLE_RATE",
			},
			{
				name: "SampleRateTooLow",
				envs: map[string]string{
					"CLICKHOUSE_URL":         "http://localhost:8123",
					"BASIC_AUTH":             "user:password",
					"OTEL_TRACE_SAMPLE_RATE": "-0.1",
				},
				wantErr: "OTEL_TRACE_SAMPLE_RATE must be between 0.0 and 1.0",
			},
			{
				name: "SampleRateTooHigh",
				envs: map[string]string{
					"CLICKHOUSE_URL":         "http://localhost:8123",
					"BASIC_AUTH":             "user:password",
					"OTEL_TRACE_SAMPLE_RATE": "1.1",
				},
				wantErr: "OTEL_TRACE_SAMPLE_RATE must be between 0.0 and 1.0",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// Clean environment before each test case
				cleanEnv(t, envKeys)

				// Set up environment for this test case
				setupEnv(t, tc.envs)

				_, err := LoadConfig()
				if err == nil {
					t.Fatalf("LoadConfig() error = nil, wantErr = true")
				}

				if got := err.Error(); got != tc.wantErr && got[:len(tc.wantErr)] != tc.wantErr {
					t.Errorf("LoadConfig() error = %v, want to contain %v", got, tc.wantErr)
				}
			})
		}
	})

	// Test for custom configurations
	t.Run("CustomConfigurations", func(t *testing.T) {
		// Table-driven tests for custom configurations
		tests := []struct {
			name      string
			envs      map[string]string
			checkFunc func(*testing.T, *Config)
		}{
			{
				name: "CustomPort",
				envs: map[string]string{
					"CLICKHOUSE_URL": "http://localhost:8123",
					"BASIC_AUTH":     "user:password",
					"PORT":           "8000",
				},
				checkFunc: func(t *testing.T, c *Config) {
					if c.ListenerPort != "8000" {
						t.Errorf("ListenerPort = %s, want 8000", c.ListenerPort)
					}
				},
			},
			{
				name: "DebugMode",
				envs: map[string]string{
					"CLICKHOUSE_URL":          "http://localhost:8123",
					"BASIC_AUTH":              "user:password",
					"OTEL_EXPORTER_LOG_DEBUG": "true",
				},
				checkFunc: func(t *testing.T, c *Config) {
					if !c.LogDebug {
						t.Errorf("LogDebug = %v, want true", c.LogDebug)
					}
				},
			},
			{
				name: "CustomSampleRate",
				envs: map[string]string{
					"CLICKHOUSE_URL":         "http://localhost:8123",
					"BASIC_AUTH":             "user:password",
					"OTEL_TRACE_SAMPLE_RATE": "0.5",
				},
				checkFunc: func(t *testing.T, c *Config) {
					if c.TraceSampleRate != 0.5 {
						t.Errorf("TraceSampleRate = %f, want 0.5", c.TraceSampleRate)
					}
				},
			},
			{
				name: "ZeroSampleRate",
				envs: map[string]string{
					"CLICKHOUSE_URL":         "http://localhost:8123",
					"BASIC_AUTH":             "user:password",
					"OTEL_TRACE_SAMPLE_RATE": "0.0",
				},
				checkFunc: func(t *testing.T, c *Config) {
					if c.TraceSampleRate != 0.0 {
						t.Errorf("TraceSampleRate = %f, want 0.0", c.TraceSampleRate)
					}
				},
			},
			{
				name: "OneSampleRate",
				envs: map[string]string{
					"CLICKHOUSE_URL":         "http://localhost:8123",
					"BASIC_AUTH":             "user:password",
					"OTEL_TRACE_SAMPLE_RATE": "1.0",
				},
				checkFunc: func(t *testing.T, c *Config) {
					if c.TraceSampleRate != 1.0 {
						t.Errorf("TraceSampleRate = %f, want 1.0", c.TraceSampleRate)
					}
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				// Clean environment before each test case
				cleanEnv(t, envKeys)

				// Set up environment for this test case
				setupEnv(t, tc.envs)

				config, err := LoadConfig()
				if err != nil {
					t.Fatalf("LoadConfig() error = %v, wantErr = false", err)
				}

				tc.checkFunc(t, config)
			})
		}
	})
}
