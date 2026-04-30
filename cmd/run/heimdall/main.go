package heimdall

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/svc/heimdall"
)

// Cmd is the heimdall command that runs the resource metering agent.
var Cmd = &cli.Command{
	Version:  "",
	Aliases:  []string{},
	Commands: []*cli.Command{},
	Name:     "heimdall",
	Usage:    "Run the resource metering agent",
	Description: `heimdall is the resource usage metering agent for Unkey infrastructure.
It collects per-deployment CPU, memory, and network egress metrics from kubelet
and writes them to ClickHouse for billing.

EXAMPLES:
  unkey run heimdall --config /etc/unkey/heimdall.toml`,
	Flags: []cli.Flag{
		cli.String("config", "Path to a TOML config file",
			cli.Default("unkey.toml"), cli.EnvVar("UNKEY_CONFIG")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load[heimdall.Config](cmd.String("config"))
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	return heimdall.Run(ctx, cfg)
}
