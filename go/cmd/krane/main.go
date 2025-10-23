package krane

import (
	"context"
	"fmt"

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
		cli.Int("http-port", "Port for the server to listen on. Default: 8080",
			cli.Default(8080), cli.EnvVar("UNKEY_HTTP_PORT")),

		// Instance Identification
		cli.String("instance-id", "Unique identifier for this instance. Auto-generated if not provided.",
			cli.Default(uid.New(uid.InstancePrefix, 4)), cli.EnvVar("UNKEY_INSTANCE_ID")),

		cli.String("backend", "Backend type for the service. Either kubernetes or docker. Default: kubernetes",
			cli.Default("kubernetes"), cli.EnvVar("UNKEY_KRANE_BACKEND")),

		cli.String("docker-socket", "Path to the docker socket. Only used if backend is docker. Default: /var/run/docker.sock",
			cli.Default("/var/run/docker.sock"), cli.EnvVar("UNKEY_DOCKER_SOCKET")),

		// This has no use outside of our demo cluster and will be removed soon
		cli.Duration("deployment-eviction-ttl", "Automatically delete deployments after some time. Use go duration formats such as 2h30m", cli.EnvVar("UNKEY_DEPLOYMENT_EVICTION_TTL")),
	},

	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	backend, err := parseBackend(cmd.String("backend"))
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}
	config := krane.Config{
		Clock:                 nil,
		HttpPort:              cmd.Int("http-port"),
		Backend:               backend,
		Platform:              cmd.String("platform"),
		Image:                 cmd.String("image"),
		Region:                cmd.String("region"),
		OtelEnabled:           false,
		OtelTraceSamplingRate: 1.0,
		InstanceID:            cmd.String("instance-id"),
		DockerSocketPath:      cmd.String("docker-socket"),
		DeploymentEvictionTTL: cmd.Duration("deployment-eviction-ttl"),
	}

	// Validate configuration
	err = config.Validate()
	if err != nil {
		return cli.Exit("Invalid configuration: "+err.Error(), 1)
	}

	// Run krane
	return krane.Run(ctx, config)
}

func parseBackend(s string) (krane.Backend, error) {
	switch s {
	case "docker":
		return krane.Docker, nil
	case "kubernetes":

		return krane.Kubernetes, nil
	default:
		return "", fmt.Errorf("unknown backend type: %s", s)
	}
}
