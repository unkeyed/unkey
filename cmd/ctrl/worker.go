package ctrl

import (
	"context"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/svc/ctrl/worker"
)

// workerCmd defines the "worker" subcommand for running the background job
// processor. The worker handles durable workflows via Restate including container
// builds, deployments, and ACME certificate provisioning. Configuration is loaded
// from a TOML file specified by --config.
var workerCmd = &cli.Command{
	Version:     "",
	Commands:    []*cli.Command{},
	Aliases:     []string{},
	Description: "",
	Name:        "worker",
	Usage:       "Run the Unkey Restate worker service for background jobs and workflows",
	Flags: []cli.Flag{
		cli.String("config", "Path to a TOML config file",
			cli.Default("unkey.toml"), cli.EnvVar("UNKEY_CONFIG")),
	},
	Action: workerAction,
}

// workerAction loads configuration from a file and starts the background worker
// service. It sets runtime-only fields before delegating to [worker.Run].
func workerAction(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load[worker.Config](cmd.String("config"))
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	// Normalize CNAME domain: trim whitespace and trailing dot
	cfg.CnameDomain = strings.TrimSuffix(strings.TrimSpace(cfg.CnameDomain), ".")

	cfg.Clock = clock.New()

	return worker.Run(ctx, cfg)
}
