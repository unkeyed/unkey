package api

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/api"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/tls"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

var Cmd = &cli.Command{
	Name:  "api",
	Usage: "Run the Unkey API server for validating and managing API keys",

	Flags: []cli.Flag{
		// Server Configuration
		cli.Int("http-port", "HTTP port for the API server to listen on. Default: 7070",
			cli.Default(7070), cli.EnvVar("UNKEY_HTTP_PORT")),
		cli.Bool("color", "Enable colored log output. Default: true",
			cli.Default(true), cli.EnvVar("UNKEY_LOGS_COLOR")),
		cli.Bool("test-mode", "Enable test mode. WARNING: Potentially unsafe, may trust client inputs blindly. Default: false",
			cli.Default(false), cli.EnvVar("UNKEY_TEST_MODE")),

		// Instance Identification
		cli.String("platform", "Cloud platform identifier for this node. Used for logging and metrics.",
			cli.EnvVar("UNKEY_PLATFORM")),
		cli.String("image", "Container image identifier. Used for logging and metrics.",
			cli.EnvVar("UNKEY_IMAGE")),
		cli.String("region", "Geographic region identifier. Used for logging and routing. Default: unknown",
			cli.Default("unknown"), cli.EnvVar("UNKEY_REGION"), cli.EnvVar("AWS_REGION")),
		cli.String("instance-id", "Unique identifier for this instance. Auto-generated if not provided.",
			cli.Default(uid.New(uid.InstancePrefix, 4)), cli.EnvVar("UNKEY_INSTANCE_ID")),

		// Database Configuration
		cli.String("database-primary", "MySQL connection string for primary database. Required for all deployments. Example: user:pass@host:3306/unkey?parseTime=true",
			cli.Required(), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
		cli.String("database-replica", "MySQL connection string for read-replica. Reduces load on primary database. Format same as database-primary.",
			cli.EnvVar("UNKEY_DATABASE_REPLICA")),

		// Caching and Storage
		cli.String("redis-url", "Redis connection string for rate-limiting and distributed counters. Example: redis://localhost:6379",
			cli.EnvVar("UNKEY_REDIS_URL")),
		cli.String("clickhouse-url", "ClickHouse connection string for analytics. Recommended for production. Example: clickhouse://user:pass@host:9000/unkey",
			cli.EnvVar("UNKEY_CLICKHOUSE_URL")),

		// Observability
		cli.Bool("otel", "Enable OpenTelemetry tracing and metrics",
			cli.EnvVar("UNKEY_OTEL")),
		cli.Float("otel-trace-sampling-rate", "Sampling rate for OpenTelemetry traces (0.0-1.0). Only used when --otel is provided. Default: 0.25",
			cli.Default(0.25), cli.EnvVar("UNKEY_OTEL_TRACE_SAMPLING_RATE")),
		cli.Int("prometheus-port", "Enable Prometheus /metrics endpoint on specified port. Set to 0 to disable.",
			cli.Default(0), cli.EnvVar("UNKEY_PROMETHEUS_PORT")),

		// TLS Configuration
		cli.String("tls-cert-file", "Path to TLS certificate file for HTTPS. Both cert and key must be provided to enable HTTPS.",
			cli.EnvVar("UNKEY_TLS_CERT_FILE")),
		cli.String("tls-key-file", "Path to TLS key file for HTTPS. Both cert and key must be provided to enable HTTPS.",
			cli.EnvVar("UNKEY_TLS_KEY_FILE")),

		// Vault Configuration
		cli.StringSlice("vault-master-keys", "Vault master keys for encryption",
			cli.EnvVar("UNKEY_VAULT_MASTER_KEYS")),
		cli.String("vault-s3-url", "S3 Compatible Endpoint URL",
			cli.EnvVar("UNKEY_VAULT_S3_URL")),
		cli.String("vault-s3-bucket", "S3 bucket name",
			cli.EnvVar("UNKEY_VAULT_S3_BUCKET")),
		cli.String("vault-s3-access-key-id", "S3 access key ID",
			cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_ID")),
		cli.String("vault-s3-access-key-secret", "S3 secret access key",
			cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_SECRET")),

		// Kafka Configuration
		cli.StringSlice("kafka-brokers", "Comma-separated list of Kafka broker addresses for distributed cache invalidation",
			cli.EnvVar("UNKEY_KAFKA_BROKERS")),

		// ClickHouse Proxy Service Configuration
		cli.String(
			"chproxy-auth-token",
			"Authentication token for ClickHouse proxy endpoints. Required when proxy is enabled.",
			cli.EnvVar("UNKEY_CHPROXY_AUTH_TOKEN"),
		),

		// Request Body Configuration
		cli.Int64("max-request-body-size", "Maximum allowed request body size in bytes. Set to 0 or negative to disable limit. Default: 10485760 (10MB)",
			cli.Default(int64(10485760)), cli.EnvVar("UNKEY_MAX_REQUEST_BODY_SIZE")),

		// Debug Configuration
		cli.Bool("debug-cache-headers", "Enable cache debug headers (X-Unkey-Debug-Cache) in HTTP responses for debugging cache behavior",
			cli.Default(false), cli.EnvVar("UNKEY_DEBUG_CACHE_HEADERS")),
	},

	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	// Check if TLS flags are properly set (both or none)
	tlsCertFile := cmd.String("tls-cert-file")
	tlsKeyFile := cmd.String("tls-key-file")
	if (tlsCertFile == "" && tlsKeyFile != "") || (tlsCertFile != "" && tlsKeyFile == "") {
		return cli.Exit("Both --tls-cert-file and --tls-key-file must be provided to enable HTTPS", 1)
	}

	// Initialize TLS config if TLS flags are provided
	var tlsConfig *tls.Config
	if tlsCertFile != "" && tlsKeyFile != "" {
		var err error
		tlsConfig, err = tls.NewFromFiles(tlsCertFile, tlsKeyFile)
		if err != nil {
			return cli.Exit("Failed to load TLS configuration: "+err.Error(), 1)
		}
	}

	var vaultS3Config *api.S3Config
	if cmd.String("vault-s3-url") != "" {
		vaultS3Config = &api.S3Config{
			URL:             cmd.String("vault-s3-url"),
			Bucket:          cmd.String("vault-s3-bucket"),
			AccessKeyID:     cmd.String("vault-s3-access-key-id"),
			AccessKeySecret: cmd.String("vault-s3-access-key-secret"),
		}
	}

	config := api.Config{
		// Basic configuration
		Platform: cmd.String("platform"),
		Image:    cmd.String("image"),
		Region:   cmd.String("region"),

		// Database configuration
		DatabasePrimary:         cmd.String("database-primary"),
		DatabaseReadonlyReplica: cmd.String("database-replica"),

		// ClickHouse
		ClickhouseURL: cmd.String("clickhouse-url"),

		// OpenTelemetry configuration
		OtelEnabled:           cmd.Bool("otel"),
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),

		// TLS Configuration
		TLSConfig: tlsConfig,

		InstanceID:     cmd.String("instance-id"),
		RedisUrl:       cmd.String("redis-url"),
		PrometheusPort: cmd.Int("prometheus-port"),
		Clock:          clock.New(),
		TestMode:       cmd.Bool("test-mode"),

		// HTTP configuration
		HttpPort: cmd.Int("http-port"),
		Listener: nil, // Production uses HttpPort

		// Vault configuration
		VaultMasterKeys: cmd.StringSlice("vault-master-keys"),
		VaultS3:         vaultS3Config,

		// Kafka configuration
		KafkaBrokers: cmd.StringSlice("kafka-brokers"),

		// ClickHouse proxy configuration
		ChproxyToken: cmd.String("chproxy-auth-token"),

		// Request body configuration
		MaxRequestBodySize: cmd.Int64("max-request-body-size"),

		// Debug configuration
		DebugCacheHeaders: cmd.Bool("debug-cache-headers"),
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return api.Run(ctx, config)
}
