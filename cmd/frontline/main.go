package frontline

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/vault/storage"
	"github.com/unkeyed/unkey/svc/frontline"
)

// Cmd is the frontline command that runs the multi-tenant ingress server for TLS
// termination and request routing to backend services.
var Cmd = &cli.Command{
	Version:     "",
	Commands:    []*cli.Command{},
	Aliases:     []string{},
	Description: "",
	Name:        "frontline",
	Usage:       "Run the Unkey Frontline server (multi-tenant frontline)",
	Flags: []cli.Flag{
		// Server Configuration
		cli.Int("http-port", "HTTP port for the Gate server to listen on. Default: 7070",
			cli.Default(7070), cli.EnvVar("UNKEY_HTTP_PORT")),

		cli.Int("https-port", "HTTPS port for the Gate server to listen on. Default: 7443",
			cli.Default(7443), cli.EnvVar("UNKEY_HTTPS_PORT")),

		cli.Bool("tls-enabled", "Enable TLS termination for the frontline. Default: true",
			cli.Default(true), cli.EnvVar("UNKEY_TLS_ENABLED")),

		cli.String("region", "Geographic region identifier. Used for logging and routing. Default: unknown",
			cli.Default("unknown"), cli.EnvVar("UNKEY_REGION"), cli.EnvVar("AWS_REGION")),

		cli.String("frontline-id", "Unique identifier for this instance. Auto-generated if not provided.",
			cli.Default(uid.New("frontline", 4)), cli.EnvVar("UNKEY_GATE_ID")),

		cli.String("default-cert-domain", "Domain to use for fallback TLS certificate when a domain has no cert configured",
			cli.EnvVar("UNKEY_DEFAULT_CERT_DOMAIN")),

		cli.String("base-domain", "Base domain for region routing. Cross-region requests forwarded to region.base-domain. Example: unkey.cloud",
			cli.Default("unkey.cloud"), cli.EnvVar("UNKEY_BASE_DOMAIN")),

		// Database Configuration - Partitioned (for hostname lookups)
		cli.String("database-primary", "MySQL connection string for partitioned primary database (frontline operations). Required. Example: user:pass@host:3306/unkey?parseTime=true",
			cli.Required(), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),

		cli.String("database-replica", "MySQL connection string for partitioned read-replica (frontline operations). Format same as database-primary.",
			cli.EnvVar("UNKEY_DATABASE_REPLICA")),

		// Observability
		cli.Bool("otel", "Enable OpenTelemetry tracing and metrics",
			cli.EnvVar("UNKEY_OTEL")),
		cli.Float("otel-trace-sampling-rate", "Sampling rate for OpenTelemetry traces (0.0-1.0). Only used when --otel is provided. Default: 0.25",
			cli.Default(0.25), cli.EnvVar("UNKEY_OTEL_TRACE_SAMPLING_RATE")),
		cli.Int("prometheus-port", "Enable Prometheus /metrics endpoint on specified port. Set to 0 to disable.", cli.EnvVar("UNKEY_PROMETHEUS_PORT")),

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

		cli.Int("max-hops", "Maximum number of hops allowed for a request",
			cli.Default(10), cli.EnvVar("UNKEY_MAX_HOPS")),

		cli.String("ctrl-addr", "Address of the control plane",
			cli.Default("localhost:8080"), cli.EnvVar("UNKEY_CTRL_ADDR")),
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

	config := frontline.Config{
		// Basic configuration
		FrontlineID: cmd.String("frontline-id"),
		Image:       cmd.String("image"),
		Region:      cmd.String("region"),

		// HTTP configuration
		HttpPort:  cmd.Int("http-port"),
		HttpsPort: cmd.Int("https-port"),

		// TLS configuration
		EnableTLS:  cmd.Bool("tls-enabled"),
		BaseDomain: cmd.String("base-domain"),
		MaxHops:    cmd.Int("max-hops"),

		// Control Plane Configuration
		CtrlAddr: cmd.String("ctrl-addr"),

		// Partitioned Database configuration (for hostname lookups)
		DatabasePrimary:         cmd.String("database-primary"),
		DatabaseReadonlyReplica: cmd.String("database-replica"),

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

	return frontline.Run(ctx, config)
}
