package krane

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/krane"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/uid"
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
		cli.String("shard",
			"Shard identifier for this cluster. Used to allow us to scale beyond one cluster per region.",
			cli.Default("default"),
			cli.EnvVar("UNKEY_SHARD"),
		),
		cli.String("image",
			"The docker image running",
			cli.Required(),
			cli.EnvVar("UNKEY_IMAGE"),
		),

		cli.String("backend",
			"Backend type for the service. Either kubernetes or docker. Default: kubernetes",
			cli.Default("kubernetes"),
			cli.EnvVar("UNKEY_KRANE_BACKEND"),
		),

		cli.String("docker-socket",
			"Path to the docker socket. Only used if backend is docker. Default: /var/run/docker.sock",
			cli.Default("/var/run/docker.sock"),
			cli.EnvVar("UNKEY_DOCKER_SOCKET"),
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
			cli.Default(9090),
			cli.EnvVar("UNKEY_PROMETHEUS_PORT")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {

	config := krane.Config{
		Clock:              nil,
		ControlPlaneURL:    cmd.String("control-plane-url"),
		ControlPlaneBearer: cmd.String("control-plane-bearer"),
		Image:              cmd.RequireString("image"),
		Region:             cmd.String("region"),
		Shard:              cmd.String("shard"),
		InstanceID:         cmd.String("instance-id"),
		RegistryURL:        cmd.String("registry-url"),
		RegistryUsername:   cmd.String("registry-username"),
		RegistryPassword:   cmd.String("registry-password"),
		PrometheusPort:     cmd.Int("prometheus-port"),
	}

	// Validate configuration
	err := config.Validate()
	if err != nil {
		return cli.Exit("Invalid configuration: "+err.Error(), 1)
	}

	// Run krane
	return krane.Run(ctx, config)
}
