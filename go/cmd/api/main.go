package api

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/api"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/tls"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name:  "api",
	Usage: "Run the Unkey API server for validating and managing API keys",

	Flags: []cli.Flag{
		// Server Configuration
		&cli.IntFlag{
			Name:     "http-port",
			Usage:    "HTTP port for the API server to listen on. Default: 7070",
			Sources:  cli.EnvVars("UNKEY_HTTP_PORT"),
			Value:    7070,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "color",
			Usage:    "Enable colored log output. Default: true",
			Sources:  cli.EnvVars("UNKEY_LOGS_COLOR"),
			Value:    true,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "test-mode",
			Usage:    "Enable test mode. WARNING: Potentially unsafe, may trust client inputs blindly. Default: false",
			Sources:  cli.EnvVars("UNKEY_TEST_MODE"),
			Value:    false,
			Required: false,
		},

		// Instance Identification
		&cli.StringFlag{
			Name:     "platform",
			Usage:    "Cloud platform identifier for this node. Used for logging and metrics.",
			Sources:  cli.EnvVars("UNKEY_PLATFORM"),
			Required: false,
		},
		&cli.StringFlag{
			Name:     "image",
			Usage:    "Container image identifier. Used for logging and metrics.",
			Sources:  cli.EnvVars("UNKEY_IMAGE"),
			Required: false,
		},
		&cli.StringFlag{
			Name:     "region",
			Usage:    "Geographic region identifier. Used for logging and routing. Default: unknown",
			Sources:  cli.EnvVars("UNKEY_REGION", "AWS_REGION"),
			Value:    "unknown",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "instance-id",
			Usage:    "Unique identifier for this instance. Auto-generated if not provided.",
			Sources:  cli.EnvVars("UNKEY_INSTANCE_ID"),
			Value:    uid.New(uid.InstancePrefix, 4),
			Required: false,
		},

		// Database Configuration
		&cli.StringFlag{
			Name:     "database-primary",
			Usage:    "MySQL connection string for primary database. Required for all deployments. Example: user:pass@host:3306/unkey?parseTime=true",
			Sources:  cli.EnvVars("UNKEY_DATABASE_PRIMARY"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     "database-replica",
			Usage:    "MySQL connection string for read-replica. Reduces load on primary database. Format same as database-primary.",
			Sources:  cli.EnvVars("UNKEY_DATABASE_REPLICA"),
			Required: false,
		},

		// Caching and Storage
		&cli.StringFlag{
			Name:     "redis-url",
			Usage:    "Redis connection string for rate-limiting and distributed counters. Example: redis://localhost:6379",
			Sources:  cli.EnvVars("UNKEY_REDIS_URL"),
			Required: false,
		},
		&cli.StringFlag{
			Name:     "clickhouse-url",
			Usage:    "ClickHouse connection string for analytics. Recommended for production. Example: clickhouse://user:pass@host:9000/unkey",
			Sources:  cli.EnvVars("UNKEY_CLICKHOUSE_URL"),
			Required: false,
		},

		// Observability
		&cli.BoolFlag{
			Name:     "otel",
			Usage:    "Enable OpenTelemetry tracing and metrics",
			Sources:  cli.EnvVars("UNKEY_OTEL"),
			Required: false,
		},

		// TLS Configuration
		&cli.StringFlag{
			Name:      "tls-cert-file",
			Usage:     "Path to TLS certificate file for HTTPS. Both cert and key must be provided to enable HTTPS.",
			Sources:   cli.EnvVars("UNKEY_TLS_CERT_FILE"),
			Required:  false,
			TakesFile: true,
		},
		&cli.StringFlag{
			Name:      "tls-key-file",
			Usage:     "Path to TLS key file for HTTPS. Both cert and key must be provided to enable HTTPS.",
			Sources:   cli.EnvVars("UNKEY_TLS_KEY_FILE"),
			Required:  false,
			TakesFile: true,
		},

		&cli.FloatFlag{
			Name:     "otel-trace-sampling-rate",
			Usage:    "Sampling rate for OpenTelemetry traces (0.0-1.0). Only used when --otel is provided. Default: 0.25",
			Sources:  cli.EnvVars("UNKEY_OTEL_TRACE_SAMPLING_RATE"),
			Value:    0.25,
			Required: false,
		},
		&cli.IntFlag{
			Name:     "prometheus-port",
			Usage:    "Enable Prometheus /metrics endpoint on specified port. Set to 0 to disable.",
			Sources:  cli.EnvVars("UNKEY_PROMETHEUS_PORT"),
			Value:    0,
			Required: false,
		},

		// Vault Configuration
		&cli.StringSliceFlag{
			Name:     "vault-master-keys",
			Usage:    "Vault master keys for encryption",
			Sources:  cli.EnvVars("UNKEY_VAULT_MASTER_KEYS"),
			Value:    []string{},
			Required: false,
		},

		// S3 Configuration
		&cli.StringFlag{
			Name:     "s3-url",
			Usage:    "S3 Compatible Endpoint URL ",
			Sources:  cli.EnvVars("UNKEY_S3_URL"),
			Value:    "",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "s3-bucket",
			Usage:    "S3 bucket name",
			Sources:  cli.EnvVars("UNKEY_S3_BUCKET"),
			Value:    "",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "s3-access-key-id",
			Usage:    "S3 access key ID",
			Sources:  cli.EnvVars("UNKEY_S3_ACCESS_KEY_ID"),
			Value:    "",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "s3-secret-access-key",
			Usage:    "S3 secret access key",
			Sources:  cli.EnvVars("UNKEY_S3_SECRET_ACCESS_KEY"),
			Value:    "",
			Required: false,
		},
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
		HttpPort: cmd.Int("http-port"),
		Region:   cmd.String("region"),

		// Database configuration
		DatabasePrimary:         cmd.String("database-primary"),
		DatabaseReadonlyReplica: cmd.String("database-readonly-replica"),

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

		// Vault configuration
		VaultMasterKeys: cmd.StringSlice("vault-master-keys"),

		// S3 configuration
		S3URL:             cmd.String("s3-url"),
		S3Bucket:          cmd.String("s3-bucket"),
		S3AccessKeyID:     cmd.String("s3-access-key-id"),
		S3SecretAccessKey: cmd.String("s3-secret-access-key"),
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return api.Run(ctx, config)
}
