package ctrl

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/ctrl"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/tls"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name:  "ctrl",
	Usage: "Run the Unkey control plane service for managing infrastructure and services",

	Flags: []cli.Flag{
		// Server Configuration
		&cli.IntFlag{
			Name:     "http-port",
			Usage:    "HTTP port for the control plane server to listen on. Default: 8080",
			Sources:  cli.EnvVars("UNKEY_HTTP_PORT"),
			Value:    8080,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "color",
			Usage:    "Enable colored log output. Default: true",
			Sources:  cli.EnvVars("UNKEY_LOGS_COLOR"),
			Value:    true,
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
			Name:     "database-hydra",
			Usage:    "MySQL connection string for hydra database. Required for all deployments. Example: user:pass@host:3306/hydra?parseTime=true",
			Sources:  cli.EnvVars("UNKEY_DATABASE_HYDRA"),
			Required: true,
		},

		// Observability
		&cli.BoolFlag{
			Name:     "otel",
			Usage:    "Enable OpenTelemetry tracing and metrics",
			Sources:  cli.EnvVars("UNKEY_OTEL"),
			Required: false,
		},
		&cli.FloatFlag{
			Name:     "otel-trace-sampling-rate",
			Usage:    "Sampling rate for OpenTelemetry traces (0.0-1.0). Only used when --otel is provided. Default: 0.25",
			Sources:  cli.EnvVars("UNKEY_OTEL_TRACE_SAMPLING_RATE"),
			Value:    0.25,
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

		// Control Plane Specific
		&cli.StringFlag{
			Name:     "auth-token",
			Usage:    "Authentication token for control plane API access. Required for secure deployments.",
			Sources:  cli.EnvVars("UNKEY_AUTH_TOKEN"),
			Required: false,
		},
		&cli.StringFlag{
			Name:     "metald-address",
			Usage:    "Full URL of the metald service for VM operations. Required for deployments. Example: https://metald.example.com:8080",
			Sources:  cli.EnvVars("UNKEY_METALD_ADDRESS"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     "spiffe-socket-path",
			Usage:    "Path to SPIFFE agent socket for mTLS authentication. Default: /var/lib/spire/agent/agent.sock",
			Sources:  cli.EnvVars("UNKEY_SPIFFE_SOCKET_PATH"),
			Value:    "/var/lib/spire/agent/agent.sock",
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
