package krane

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/svc/krane"
)

// Cmd is the krane command that runs the Kubernetes deployment service for managing
// container lifecycles and deployments in a Kubernetes cluster.
var Cmd = &cli.Command{
	Version:  "",
	Aliases:  []string{},
	Commands: []*cli.Command{},
	Name:     "krane",
	Usage:    "Run the k8s management service",
	Description: `krane (/kreɪn/) is the kubernetes deployment service for Unkey infrastructure.
It manages the lifecycle of deployments in a kubernetes cluster:

EXAMPLES:
  unkey run krane --config /etc/unkey/krane.toml`,
	Flags: []cli.Flag{
		cli.String("config", "Path to a TOML config file",
			cli.Default("unkey.toml"), cli.EnvVar("UNKEY_CONFIG")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load[krane.Config](cmd.String("config"))
	if err != nil {
		return fmt.Errorf("unable to load config: %w")
	}

	cfg.Clock = clock.New()

	return krane.Run(ctx, cfg)
}
