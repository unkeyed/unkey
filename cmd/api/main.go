package api

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/tls"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api"
)

// Cmd is the api command that runs the Unkey API server for validating and managing
// API keys, rate limiting, and analytics.
var Cmd = &cli.Command{
	Aliases:     []string{},
	Description: "",
	Version:     "",
	Commands:    []*cli.Command{},
	Name:        "api",
	Usage:       "Run the Unkey API server for validating and managing API keys",

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
		cli.String("clickhouse-analytics-url", "ClickHouse base URL for workspace-specific analytics connections. Workspace credentials are injected programmatically. Example: http://clickhouse:8123/default",
			cli.EnvVar("UNKEY_CLICKHOUSE_ANALYTICS_URL")),

		// Observability
		cli.Bool("otel", "Enable OpenTelemetry tracing and metrics",
			cli.EnvVar("UNKEY_OTEL")),
		cli.Float("otel-trace-sampling-rate", "Sampling rate for OpenTelemetry traces (0.0-1.0). Only used when --otel is provided. Default: 0.25",
			cli.Default(0.25), cli.EnvVar("UNKEY_OTEL_TRACE_SAMPLING_RATE")),
		cli.Int("prometheus-port", "Enable Prometheus /metrics endpoint on specified port. Set to 0 to disable.", cli.EnvVar("UNKEY_PROMETHEUS_PORT")),

		// TLS Configuration
		cli.String("tls-cert-file", "Path to TLS certificate file for HTTPS. Both cert and key must be provided to enable HTTPS.",
			cli.EnvVar("UNKEY_TLS_CERT_FILE")),
		cli.String("tls-key-file", "Path to TLS key file for HTTPS. Both cert and key must be provided to enable HTTPS.",
			cli.EnvVar("UNKEY_TLS_KEY_FILE")),

		// Vault Configuration
		cli.String("vault-url", "URL of the remote vault service for encryption/decryption",
			cli.EnvVar("UNKEY_VAULT_URL")),
		cli.String("vault-token", "Bearer token for vault service authentication",
			cli.EnvVar("UNKEY_VAULT_TOKEN")),

		// Gossip Cluster Configuration
		cli.Bool("gossip-enabled", "Enable gossip-based distributed cache invalidation",
			cli.Default(false), cli.EnvVar("UNKEY_GOSSIP_ENABLED")),
		cli.String("gossip-bind-addr", "Address for gossip listeners. Default: 0.0.0.0",
			cli.Default("0.0.0.0"), cli.EnvVar("UNKEY_GOSSIP_BIND_ADDR")),
		cli.Int("gossip-lan-port", "LAN memberlist port. Default: 7946",
			cli.Default(7946), cli.EnvVar("UNKEY_GOSSIP_LAN_PORT")),
		cli.Int("gossip-wan-port", "WAN memberlist port for bridges. Default: 7947",
			cli.Default(7947), cli.EnvVar("UNKEY_GOSSIP_WAN_PORT")),
		cli.StringSlice("gossip-lan-seeds", "LAN seed addresses (e.g. k8s headless service DNS)",
			cli.EnvVar("UNKEY_GOSSIP_LAN_SEEDS")),
		cli.StringSlice("gossip-wan-seeds", "Cross-region bridge seed addresses",
			cli.EnvVar("UNKEY_GOSSIP_WAN_SEEDS")),
		cli.String("gossip-secret-key", "Base64-encoded AES-256 key for encrypting gossip traffic",
			cli.EnvVar("UNKEY_GOSSIP_SECRET_KEY")),

		// ClickHouse Proxy Service Configuration
		cli.String(
			"chproxy-auth-token",
			"Authentication token for ClickHouse proxy endpoints. Required when proxy is enabled.",
			cli.EnvVar("UNKEY_CHPROXY_AUTH_TOKEN"),
		),

		// Profiling Configuration
		cli.Bool(
			"pprof-enabled",
			"Enable pprof profiling endpoints at /debug/pprof/*. Default: false",
			cli.Default(false),
			cli.EnvVar("UNKEY_PPROF_ENABLED"),
		),
		cli.String(
			"pprof-username",
			"Username for pprof Basic Auth. Optional - if username and password are not set, pprof will be accessible without authentication.",
			cli.EnvVar("UNKEY_PPROF_USERNAME"),
		),
		cli.String(
			"pprof-password",
			"Password for pprof Basic Auth. Optional - if username and password are not set, pprof will be accessible without authentication.",
			cli.EnvVar("UNKEY_PPROF_PASSWORD"),
		),

		// Request Body Configuration
		cli.Int64("max-request-body-size", "Maximum allowed request body size in bytes. Set to 0 or negative to disable limit. Default: 10485760 (10MB)",
			cli.Default(int64(10485760)), cli.EnvVar("UNKEY_MAX_REQUEST_BODY_SIZE")),

		// Logging Sampler Configuration
		cli.Float("log-sample-rate", "Baseline probability (0.0-1.0) of emitting log events. Default: 1.0",
			cli.Default(1.0), cli.EnvVar("UNKEY_LOG_SAMPLE_RATE")),
		cli.Duration("log-slow-threshold", "Duration threshold for slow event sampling. Default: 1s",
			cli.Default(time.Second), cli.EnvVar("UNKEY_LOG_SLOW_THRESHOLD")),

		// CTRL Service Configuration
		cli.String("ctrl-url", "CTRL service connection URL for deployment management. Example: http://ctrl:7091",
			cli.EnvVar("UNKEY_CTRL_URL")),
		cli.String("ctrl-token", "Bearer token for CTRL service authentication",
			cli.EnvVar("UNKEY_CTRL_TOKEN")),
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

	config := api.Config{
		// Basic configuration
		Platform: cmd.String("platform"),
		Image:    cmd.String("image"),
		Region:   cmd.String("region"),

		// Database configuration
		DatabasePrimary:         cmd.String("database-primary"),
		DatabaseReadonlyReplica: cmd.String("database-replica"),

		// ClickHouse
		ClickhouseURL:          cmd.String("clickhouse-url"),
		ClickhouseAnalyticsURL: cmd.String("clickhouse-analytics-url"),

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
		VaultURL:   cmd.String("vault-url"),
		VaultToken: cmd.String("vault-token"),

		// Gossip cluster configuration
		GossipEnabled:   cmd.Bool("gossip-enabled"),
		GossipBindAddr:  cmd.String("gossip-bind-addr"),
		GossipLANPort:   cmd.Int("gossip-lan-port"),
		GossipWANPort:   cmd.Int("gossip-wan-port"),
		GossipLANSeeds:  cmd.StringSlice("gossip-lan-seeds"),
		GossipWANSeeds:  cmd.StringSlice("gossip-wan-seeds"),
		GossipSecretKey: cmd.String("gossip-secret-key"),

		// ClickHouse proxy configuration
		ChproxyToken: cmd.String("chproxy-auth-token"),

		// CTRL service configuration
		CtrlURL:   cmd.String("ctrl-url"),
		CtrlToken: cmd.String("ctrl-token"),

		// Profiling configuration
		PprofEnabled:  cmd.Bool("pprof-enabled"),
		PprofUsername: cmd.String("pprof-username"),
		PprofPassword: cmd.String("pprof-password"),

		// Request body configuration
		MaxRequestBodySize: cmd.Int64("max-request-body-size"),

		// Logging sampler configuration
		LogSampleRate:    cmd.Float("log-sample-rate"),
		LogSlowThreshold: cmd.Duration("log-slow-threshold"),
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return api.Run(ctx, config)
}
