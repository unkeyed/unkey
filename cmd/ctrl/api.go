package ctrl

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/tls"
	"github.com/unkeyed/unkey/pkg/uid"
	ctrlapi "github.com/unkeyed/unkey/svc/ctrl/api"
)

// apiCmd defines the "api" subcommand for running the control plane HTTP server.
// The server handles infrastructure management, build orchestration, and service
// coordination. It requires a MySQL database (--database-primary) and S3 storage
// for build artifacts. Optional integrations include Vault for secrets, Restate
// for workflows, and ACME for automatic TLS certificates.
var apiCmd = &cli.Command{
	Version:     "",
	Commands:    []*cli.Command{},
	Aliases:     []string{},
	Description: "",
	Name:        "api",
	Usage:       "Run the Unkey control plane service for managing infrastructure and services",
	Flags: []cli.Flag{
		// Server Configuration
		cli.Int("http-port", "HTTP port for the control plane server to listen on. Default: 8080",
			cli.Default(8080), cli.EnvVar("UNKEY_HTTP_PORT")),
		cli.Int("prometheus-port", "Port for Prometheus metrics, set to 0 to disable.",
			cli.Default(0), cli.EnvVar("UNKEY_PROMETHEUS_PORT")),
		cli.Bool("color", "Enable colored log output. Default: true",
			cli.Default(true), cli.EnvVar("UNKEY_LOGS_COLOR")),

		// Instance Identification
		cli.String("platform", "Cloud platform identifier for this node. Used for logging and metrics.",
			cli.EnvVar("UNKEY_PLATFORM")),
		cli.String("region", "Geographic region identifier. Used for logging and routing. Default: unknown",
			cli.Default("unknown"), cli.EnvVar("UNKEY_REGION"), cli.EnvVar("AWS_REGION")),
		cli.String("instance-id", "Unique identifier for this instance. Auto-generated if not provided.",
			cli.Default(uid.New(uid.InstancePrefix, 4)), cli.EnvVar("UNKEY_INSTANCE_ID")),

		// Database Configuration
		cli.String("database-primary", "MySQL connection string for primary database. Required for all deployments. Example: user:pass@host:3306/unkey?parseTime=true",
			cli.Required(), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),

		// Observability
		cli.Bool("otel", "Enable OpenTelemetry tracing and metrics",
			cli.EnvVar("UNKEY_OTEL")),
		cli.Float("otel-trace-sampling-rate", "Sampling rate for OpenTelemetry traces (0.0-1.0). Only used when --otel is provided. Default: 0.25",
			cli.Default(0.25), cli.EnvVar("UNKEY_OTEL_TRACE_SAMPLING_RATE")),

		// TLS Configuration
		cli.String("tls-cert-file", "Path to TLS certificate file for HTTPS. Both cert and key must be provided to enable HTTPS.",
			cli.EnvVar("UNKEY_TLS_CERT_FILE")),
		cli.String("tls-key-file", "Path to TLS key file for HTTPS. Both cert and key must be provided to enable HTTPS.",
			cli.EnvVar("UNKEY_TLS_KEY_FILE")),

		// Control Plane Specific
		cli.String("auth-token", "Authentication token for control plane API access. Required for secure deployments.",
			cli.Required(),
			cli.EnvVar("UNKEY_AUTH_TOKEN")),

		// Restate Configuration
		cli.String("restate-url", "URL of the Restate ingress endpoint for invoking workflows. Example: http://restate:8080",
			cli.Default("http://restate:8080"), cli.EnvVar("UNKEY_RESTATE_INGRESS_URL")),
		cli.String("restate-api-key", "API key for Restate ingress requests",
			cli.EnvVar("UNKEY_RESTATE_API_KEY")),
		cli.String("clickhouse-url", "ClickHouse connection string for analytics. Recommended for production. Example: clickhouse://user:pass@host:9000/unkey",
			cli.EnvVar("UNKEY_CLICKHOUSE_URL")),

		// Build S3 configuration
		cli.String("build-s3-url", "S3 URL for build storage",
			cli.Required(), cli.EnvVar("UNKEY_BUILD_S3_URL")),
		cli.String("build-s3-external-url", "External S3 URL for presigned URLs",
			cli.EnvVar("UNKEY_BUILD_S3_EXTERNAL_URL")),
		cli.String("build-s3-bucket", "S3 bucket for build storage",
			cli.Required(), cli.EnvVar("UNKEY_BUILD_S3_BUCKET")),
		cli.String("build-s3-access-key-id", "S3 access key ID",
			cli.Required(), cli.EnvVar("UNKEY_BUILD_S3_ACCESS_KEY_ID")),
		cli.String("build-s3-access-key-secret", "S3 access key secret",
			cli.Required(), cli.EnvVar("UNKEY_BUILD_S3_ACCESS_KEY_SECRET")),

		cli.StringSlice("available-regions", "Available regions for deployment", cli.EnvVar("UNKEY_AVAILABLE_REGIONS"), cli.Default([]string{"local.dev"})),
	},
	Action: apiAction,
}

// apiAction validates configuration and starts the control plane API server.
// It returns an error if TLS is partially configured (only cert or only key),
// if required configuration is missing, or if the server fails to start.
// The function blocks until the context is cancelled or the server exits.
func apiAction(ctx context.Context, cmd *cli.Command) error {
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

	config := ctrlapi.Config{
		// Basic configuration
		HttpPort:       cmd.Int("http-port"),
		PrometheusPort: cmd.Int("prometheus-port"),
		Region:         cmd.String("region"),
		InstanceID:     cmd.String("instance-id"),

		// Database configuration
		DatabasePrimary: cmd.String("database-primary"),

		// Observability
		OtelEnabled:           cmd.Bool("otel"),
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),

		// TLS Configuration
		TLSConfig: tlsConfig,

		// Control Plane Specific
		AuthToken: cmd.String("auth-token"),

		// Build configuration
		BuildS3: ctrlapi.S3Config{
			URL:             cmd.String("build-s3-url"),
			ExternalURL:     cmd.String("build-s3-external-url"),
			Bucket:          cmd.String("build-s3-bucket"),
			AccessKeySecret: cmd.String("build-s3-access-key-secret"),
			AccessKeyID:     cmd.String("build-s3-access-key-id"),
		},

		// Restate configuration (API is a client, only needs ingress URL)
		Restate: ctrlapi.RestateConfig{
			URL:    cmd.String("restate-url"),
			APIKey: cmd.String("restate-api-key"),
		},

		AvailableRegions: cmd.RequireStringSlice("available-regions"),
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return ctrlapi.Run(ctx, config)
}
