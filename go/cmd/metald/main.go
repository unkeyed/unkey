package metald

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/metald"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

var Cmd = &cli.Command{
	Name:  "metald",
	Usage: "Run the Metald VM management service",
	Description: `Metald is the VM management service for Unkey infrastructure.

It manages the lifecycle of virtual machines using different backends:
- firecracker: High-performance microVMs (Linux only)
- docker: Container-based VMs for development
- k8s: Kubernetes pod-based VMs for cloud environments

EXAMPLES:
unkey run metald                                   # Run with default configuration
unkey run metald --backend docker                  # Use Docker backend for development
unkey run metald --backend k8s                     # Use Kubernetes backend
unkey run metald --backend firecracker              # Use Firecracker (Linux only)`,

	Flags: []cli.Flag{
		// Server Configuration
		cli.String("address", "Bind address for the server. Default: 0.0.0.0",
			cli.Default("0.0.0.0"), cli.EnvVar("UNKEY_METALD_ADDRESS")),
		cli.String("port", "Port for the server to listen on. Default: 8080",
			cli.Default("8080"), cli.EnvVar("UNKEY_METALD_PORT")),

		// Backend Configuration
		cli.String("backend", "Backend type: firecracker, docker, or k8s. Default: docker",
			cli.Default("docker"), cli.EnvVar("UNKEY_METALD_BACKEND")),

		// Database Configuration
		cli.String("database-dir", "Directory for local database storage. Default: /tmp/metald",
			cli.Default("/tmp/metald"), cli.EnvVar("UNKEY_METALD_DATABASE_DIR")),

		// Jailer Configuration (Firecracker only)
		cli.Int("jailer-uid", "User ID for jailer process (firecracker only). Default: 1000",
			cli.Default(1000), cli.EnvVar("UNKEY_METALD_JAILER_UID")),
		cli.Int("jailer-gid", "Group ID for jailer process (firecracker only). Default: 1000",
			cli.Default(1000), cli.EnvVar("UNKEY_METALD_JAILER_GID")),
		cli.String("jailer-chroot-dir", "Chroot base directory (firecracker only). Default: /srv/jailer",
			cli.Default("/srv/jailer"), cli.EnvVar("UNKEY_METALD_JAILER_CHROOT_DIR")),

		// AssetManager Configuration
		cli.Bool("assetmanager-enabled", "Enable AssetManager for kernel/rootfs management. Default: false",
			cli.Default(false), cli.EnvVar("UNKEY_METALD_ASSETMANAGER_ENABLED")),
		cli.String("assetmanager-endpoint", "AssetManager service endpoint",
			cli.EnvVar("UNKEY_METALD_ASSETMANAGER_ENDPOINT")),

		// Billing Configuration
		cli.Bool("billing-enabled", "Enable billing integration. Default: false",
			cli.Default(false), cli.EnvVar("UNKEY_METALD_BILLING_ENABLED")),
		cli.String("billing-endpoint", "Billing service endpoint",
			cli.EnvVar("UNKEY_METALD_BILLING_ENDPOINT")),
		cli.Bool("billing-mock", "Use mock billing client for testing. Default: false",
			cli.Default(false), cli.EnvVar("UNKEY_METALD_BILLING_MOCK")),

		// TLS Configuration
		cli.String("tls-mode", "TLS mode: disabled, file, or spiffe. Default: disabled",
			cli.Default("disabled"), cli.EnvVar("UNKEY_METALD_TLS_MODE")),
		cli.String("tls-cert-file", "Path to TLS certificate file (file mode only)",
			cli.EnvVar("UNKEY_METALD_TLS_CERT_FILE")),
		cli.String("tls-key-file", "Path to TLS key file (file mode only)",
			cli.EnvVar("UNKEY_METALD_TLS_KEY_FILE")),
		cli.String("tls-ca-file", "Path to CA certificate file",
			cli.EnvVar("UNKEY_METALD_TLS_CA_FILE")),
		cli.String("tls-spiffe-socket", "SPIFFE Workload API socket path. Default: /tmp/spire-agent/public/api.sock",
			cli.Default("/tmp/spire-agent/public/api.sock"), cli.EnvVar("UNKEY_METALD_TLS_SPIFFE_SOCKET")),

		// OpenTelemetry Configuration
		cli.Bool("otel-enabled", "Enable OpenTelemetry tracing and metrics. Default: false",
			cli.Default(false), cli.EnvVar("UNKEY_METALD_OTEL_ENABLED")),
		cli.String("otel-service-name", "Service name for OpenTelemetry. Default: metald",
			cli.Default("metald"), cli.EnvVar("UNKEY_METALD_OTEL_SERVICE_NAME")),
		cli.String("otel-service-version", "Service version for OpenTelemetry. Default: 0.1.0",
			cli.Default("0.1.0"), cli.EnvVar("UNKEY_METALD_OTEL_SERVICE_VERSION")),
		cli.Float("otel-sampling-rate", "Trace sampling rate (0.0-1.0). Default: 1.0",
			cli.Default(1.0), cli.EnvVar("UNKEY_METALD_OTEL_SAMPLING_RATE")),
		cli.String("otel-endpoint", "OTLP endpoint. Default: localhost:4318",
			cli.Default("localhost:4318"), cli.EnvVar("UNKEY_METALD_OTEL_ENDPOINT")),
		cli.Bool("otel-prometheus-enabled", "Enable Prometheus metrics. Default: true",
			cli.Default(true), cli.EnvVar("UNKEY_METALD_OTEL_PROMETHEUS_ENABLED")),
		cli.String("otel-prometheus-port", "Prometheus metrics port. Default: 9464",
			cli.Default("9464"), cli.EnvVar("UNKEY_METALD_OTEL_PROMETHEUS_PORT")),
		cli.String("otel-prometheus-interface", "Prometheus metrics interface. Default: 127.0.0.1",
			cli.Default("127.0.0.1"), cli.EnvVar("UNKEY_METALD_OTEL_PROMETHEUS_INTERFACE")),
		cli.Bool("otel-high-cardinality", "Enable high-cardinality labels. Default: false",
			cli.Default(false), cli.EnvVar("UNKEY_METALD_OTEL_HIGH_CARDINALITY_ENABLED")),

		// Instance Identification
		cli.String("instance-id", "Unique identifier for this instance. Auto-generated if not provided.",
			cli.Default(uid.New(uid.InstancePrefix, 4)), cli.EnvVar("UNKEY_METALD_INSTANCE_ID")),
	},

	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	config := metald.Config{
		Server: metald.ServerConfig{
			Address: cmd.String("address"),
			Port:    cmd.String("port"),
		},
		Backend: metald.BackendConfig{
			Type: cmd.String("backend"),
			Jailer: metald.JailerConfig{
				UID:           uint32(cmd.Int("jailer-uid")),
				GID:           uint32(cmd.Int("jailer-gid")),
				ChrootBaseDir: cmd.String("jailer-chroot-dir"),
			},
		},
		Database: metald.DatabaseConfig{
			DataDir: cmd.String("database-dir"),
		},
		AssetManager: metald.AssetManagerConfig{
			Enabled:  cmd.Bool("assetmanager-enabled"),
			Endpoint: cmd.String("assetmanager-endpoint"),
		},
		Billing: metald.BillingConfig{
			Enabled:  cmd.Bool("billing-enabled"),
			Endpoint: cmd.String("billing-endpoint"),
			MockMode: cmd.Bool("billing-mock"),
		},
		TLS: metald.TLSConfig{
			Mode:              cmd.String("tls-mode"),
			CertFile:          cmd.String("tls-cert-file"),
			KeyFile:           cmd.String("tls-key-file"),
			CAFile:            cmd.String("tls-ca-file"),
			SPIFFESocketPath:  cmd.String("tls-spiffe-socket"),
			EnableCertCaching: true,
			CertCacheTTL:      "5s",
		},
		OpenTelemetry: metald.OpenTelemetryConfig{
			Enabled:                      cmd.Bool("otel-enabled"),
			ServiceName:                  cmd.String("otel-service-name"),
			ServiceVersion:               cmd.String("otel-service-version"),
			TracingSamplingRate:          cmd.Float("otel-sampling-rate"),
			OTLPEndpoint:                 cmd.String("otel-endpoint"),
			PrometheusEnabled:            cmd.Bool("otel-prometheus-enabled"),
			PrometheusPort:               cmd.String("otel-prometheus-port"),
			PrometheusInterface:          cmd.String("otel-prometheus-interface"),
			HighCardinalityLabelsEnabled: cmd.Bool("otel-high-cardinality"),
		},
		InstanceID: cmd.String("instance-id"),
	}

	// Validate configuration
	err := config.Validate()
	if err != nil {
		return cli.Exit("Invalid configuration: "+err.Error(), 1)
	}

	// Run metald
	return metald.Run(ctx, config)
}
