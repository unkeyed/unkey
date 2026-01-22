package ctrl

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/worker"
)

// workerCmd defines the "worker" subcommand for running the background job
// processor. The worker handles durable workflows via Restate including container
// builds, deployments, and ACME certificate provisioning. It supports two build
// backends: "docker" for local development and "depot" for production.
var workerCmd = &cli.Command{
	Version:     "",
	Commands:    []*cli.Command{},
	Aliases:     []string{},
	Description: "",
	Name:        "worker",
	Usage:       "Run the Unkey Restate worker service for background jobs and workflows",
	Flags: []cli.Flag{
		// Server Configuration
		cli.Int("http-port", "HTTP port for the health endpoint. Default: 7092",
			cli.Default(7092), cli.EnvVar("UNKEY_WORKER_HTTP_PORT")),
		cli.Int("prometheus-port", "Port for Prometheus metrics, set to 0 to disable.",
			cli.Default(0), cli.EnvVar("UNKEY_PROMETHEUS_PORT")),

		// Instance Identification
		cli.String("instance-id", "Unique identifier for this instance. Auto-generated if not provided.",
			cli.Default(uid.New(uid.InstancePrefix, 4)), cli.EnvVar("UNKEY_INSTANCE_ID")),

		// Database Configuration
		cli.String("database-primary", "MySQL connection string for primary database. Required for all deployments. Example: user:pass@host:3306/unkey?parseTime=true",
			cli.Required(), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),

		// Authentication
		cli.String("auth-token", "Authentication token for worker API access.",
			cli.EnvVar("UNKEY_AUTH_TOKEN")),

		cli.String("vault-url", "Url where vault is availab;e",
			cli.EnvVar("UNKEY_VAULT_URL"), cli.Default("https://vault.unkey.cloud")),

		cli.String("vault-token", "Authentication for vault",
			cli.EnvVar("UNKEY_VAULT_TOKEN")),

		// Build Configuration
		cli.String("build-backend", "Build backend to use: 'docker' for local, 'depot' for production. Default: depot",
			cli.Default("depot"), cli.EnvVar("UNKEY_BUILD_BACKEND")),
		cli.String("build-s3-url", "S3 Compatible Endpoint URL for build contexts",
			cli.Required(), cli.EnvVar("UNKEY_BUILD_S3_URL")),
		cli.String("build-s3-bucket", "S3 bucket name for build contexts",
			cli.Required(), cli.EnvVar("UNKEY_BUILD_S3_BUCKET")),
		cli.String("build-s3-access-key-id", "S3 access key ID for build contexts",
			cli.Required(), cli.EnvVar("UNKEY_BUILD_S3_ACCESS_KEY_ID")),
		cli.String("build-s3-access-key-secret", "S3 secret access key for build contexts",
			cli.Required(), cli.EnvVar("UNKEY_BUILD_S3_ACCESS_KEY_SECRET")),
		cli.String("build-platform", "Run builds on this platform ('dynamic', 'linux/amd64', 'linux/arm64')",
			cli.Default("linux/amd64"), cli.EnvVar("UNKEY_BUILD_PLATFORM")),

		// Registry Configuration
		cli.String("registry-url", "URL of the container registry for pulling images. Example: registry.depot.dev",
			cli.EnvVar("UNKEY_REGISTRY_URL")),
		cli.String("registry-username", "Username for authenticating with the container registry.",
			cli.EnvVar("UNKEY_REGISTRY_USERNAME")),
		cli.String("registry-password", "Password/token for authenticating with the container registry.",
			cli.EnvVar("UNKEY_REGISTRY_PASSWORD")),

		// Depot Build Backend Configuration
		cli.String("depot-api-url", "Depot API endpoint URL",
			cli.EnvVar("UNKEY_DEPOT_API_URL")),
		cli.String("depot-project-region", "Build data will be stored in the chosen region ('us-east-1','eu-central-1')",
			cli.EnvVar("UNKEY_DEPOT_PROJECT_REGION"), cli.Default("us-east-1")),

		// ACME Configuration
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
		cli.String("restate-url", "URL of the Restate ingress endpoint for invoking workflows. Example: http://restate:8080",
			cli.Default("http://restate:8080"), cli.EnvVar("UNKEY_RESTATE_INGRESS_URL")),
		cli.String("restate-admin-url", "URL of the Restate admin endpoint for service registration. Example: http://restate:9070",
			cli.Default("http://restate:9070"), cli.EnvVar("UNKEY_RESTATE_ADMIN_URL")),
		cli.Int("restate-http-port", "Port where we listen for Restate HTTP requests. Example: 9080",
			cli.Default(9080), cli.EnvVar("UNKEY_RESTATE_HTTP_PORT")),
		cli.String("restate-register-as", "URL of this service for self-registration with Restate. Example: http://worker:9080",
			cli.EnvVar("UNKEY_RESTATE_REGISTER_AS")),
		cli.String("restate-api-key", "API key for Restate ingress requests",
			cli.EnvVar("UNKEY_RESTATE_API_KEY")),

		// ClickHouse Configuration
		cli.String("clickhouse-url", "ClickHouse connection string for analytics. Required. Example: clickhouse://user:pass@host:9000/unkey",
			cli.EnvVar("UNKEY_CLICKHOUSE_URL")),

		// Sentinel configuration
		cli.String("sentinel-image", "The image new sentinels get deployed with", cli.Default("ghcr.io/unkeyed/unkey:local"), cli.EnvVar("UNKEY_SENTINEL_IMAGE")),
		cli.StringSlice("available-regions", "Available regions for deployment", cli.EnvVar("UNKEY_AVAILABLE_REGIONS"), cli.Default([]string{"local.dev"})),
	},
	Action: workerAction,
}

// workerAction validates configuration and starts the background worker service.
// It returns an error if required configuration is missing or if the worker fails
// to start. The function blocks until the context is cancelled or the worker exits.
func workerAction(ctx context.Context, cmd *cli.Command) error {
	config := worker.Config{
		// Basic configuration
		HttpPort:       cmd.Int("http-port"),
		PrometheusPort: cmd.Int("prometheus-port"),
		InstanceID:     cmd.String("instance-id"),

		// Database configuration
		DatabasePrimary: cmd.String("database-primary"),

		// Vault configuration
		VaultURL:   cmd.String("vault-url"),
		VaultToken: cmd.String("vault-token"),

		// Build configuration
		BuildBackend: worker.BuildBackend(cmd.String("build-backend")),
		BuildS3: worker.S3Config{
			URL:             cmd.String("build-s3-url"),
			Bucket:          cmd.String("build-s3-bucket"),
			AccessKeyID:     cmd.String("build-s3-access-key-id"),
			AccessKeySecret: cmd.String("build-s3-access-key-secret"),
			ExternalURL:     "",
		},
		BuildPlatform: cmd.String("build-platform"),

		// Registry configuration
		RegistryURL:      cmd.String("registry-url"),
		RegistryUsername: cmd.String("registry-username"),
		RegistryPassword: cmd.String("registry-password"),

		// Depot build backend configuration
		Depot: worker.DepotConfig{
			APIUrl:        cmd.String("depot-api-url"),
			ProjectRegion: cmd.String("depot-project-region"),
		},

		// Acme configuration
		Acme: worker.AcmeConfig{
			Enabled:     cmd.Bool("acme-enabled"),
			EmailDomain: cmd.String("acme-email-domain"),
			Route53: worker.Route53Config{
				Enabled:         cmd.Bool("acme-route53-enabled"),
				AccessKeyID:     cmd.String("acme-route53-access-key-id"),
				SecretAccessKey: cmd.String("acme-route53-secret-access-key"),
				Region:          cmd.String("acme-route53-region"),
				HostedZoneID:    cmd.String("acme-route53-hosted-zone-id"),
			},
		},

		DefaultDomain: cmd.String("default-domain"),

		// Restate configuration
		Restate: worker.RestateConfig{
			URL:        cmd.String("restate-url"),
			AdminURL:   cmd.String("restate-admin-url"),
			HttpPort:   cmd.Int("restate-http-port"),
			RegisterAs: cmd.String("restate-register-as"),
			APIKey:     cmd.String("restate-api-key"),
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

	return worker.Run(ctx, config)
}
