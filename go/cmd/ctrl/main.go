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
		cli.String("database-partition", "MySQL connection string for partition database. Required for all deployments. Example: user:pass@host:3306/partition_002?parseTime=true",
			cli.Required(), cli.EnvVar("UNKEY_DATABASE_PARTITION")),

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
		cli.String("krane-address", "Full URL of the krane service for VM operations. Required for deployments. Example: https://krane.example.com:8080",
			cli.Required(), cli.EnvVar("UNKEY_KRANE_ADDRESS")),
		cli.String("api-key", "API key for simple authentication (demo purposes only). Will be replaced with JWT authentication.",
			cli.Required(), cli.EnvVar("UNKEY_API_KEY")),
		cli.String("spiffe-socket-path", "Path to SPIFFE agent socket for mTLS authentication. Default: /var/lib/spire/agent/agent.sock",
			cli.Default("/var/lib/spire/agent/agent.sock"), cli.EnvVar("UNKEY_SPIFFE_SOCKET_PATH")),

		// Vault Configuration
		cli.StringSlice("vault-master-keys", "Vault master keys for encryption",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_MASTER_KEYS")),
		cli.String("vault-s3-url", "S3 Compatible Endpoint URL",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_S3_URL")),
		cli.String("vault-s3-bucket", "S3 bucket name",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_S3_BUCKET")),
		cli.String("vault-s3-access-key-id", "S3 access key ID",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_ID")),
		cli.String("vault-s3-access-key-secret", "S3 secret access key",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_SECRET")),

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
		cli.Bool("acme-cloudflare-enabled", "Enable Cloudflare for wildcard certificates", cli.EnvVar("UNKEY_ACME_CLOUDFLARE_ENABLED")),
		cli.String("acme-cloudflare-api-token", "Cloudflare API token for Let's Encrypt", cli.EnvVar("UNKEY_ACME_CLOUDFLARE_API_TOKEN")),

		cli.String("default-domain", "Default domain for auto-generated hostnames", cli.Default("unkey.app"), cli.EnvVar("UNKEY_DEFAULT_DOMAIN")),

		// Restate Configuration
		cli.String("restate-ingress-url", "URL of the Restate ingress endpoint for invoking workflows. Example: http://restate:8080",
			cli.Default("http://restate:8080"), cli.EnvVar("UNKEY_RESTATE_INGRESS_URL")),
		cli.String("restate-admin-url", "URL of the Restate admin endpoint for service registration. Example: http://restate:9070",
			cli.Default("http://restate:9070"), cli.EnvVar("UNKEY_RESTATE_ADMIN_URL")),
		cli.Int("restate-http-port", "Port where we listen for Restate HTTP requests. Example: 9080",
			cli.Default(9080), cli.EnvVar("UNKEY_RESTATE_HTTP_PORT")),
		cli.String("restate-register-as", "URL of this service for self-registration with Restate. Example: http://ctrl:9080",
			cli.EnvVar("UNKEY_RESTATE_REGISTER_AS")),
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
		DatabasePrimary:   cmd.String("database-primary"),
		DatabasePartition: cmd.String("database-partition"),

		// Observability
		OtelEnabled:           cmd.Bool("otel"),
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),

		// TLS Configuration
		TLSConfig: tlsConfig,

		// Control Plane Specific
		AuthToken:        cmd.String("auth-token"),
		KraneAddress:     cmd.String("krane-address"),
		APIKey:           cmd.String("api-key"),
		SPIFFESocketPath: cmd.String("spiffe-socket-path"),

		// Vault configuration
		VaultMasterKeys: cmd.StringSlice("vault-master-keys"),
		VaultS3: ctrl.S3Config{
			ExternalURL:     cmd.String(""),
			URL:             cmd.String("vault-s3-url"),
			Bucket:          cmd.String("vault-s3-bucket"),
			AccessKeySecret: cmd.String("vault-s3-access-key-secret"),
			AccessKeyID:     cmd.String("vault-s3-access-key-id"),
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
			Enabled: cmd.Bool("acme-enabled"),
			Cloudflare: ctrl.CloudflareConfig{
				Enabled:  cmd.Bool("acme-cloudflare-enabled"),
				ApiToken: cmd.String("acme-cloudflare-api-token"),
			},
		},

		DefaultDomain: cmd.String("default-domain"),

		// Restate configuration
		Restate: ctrl.RestateConfig{
			IngressURL: cmd.String("restate-ingress-url"),
			AdminURL:   cmd.String("restate-admin-url"),
			HttpPort:   cmd.Int("restate-http-port"),
			RegisterAs: cmd.String("restate-register-as"),
		},

		// Common
		Clock: clock.New(),
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return ctrl.Run(ctx, config)
}
