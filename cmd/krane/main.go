package krane

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane"
)

var Cmd = &cli.Command{
	Version:  "",
	Aliases:  []string{},
	Commands: []*cli.Command{},
	Name:     "krane",
	Usage:    "Run the k8s management service",
	Description: `krane (/kreɪn/) is the kubernetes deployment service for Unkey infrastructure.
It manages the lifecycle of deployments in a kubernetes cluster:

EXAMPLES:
unkey run krane                                   # Run with default configuration`,
	Flags: []cli.Flag{
		// Server Configuration
		cli.String("control-plane-url",
			"URL of the control plane to connect to",
			cli.Default("https://control.unkey.cloud"),
			cli.EnvVar("UNKEY_CONTROL_PLANE_URL"),
		),
		cli.String("control-plane-bearer",
			"Bearer token for authenticating with the control plane",
			cli.Default(""),
			cli.EnvVar("UNKEY_CONTROL_PLANE_BEARER"),
		),

		// Instance Identification
		cli.String("instance-id",
			"Unique identifier for this instance. Auto-generated if not provided.",
			cli.Default(uid.New(uid.InstancePrefix, 4)),
			cli.EnvVar("UNKEY_INSTANCE_ID"),
		),
		cli.String("region",
			"The cloud region with platform, e.g. aws:us-east-1",
			cli.Required(),
			cli.EnvVar("UNKEY_REGION"),
		),

		cli.String("registry-url",
			"URL of the container registry for pulling images. Example: registry.depot.dev",
			cli.EnvVar("UNKEY_REGISTRY_URL"),
		),

		cli.String("registry-username",
			"Username for authenticating with the container registry.",
			cli.EnvVar("UNKEY_REGISTRY_USERNAME"),
		),

		cli.String("registry-password",
			"Password/token for authenticating with the container registry.",
			cli.EnvVar("UNKEY_REGISTRY_PASSWORD"),
		),

		cli.Int("prometheus-port",
			"Port for Prometheus metrics, set to 0 to disable.",
			cli.Default(0),
			cli.EnvVar("UNKEY_PROMETHEUS_PORT")),

		cli.Int("rpc-port",
			"Port for RPC server",
			cli.Default(8070),
			cli.EnvVar("UNKEY_RPC_PORT")),

		// Vault Configuration
		cli.StringSlice("vault-master-keys", "Master keys for vault encryption (base64 encoded)",
			cli.EnvVar("UNKEY_VAULT_MASTER_KEYS")),
		cli.String("vault-s3-url", "S3 URL for vault storage",
			cli.EnvVar("UNKEY_VAULT_S3_URL")),
		cli.String("vault-s3-bucket", "S3 bucket for vault storage",
			cli.EnvVar("UNKEY_VAULT_S3_BUCKET")),
		cli.String("vault-s3-access-key-id", "S3 access key ID for vault storage",
			cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_ID")),
		cli.String("vault-s3-access-key-secret", "S3 access key secret for vault storage",
			cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_SECRET")),

		cli.String("cluster-id", "ID of the cluster",
			cli.Default(uid.Nano("")),
			cli.EnvVar("UNKEY_CLUSTER_ID")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {

	config := krane.Config{
		Clock:            nil,
		Region:           cmd.RequireString("region"),
		InstanceID:       cmd.RequireString("instance-id"),
		RegistryURL:      cmd.RequireString("registry-url"),
		RegistryUsername: cmd.RequireString("registry-username"),
		RegistryPassword: cmd.RequireString("registry-password"),
		RPCPort:          cmd.RequireInt("rpc-port"),
		VaultMasterKeys:  cmd.RequireStringSlice("vault-master-keys"),
		VaultS3: krane.S3Config{
			URL:             cmd.RequireString("vault-s3-url"),
			Bucket:          cmd.RequireString("vault-s3-bucket"),
			AccessKeyID:     cmd.RequireString("vault-s3-access-key-id"),
			AccessKeySecret: cmd.RequireString("vault-s3-access-key-secret"),
		},
		PrometheusPort:     cmd.RequireInt("prometheus-port"),
		ControlPlaneURL:    cmd.RequireString("control-plane-url"),
		ControlPlaneBearer: cmd.RequireString("control-plane-bearer"),
		ClusterID:          cmd.RequireString("cluster-id"),
	}

	// Validate configuration
	err := config.Validate()
	if err != nil {
		return cli.Exit("Invalid configuration: "+err.Error(), 1)
	}

	// Run krane
	return krane.Run(ctx, config)
}
