package ctrl

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/tls"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl"
)

var Cmd = &cli.Command{
	Version:     "",
	Commands:    []*cli.Command{},
	Aliases:     []string{},
	Description: "",
	Name:        "ctrl",
	Usage:       "Run the Unkey control plane service for managing infrastructure and services",
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
		cli.String("spiffe-socket-path", "Path to SPIFFE agent socket for mTLS authentication. Default: /var/lib/spire/agent/agent.sock",
			cli.Default("/var/lib/spire/agent/agent.sock"), cli.EnvVar("UNKEY_SPIFFE_SOCKET_PATH")),

		// Vault Configuration - General secrets (env vars, API keys)
		cli.StringSlice("vault-master-keys", "Vault master keys for encryption (general vault)",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_MASTER_KEYS")),
		cli.String("vault-s3-url", "S3 endpoint URL for general vault",
			cli.EnvVar("UNKEY_VAULT_S3_URL")),
		cli.String("vault-s3-bucket", "S3 bucket for general vault (env vars, API keys)",
			cli.EnvVar("UNKEY_VAULT_S3_BUCKET")),
		cli.String("vault-s3-access-key-id", "S3 access key ID for general vault",
			cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_ID")),
		cli.String("vault-s3-access-key-secret", "S3 secret access key for general vault",
			cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_SECRET")),

		// ACME Vault Configuration - Let's Encrypt certificates
		cli.StringSlice("acme-vault-master-keys", "Vault master keys for encryption (ACME vault)",
			cli.EnvVar("UNKEY_ACME_VAULT_MASTER_KEYS")),
		cli.String("acme-vault-s3-url", "S3 endpoint URL for ACME vault",
			cli.EnvVar("UNKEY_ACME_VAULT_S3_URL")),
		cli.String("acme-vault-s3-bucket", "S3 bucket for ACME vault (Let's Encrypt certs)",
			cli.EnvVar("UNKEY_ACME_VAULT_S3_BUCKET")),
		cli.String("acme-vault-s3-access-key-id", "S3 access key ID for ACME vault",
			cli.EnvVar("UNKEY_ACME_VAULT_S3_ACCESS_KEY_ID")),
		cli.String("acme-vault-s3-access-key-secret", "S3 secret access key for ACME vault",
			cli.EnvVar("UNKEY_ACME_VAULT_S3_ACCESS_KEY_SECRET")),

		// Build Configuration
		cli.String("build-backend", "Build backend to use: 'docker' for local, 'depot' for production. Default: depot",
			cli.Default("depot"), cli.EnvVar("UNKEY_BUILD_BACKEND")),
		cli.String("build-s3-url", "S3 Compatible Endpoint URL for build contexts (internal)",
			cli.Required(), cli.EnvVar("UNKEY_BUILD_S3_URL")),
		cli.String("build-s3-external-url", "S3 Compatible Endpoint URL for build contexts (external/public)",
			cli.EnvVar("UNKEY_BUILD_S3_EXTERNAL_URL")),
		cli.String("build-s3-bucket", "S3 bucket name for build contexts",
			cli.Required(), cli.EnvVar("UNKEY_BUILD_S3_BUCKET")),
		cli.String("build-s3-access-key-id", "S3 access key ID for build contexts",
			cli.Required(), cli.EnvVar("UNKEY_BUILD_S3_ACCESS_KEY_ID")),
		cli.String("build-s3-access-key-secret", "S3 secret access key for build contexts",
			cli.Required(), cli.EnvVar("UNKEY_BUILD_S3_ACCESS_KEY_SECRET")),

		cli.String("registry-url", "URL of the container registry for pulling images. Example: registry.depot.dev",
			cli.EnvVar("UNKEY_REGISTRY_URL")),
		cli.String("registry-username", "Username for authenticating with the container registry.",
			cli.EnvVar("UNKEY_REGISTRY_USERNAME")),
		cli.String("registry-password", "Password/token for authenticating with the container registry.",
			cli.EnvVar("UNKEY_REGISTRY_PASSWORD")),
		cli.String("build-platform", "Run builds on this platform ('dynamic', 'linux/amd64', 'linux/arm64')",
			cli.EnvVar("UNKEY_BUILD_PLATFORM"), cli.Default("linux/amd64")),
		// Depot Build Backend Configuration
		cli.String("depot-api-url", "Depot API endpoint URL",
			cli.EnvVar("UNKEY_DEPOT_API_URL")),
		cli.String("depot-project-region", "Build data will be stored in the chosen region ('us-east-1','eu-central-1')",
			cli.EnvVar("UNKEY_DEPOT_PROJECT_REGION"), cli.Default("us-east-1")),

		cli.Bool("acme-enabled", "Enable Let's Encrypt for acme challenges", cli.EnvVar("UNKEY_ACME_ENABLED")),
		cli.String("acme-email-domain", "Domain for ACME registration emails (workspace_id@domain)", cli.Default("unkey.com"), cli.EnvVar("UNKEY_ACME_EMAIL_DOMAIN")),

		// Cloudflare DNS provider
		cli.Bool("acme-cloudflare-enabled", "Enable Cloudflare for wildcard certificates", cli.EnvVar("UNKEY_ACME_CLOUDFLARE_ENABLED")),
		cli.String("acme-cloudflare-api-token", "Cloudflare API token for Let's Encrypt", cli.EnvVar("UNKEY_ACME_CLOUDFLARE_API_TOKEN")),

		// Route53 DNS provider
		cli.Bool("acme-route53-enabled", "Enable Route53 for DNS-01 challenges", cli.EnvVar("UNKEY_ACME_ROUTE53_ENABLED")),
		cli.String("acme-route53-access-key-id", "AWS access key ID for Route53", cli.EnvVar("UNKEY_ACME_ROUTE53_ACCESS_KEY_ID")),
		cli.String("acme-route53-secret-access-key", "AWS secret access key for Route53", cli.EnvVar("UNKEY_ACME_ROUTE53_SECRET_ACCESS_KEY")),
		cli.String("acme-route53-region", "AWS region for Route53", cli.Default("us-east-1"), cli.EnvVar("UNKEY_ACME_ROUTE53_REGION")),
		cli.String("acme-route53-hosted-zone-id", "Route53 hosted zone ID (bypasses auto-discovery, required when wildcard CNAMEs exist)", cli.EnvVar("UNKEY_ACME_ROUTE53_HOSTED_ZONE_ID")),

		cli.String("default-domain", "Default domain for auto-generated hostnames", cli.Default("unkey.app"), cli.EnvVar("UNKEY_DEFAULT_DOMAIN")),

		// Restate Configuration
		cli.String("restate-frontline-url", "URL of the Restate frontline endpoint for invoking workflows. Example: http://restate:8080",
			cli.Default("http://restate:8080"), cli.EnvVar("UNKEY_RESTATE_INGRESS_URL")),
		cli.String("restate-admin-url", "URL of the Restate admin endpoint for service registration. Example: http://restate:9070",
			cli.Default("http://restate:9070"), cli.EnvVar("UNKEY_RESTATE_ADMIN_URL")),
		cli.Int("restate-http-port", "Port where we listen for Restate HTTP requests. Example: 9080",
			cli.Default(9080), cli.EnvVar("UNKEY_RESTATE_HTTP_PORT")),
		cli.String("restate-register-as", "URL of this service for self-registration with Restate. Example: http://ctrl:9080",
			cli.EnvVar("UNKEY_RESTATE_REGISTER_AS")),
		cli.String("clickhouse-url", "ClickHouse connection string for analytics. Recommended for production. Example: clickhouse://user:pass@host:9000/unkey",
			cli.EnvVar("UNKEY_CLICKHOUSE_URL")),

		// The image new sentinels get deployed with
		cli.String("sentinel-image", "The image new sentinels get deployed with", cli.Default("ghcr.io/unkeyed/unkey:local"), cli.EnvVar("UNKEY_SENTINEL_IMAGE")),
		cli.StringSlice("available-regions", "Available regions for deployment", cli.EnvVar("UNKEY_AVAILABLE_REGIONS"), cli.Default([]string{"dev:local"})),
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
		Platform:         cmd.String("platform"),
		BuildPlatform:    cmd.String("build-platform"),
		Image:            cmd.String("image"),
		HttpPort:         cmd.Int("http-port"),
		Region:           cmd.String("region"),
		InstanceID:       cmd.String("instance-id"),
		RegistryURL:      cmd.String("registry-url"),
		RegistryUsername: cmd.String("registry-username"),
		RegistryPassword: cmd.String("registry-password"),

		// Database configuration
		DatabasePrimary: cmd.String("database-primary"),

		// Observability
		OtelEnabled:           cmd.Bool("otel"),
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),

		// TLS Configuration
		TLSConfig: tlsConfig,

		// Control Plane Specific
		AuthToken:        cmd.String("auth-token"),
		SPIFFESocketPath: cmd.String("spiffe-socket-path"),

		// Vault configuration - General secrets
		VaultMasterKeys: cmd.StringSlice("vault-master-keys"),
		VaultS3: ctrl.S3Config{
			URL:             cmd.String("vault-s3-url"),
			Bucket:          cmd.String("vault-s3-bucket"),
			AccessKeyID:     cmd.String("vault-s3-access-key-id"),
			AccessKeySecret: cmd.String("vault-s3-access-key-secret"),
			ExternalURL:     "",
		},
		// ACME Vault configuration - Let's Encrypt certificates
		AcmeVaultMasterKeys: cmd.StringSlice("acme-vault-master-keys"),
		AcmeVaultS3: ctrl.S3Config{
			URL:             cmd.String("acme-vault-s3-url"),
			Bucket:          cmd.String("acme-vault-s3-bucket"),
			AccessKeyID:     cmd.String("acme-vault-s3-access-key-id"),
			AccessKeySecret: cmd.String("acme-vault-s3-access-key-secret"),
			ExternalURL:     "",
		},

		// Build configuration
		BuildBackend: ctrl.BuildBackend(cmd.String("build-backend")),
		BuildS3: ctrl.S3Config{
			URL:             cmd.String("build-s3-url"),
			ExternalURL:     cmd.String("build-s3-external-url"),
			Bucket:          cmd.String("build-s3-bucket"),
			AccessKeySecret: cmd.String("build-s3-access-key-secret"),
			AccessKeyID:     cmd.String("build-s3-access-key-id"),
		},

		// Depot build backend configuration
		Depot: ctrl.DepotConfig{
			APIUrl:        cmd.String("depot-api-url"),
			ProjectRegion: cmd.String("depot-project-region"),
		},

		// Acme configuration
		Acme: ctrl.AcmeConfig{
			Enabled:     cmd.Bool("acme-enabled"),
			EmailDomain: cmd.String("acme-email-domain"),
			Cloudflare: ctrl.CloudflareConfig{
				Enabled:  cmd.Bool("acme-cloudflare-enabled"),
				ApiToken: cmd.String("acme-cloudflare-api-token"),
			},
			Route53: ctrl.Route53Config{
				Enabled:         cmd.Bool("acme-route53-enabled"),
				AccessKeyID:     cmd.String("acme-route53-access-key-id"),
				SecretAccessKey: cmd.String("acme-route53-secret-access-key"),
				Region:          cmd.String("acme-route53-region"),
				HostedZoneID:    cmd.String("acme-route53-hosted-zone-id"),
			},
		},

		DefaultDomain: cmd.String("default-domain"),

		// Restate configuration
		Restate: ctrl.RestateConfig{
			FrontlineURL: cmd.String("restate-frontline-url"),
			AdminURL:     cmd.String("restate-admin-url"),
			HttpPort:     cmd.Int("restate-http-port"),
			RegisterAs:   cmd.String("restate-register-as"),
		},

		// Clickhouse Configuration
		ClickhouseURL: cmd.String("clickhouse-url"),

		// Common
		Clock: clock.New(),

		// Sentinel configuration
		SentinelImage:    cmd.String("sentinel-image"),
		AvailableRegions: cmd.RequireStringSlice("available-regions"),
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return ctrl.Run(ctx, config)
}
