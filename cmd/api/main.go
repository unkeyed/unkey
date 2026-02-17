package api

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/tls"
	"github.com/unkeyed/unkey/svc/api"
)

// Cmd is the api command that runs the Unkey API server for validating and managing
// API keys, rate limiting, and analytics.
var Cmd = &cli.Command{
	Aliases:     []string{},
	Description: "",
	Version:     "",
	Commands:    []*cli.Command{},
	Name:        "api",
	Usage:       "Run the Unkey API server for validating and managing API keys",

	Flags: []cli.Flag{
		cli.String("config", "Path to a TOML config file",
			cli.Default("unkey.toml"), cli.EnvVar("UNKEY_CONFIG")),
	},

	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load[api.Config](cmd.String("config"))
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)

	}

	// Resolve TLS config from file paths
	if cfg.TLS.CertFile != "" {
		tlsCfg, tlsErr := tls.NewFromFiles(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		if tlsErr != nil {
			return cli.Exit("Failed to load TLS configuration: "+tlsErr.Error(), 1)
		}
		cfg.TLSConfig = tlsCfg
	}

	cfg.Clock = clock.New()

	return api.Run(ctx, cfg)
}
