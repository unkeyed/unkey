package ctrl

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/ctrl"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/tls"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

var Cmd = &cli.Command{
	Name:  "ctrl",
	Usage: "Run the Unkey control plane service for managing infrastructure and services",

	Flags: []cli.Flag{
		// Server Configuration
		cli.Int("http-port", "HTTP port for the control plane server to listen on. Default: 8080",
			cli.Default(8080), cli.EnvVar("UNKEY_HTTP_PORT")),
		cli.Bool("color", "Enable colored log output. Default: true",
			cli.Default(true), cli.EnvVar("UNKEY_LOGS_COLOR")),

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
		cli.String("database-hydra", "MySQL connection string for hydra database. Required for all deployments. Example: user:pass@host:3306/hydra?parseTime=true",
			cli.Required(), cli.EnvVar("UNKEY_DATABASE_HYDRA")),

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
			cli.EnvVar("UNKEY_AUTH_TOKEN")),
		cli.String("metald-address", "Full URL of the metald service for VM operations. Required for deployments. Example: https://metald.example.com:8080",
			cli.Required(), cli.EnvVar("UNKEY_METALD_ADDRESS")),
		cli.String("spiffe-socket-path", "Path to SPIFFE agent socket for mTLS authentication. Default: /var/lib/spire/agent/agent.sock",
			cli.Default("/var/lib/spire/agent/agent.sock"), cli.EnvVar("UNKEY_SPIFFE_SOCKET_PATH")),
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
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),

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
