// Package util contains shared command wiring for release service binaries.
package util

import (
	"context"
	"fmt"
	"os"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/config"
)

// RunServiceCommand creates and runs a service command with the standard --config flag.
// It loads the service config type from TOML before delegating runtime setup to
// the service's Run function.
func RunServiceCommand[T any](name string, usage string, run func(context.Context, T) error) {
	cmd := &cli.Command{
		Name:  name,
		Usage: usage,
		Flags: []cli.Flag{
			cli.String("config", "Path to a TOML config file",
				cli.Default("unkey.toml"), cli.EnvVar("UNKEY_CONFIG")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg, err := config.Load[T](cmd.String("config"))
			if err != nil {
				return fmt.Errorf("unable to load config: %w", err)
			}
			return run(ctx, cfg)
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
