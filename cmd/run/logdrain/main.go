package logdrain

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/svc/logdrain"
)

// Cmd runs the logdrain service that forwards customer logs from
// ClickHouse to third-party providers like Axiom.
var Cmd = &cli.Command{
	Version:  "",
	Aliases:  []string{},
	Commands: []*cli.Command{},
	Name:     "logdrain",
	Usage:    "Run the log drain forwarder",
	Description: `logdrain forwards customer logs from ClickHouse to third-party providers.
It tails runtime_logs_raw_v1 and sentinel_requests_raw_v1 per drain group,
applies filters, and pushes batches with retry and backoff.

EXAMPLES:
  unkey run logdrain --config /etc/unkey/logdrain.toml`,
	Flags: []cli.Flag{
		cli.String("config", "Path to a TOML config file",
			cli.Default("unkey.toml"), cli.EnvVar("UNKEY_CONFIG")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load[logdrain.Config](cmd.String("config"))
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	return logdrain.Run(ctx, cfg)
}
