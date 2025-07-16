package run

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/api"
	"github.com/unkeyed/unkey/go/apps/ctrl"
	"github.com/unkeyed/unkey/go/cmd/cli/cli"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/tls"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

var Command = &cli.Command{
	Name:  "run",
	Usage: "Run Unkey services",
	Description: `Run various Unkey services including:
  - api: The main API server for validating and managing API keys
  - ctrl: The control plane service for managing infrastructure`,
	Commands: []*cli.Command{
		apiCommand,
		ctrlCommand,
	},
}

var apiCommand = &cli.Command{
	Name:  "api",
	Usage: "Run the Unkey API server for validating and managing API keys",
	Flags: []cli.Flag{
		// Server Configuration
		cli.Int("http-port", "HTTP port for the API server to listen on", 7070, "UNKEY_HTTP_PORT", false),
		cli.Bool("color", "Enable colored log output", "UNKEY_LOGS_COLOR", false),
		cli.Bool("test-mode", "Enable test mode. WARNING: Potentially unsafe, may trust client inputs blindly", "UNKEY_TEST_MODE", false),

		// Instance Identification
		cli.String("platform", "Cloud platform identifier for this node. Used for logging and metrics", "", "UNKEY_PLATFORM", false),
		cli.String("image", "Container image identifier. Used for logging and metrics", "", "UNKEY_IMAGE", false),
		cli.String("region", "Geographic region identifier. Used for logging and routing", "unknown", "UNKEY_REGION", false),
		cli.String("instance-id", "Unique identifier for this instance. Auto-generated if not provided", uid.New(uid.InstancePrefix, 4), "UNKEY_INSTANCE_ID", false),

		// Database Configuration
		cli.String("database-primary", "MySQL connection string for primary database. Required for all deployments. Example: user:pass@host:3306/unkey?parseTime=true", "", "UNKEY_DATABASE_PRIMARY", true),
		cli.String("database-replica", "MySQL connection string for read-replica. Reduces load on primary database. Format same as database-primary", "", "UNKEY_DATABASE_REPLICA", false),

		// Caching and Storage
		cli.String("redis-url", "Redis connection string for rate-limiting and distributed counters. Example: redis://localhost:6379", "", "UNKEY_REDIS_URL", false),
		cli.String("clickhouse-url", "ClickHouse connection string for analytics. Recommended for production. Example: clickhouse://user:pass@host:9000/unkey", "", "UNKEY_CLICKHOUSE_URL", false),

		// Observability
		cli.Bool("otel", "Enable OpenTelemetry tracing and metrics", "UNKEY_OTEL", false),
		cli.Float("otel-trace-sampling-rate", "Sampling rate for OpenTelemetry traces (0.0-1.0). Only used when --otel is provided", 0.25, "UNKEY_OTEL_TRACE_SAMPLING_RATE", false),
		cli.Int("prometheus-port", "Enable Prometheus /metrics endpoint on specified port. Set to 0 to disable", 0, "UNKEY_PROMETHEUS_PORT", false),

		// TLS Configuration
		cli.String("tls-cert-file", "Path to TLS certificate file for HTTPS. Both cert and key must be provided to enable HTTPS", "", "UNKEY_TLS_CERT_FILE", false),
		cli.String("tls-key-file", "Path to TLS key file for HTTPS. Both cert and key must be provided to enable HTTPS", "", "UNKEY_TLS_KEY_FILE", false),

		// Vault Configuration
		cli.StringSlice("vault-master-keys", "Vault master keys for encryption (comma-separated)", []string{}, "UNKEY_VAULT_MASTER_KEYS", false),
		cli.String("vault-s3-url", "S3 Compatible Endpoint URL", "", "UNKEY_VAULT_S3_URL", false),
		cli.String("vault-s3-bucket", "S3 bucket name", "", "UNKEY_VAULT_S3_BUCKET", false),
		cli.String("vault-s3-access-key-id", "S3 access key ID", "", "UNKEY_VAULT_S3_ACCESS_KEY_ID", false),
		cli.String("vault-s3-secret-access-key", "S3 secret access key", "", "UNKEY_VAULT_S3_SECRET_ACCESS_KEY", false),
	},
	Action: apiAction,
}

var ctrlCommand = &cli.Command{
	Name:  "ctrl",
	Usage: "Run the Unkey control plane service for managing infrastructure and services",
	Flags: []cli.Flag{
		// Server Configuration
		cli.Int("http-port", "HTTP port for the control plane server to listen on", 8080, "UNKEY_HTTP_PORT", false),
		cli.Bool("color", "Enable colored log output", "UNKEY_LOGS_COLOR", false),

		// Instance Identification
		cli.String("platform", "Cloud platform identifier for this node. Used for logging and metrics", "", "UNKEY_PLATFORM", false),
		cli.String("image", "Container image identifier. Used for logging and metrics", "", "UNKEY_IMAGE", false),
		cli.String("region", "Geographic region identifier. Used for logging and routing", "unknown", "UNKEY_REGION", false),
		cli.String("instance-id", "Unique identifier for this instance. Auto-generated if not provided", uid.New(uid.InstancePrefix, 4), "UNKEY_INSTANCE_ID", false),

		// Database Configuration
		cli.String("database-primary", "MySQL connection string for primary database. Required for all deployments. Example: user:pass@host:3306/unkey?parseTime=true", "", "UNKEY_DATABASE_PRIMARY", true),
		cli.String("database-hydra", "MySQL connection string for hydra database. Required for all deployments. Example: user:pass@host:3306/hydra?parseTime=true", "", "UNKEY_DATABASE_HYDRA", true),

		// Observability
		cli.Bool("otel", "Enable OpenTelemetry tracing and metrics", "UNKEY_OTEL", false),
		cli.Float("otel-trace-sampling-rate", "Sampling rate for OpenTelemetry traces (0.0-1.0). Only used when --otel is provided", 0.25, "UNKEY_OTEL_TRACE_SAMPLING_RATE", false),

		// TLS Configuration
		cli.String("tls-cert-file", "Path to TLS certificate file for HTTPS. Both cert and key must be provided to enable HTTPS", "", "UNKEY_TLS_CERT_FILE", false),
		cli.String("tls-key-file", "Path to TLS key file for HTTPS. Both cert and key must be provided to enable HTTPS", "", "UNKEY_TLS_KEY_FILE", false),

		// Control Plane Specific
		cli.String("auth-token", "Authentication token for control plane API access. Required for secure deployments", "", "UNKEY_AUTH_TOKEN", false),
		cli.String("metald-address", "Full URL of the metald service for VM operations. Required for deployments. Example: https://metald.example.com:8080", "", "UNKEY_METALD_ADDRESS", true),
		cli.String("spiffe-socket-path", "Path to SPIFFE agent socket for mTLS authentication", "/var/lib/spire/agent/agent.sock", "UNKEY_SPIFFE_SOCKET_PATH", false),
	},
	Action: ctrlAction,
}

func apiAction(ctx context.Context, cmd *cli.Command) error {
	// Check if TLS flags are properly set (both or none)
	tlsCertFile := cmd.String("tls-cert-file")
	tlsKeyFile := cmd.String("tls-key-file")
	if (tlsCertFile == "" && tlsKeyFile != "") || (tlsCertFile != "" && tlsKeyFile == "") {
		return fmt.Errorf("both --tls-cert-file and --tls-key-file must be provided to enable HTTPS")
	}

	// Initialize TLS config if TLS flags are provided
	var tlsConfig *tls.Config
	if tlsCertFile != "" && tlsKeyFile != "" {
		var err error
		tlsConfig, err = tls.NewFromFiles(tlsCertFile, tlsKeyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS configuration: %w", err)
		}
	}

	// Parse vault master keys from StringSlice flag
	vaultMasterKeys := cmd.StringSlice("vault-master-keys")

	// Get sampling rate directly as float
	samplingRate := cmd.Float("otel-trace-sampling-rate")

	var vaultS3Config *api.S3Config
	if cmd.String("vault-s3-url") != "" {
		vaultS3Config = &api.S3Config{
			URL:             cmd.String("vault-s3-url"),
			Bucket:          cmd.String("vault-s3-bucket"),
			AccessKeyID:     cmd.String("vault-s3-access-key-id"),
			SecretAccessKey: cmd.String("vault-s3-secret-access-key"),
		}
	}

	config := api.Config{
		// Basic configuration
		Platform: cmd.String("platform"),
		Image:    cmd.String("image"),
		HttpPort: cmd.Int("http-port"),
		Region:   cmd.String("region"),

		// Database configuration
		DatabasePrimary:         cmd.String("database-primary"),
		DatabaseReadonlyReplica: cmd.String("database-replica"),

		// ClickHouse
		ClickhouseURL: cmd.String("clickhouse-url"),

		// OpenTelemetry configuration
		OtelEnabled:           cmd.Bool("otel"),
		OtelTraceSamplingRate: samplingRate,

		// TLS Configuration
		TLSConfig: tlsConfig,

		InstanceID:     cmd.String("instance-id"),
		RedisUrl:       cmd.String("redis-url"),
		PrometheusPort: cmd.Int("prometheus-port"),
		Clock:          clock.New(),
		TestMode:       cmd.Bool("test-mode"),

		// Vault configuration
		VaultMasterKeys: vaultMasterKeys,
		VaultS3:         vaultS3Config,
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return api.Run(ctx, config)
}

func ctrlAction(ctx context.Context, cmd *cli.Command) error {
	// Check if TLS flags are properly set (both or none)
	tlsCertFile := cmd.String("tls-cert-file")
	tlsKeyFile := cmd.String("tls-key-file")
	if (tlsCertFile == "" && tlsKeyFile != "") || (tlsCertFile != "" && tlsKeyFile == "") {
		return fmt.Errorf("both --tls-cert-file and --tls-key-file must be provided to enable HTTPS")
	}

	// Initialize TLS config if TLS flags are provided
	var tlsConfig *tls.Config
	if tlsCertFile != "" && tlsKeyFile != "" {
		var err error
		tlsConfig, err = tls.NewFromFiles(tlsCertFile, tlsKeyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS configuration: %w", err)
		}
	}

	// Get sampling rate directly as float
	samplingRate := cmd.Float("otel-trace-sampling-rate")

	config := ctrl.Config{
		// Basic configuration
		Platform:   cmd.String("platform"),
		Image:      cmd.String("image"),
		HttpPort:   cmd.Int("http-port"),
		Region:     cmd.String("region"),
		InstanceID: cmd.String("instance-id"),

		// Database configuration
		DatabasePrimary: cmd.String("database-primary"),
		DatabaseHydra:   cmd.String("database-hydra"),

		// Observability
		OtelEnabled:           cmd.Bool("otel"),
		OtelTraceSamplingRate: samplingRate,

		// TLS Configuration
		TLSConfig: tlsConfig,

		// Control Plane Specific
		AuthToken:        cmd.String("auth-token"),
		MetaldAddress:    cmd.String("metald-address"),
		SPIFFESocketPath: cmd.String("spiffe-socket-path"),

		// Common
		Clock: clock.New(),
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return ctrl.Run(ctx, config)
}
