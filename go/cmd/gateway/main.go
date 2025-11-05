package gateway

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/gateway"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
)

var Cmd = &cli.Command{
	Version:     "",
	Commands:    []*cli.Command{},
	Aliases:     []string{},
	Description: "",
	Name:        "gateway",
	Usage:       "Run the Unkey Gateway server (per-tenant gateway)",
	Flags: []cli.Flag{
		// Server Configuration
		cli.Int("http-port", "HTTP port for the Gateway server to listen on. Default: 8080",
			cli.Default(8080), cli.EnvVar("UNKEY_HTTP_PORT")),

		cli.Int("https-port", "HTTPS port for the Gateway server to listen on. Default: 8443",
			cli.Default(8443), cli.EnvVar("UNKEY_HTTPS_PORT")),

		cli.Bool("tls-enabled", "Enable TLS for the gateway. Default: false (uses mTLS via SPIRE)",
			cli.Default(false), cli.EnvVar("UNKEY_TLS_ENABLED")),

		// Instance Identification
		cli.String("platform", "Cloud platform identifier for this node. Used for logging and metrics.",
			cli.EnvVar("UNKEY_PLATFORM")),

		cli.String("image", "Container image identifier. Used for logging and metrics.",
			cli.EnvVar("UNKEY_IMAGE")),

		cli.String("region", "Geographic region identifier. Used for logging and routing. Default: unknown",
			cli.Default("unknown"), cli.EnvVar("UNKEY_REGION"), cli.EnvVar("AWS_REGION")),

		cli.String("gateway-id", "Unique identifier for this instance. Auto-generated if not provided.",
			cli.Default(uid.New("gateway", 4)), cli.EnvVar("UNKEY_GATEWAY_ID")),

		cli.String("deployment-id", "Deployment ID this gateway belongs to (e.g., d-abc123). Required.",
			cli.Required(), cli.EnvVar("UNKEY_DEPLOYMENT_ID")),

		cli.String("workspace-id", "Workspace ID this gateway belongs to. Required.",
			cli.Required(), cli.EnvVar("UNKEY_WORKSPACE_ID")),

		cli.String("ctrl-addr", "Address for the control plane to connect to",
			cli.EnvVar("UNKEY_CTRL_ADDR")),

		// Database Configuration - Partitioned (for gateway operations)
		cli.String("database-primary", "MySQL connection string for partitioned primary database (gateway operations). Required. Example: user:pass@host:3306/partition_001?parseTime=true",
			cli.Required(), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),

		cli.String("database-replica", "MySQL connection string for partitioned read-replica (gateway operations). Format same as database-primary.",
			cli.EnvVar("UNKEY_DATABASE_REPLICA")),

		// Database Configuration - Keys Service
		cli.String("main-database-primary", "MySQL connection string for keys service primary database (non-partitioned). Required. Example: user:pass@host:3306/unkey?parseTime=true",
			cli.Required(), cli.EnvVar("UNKEY_KEYS_DATABASE_PRIMARY")),

		cli.String("main-database-replica", "MySQL connection string for keys service read-replica (non-partitioned). Format same as main-database-primary.",
			cli.EnvVar("UNKEY_KEYS_DATABASE_REPLICA")),

		// ClickHouse Configuration
		cli.String("clickhouse-url", "ClickHouse connection string for analytics. Recommended for production. Example: clickhouse://user:pass@host:9000/unkey",
			cli.EnvVar("UNKEY_CLICKHOUSE_URL")),

		// Redis Configuration
		cli.String("redis-url", "Redis connection string for caching. Recommended for production. Example: redis://user:pass@host:6379/0",
			cli.EnvVar("UNKEY_REDIS_URL")),

		// SPIRE/mTLS Configuration
		cli.Bool("spire-enabled", "Enable SPIRE-based mTLS for ingress communication. Default: false",
			cli.Default(false), cli.EnvVar("UNKEY_SPIRE_ENABLED")),

		cli.String("spire-socket-path", "Path to SPIRE agent socket. Default: /run/spire/sockets/agent.sock",
			cli.Default("/run/spire/sockets/agent.sock"), cli.EnvVar("UNKEY_SPIRE_SOCKET_PATH")),

		// Observability
		cli.Bool("otel", "Enable OpenTelemetry tracing and metrics",
			cli.EnvVar("UNKEY_OTEL")),
		cli.Float("otel-trace-sampling-rate", "Sampling rate for OpenTelemetry traces (0.0-1.0). Only used when --otel is provided. Default: 0.25",
			cli.Default(0.25), cli.EnvVar("UNKEY_OTEL_TRACE_SAMPLING_RATE")),
		cli.Int("prometheus-port", "Enable Prometheus /metrics endpoint on specified port. Set to 0 to disable.",
			cli.Default(0), cli.EnvVar("UNKEY_PROMETHEUS_PORT")),

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
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	var vaultS3Config *storage.S3Config
	if cmd.String("vault-s3-url") != "" {
		vaultS3Config = &storage.S3Config{
			Logger:            nil,
			S3URL:             cmd.String("vault-s3-url"),
			S3Bucket:          cmd.String("vault-s3-bucket"),
			S3AccessKeySecret: cmd.String("vault-s3-access-key-secret"),
			S3AccessKeyID:     cmd.String("vault-s3-access-key-id"),
		}
	}

	config := gateway.Config{
		// Basic configuration
		GatewayID:    cmd.String("gateway-id"),
		DeploymentID: cmd.String("deployment-id"),
		WorkspaceID:  cmd.String("workspace-id"),
		Platform:     cmd.String("platform"),
		Image:        cmd.String("image"),
		Region:       cmd.String("region"),

		// HTTP configuration
		HttpPort:  cmd.Int("http-port"),
		HttpsPort: cmd.Int("https-port"),

		// TLS configuration
		EnableTLS: cmd.Bool("tls-enabled"),

		// Control Plane configuration
		CtrlAddr: cmd.String("ctrl-addr"),

		// Partitioned Database configuration (for portal operations)
		DatabasePrimary:         cmd.String("database-primary"),
		DatabaseReadonlyReplica: cmd.String("database-replica"),

		// Keys Database configuration
		MainDatabasePrimary:         cmd.String("main-database-primary"),
		MainDatabaseReadonlyReplica: cmd.String("main-database-replica"),

		// ClickHouse
		ClickhouseURL: cmd.String("clickhouse-url"),

		// Redis Configuration
		RedisURL: cmd.String("redis-url"),

		// SPIRE configuration
		EnableSpire:     cmd.Bool("spire-enabled"),
		SpireSocketPath: cmd.String("spire-socket-path"),

		// OpenTelemetry configuration
		OtelEnabled:           cmd.Bool("otel"),
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),
		PrometheusPort:        cmd.Int("prometheus-port"),

		// Vault configuration
		VaultMasterKeys: cmd.StringSlice("vault-master-keys"),
		VaultS3:         vaultS3Config,
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return gateway.Run(ctx, config)
}
