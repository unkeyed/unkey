package ctrl

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/config"
	ctrlapi "github.com/unkeyed/unkey/svc/ctrl/api"
)

// apiCmd defines the "api" subcommand for running the control plane HTTP server.
// The server handles infrastructure management, build orchestration, and service
// coordination. Configuration is loaded from a TOML file specified by --config.
var apiCmd = &cli.Command{
	Version:     "",
	Commands:    []*cli.Command{},
	Aliases:     []string{},
	Description: "",
	Name:        "api",
	Usage:       "Run the Unkey control plane service for managing infrastructure and services",
	Flags: []cli.Flag{
		cli.String("config", "Path to a TOML config file",
			cli.Default("unkey.toml"), cli.EnvVar("UNKEY_CONFIG")),
	},
	Action: apiAction,
}

// apiAction loads configuration from a file and starts the control plane API server.
// It resolves TLS from file paths if configured and sets runtime-only fields
// before delegating to [ctrlapi.Run].
func apiAction(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load[ctrlapi.Config](cmd.String("config"))
	if err != nil {
		return cli.Exit("Failed to load config: "+err.Error(), 1)
	}

	return ctrlapi.Run(ctx, cfg)
}
