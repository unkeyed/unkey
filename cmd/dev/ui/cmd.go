package ui

import (
	"context"

	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd is the interactive dev TUI for humans. Agents and CI should call the
// headless subcommands (unkey dev seed, unkey dev stripe, mise run) instead.
var Cmd = &cli.Command{
	Name:  "ui",
	Usage: "Interactive terminal UI for local development (humans only; agents use headless dev subcommands)",
	Action: func(ctx context.Context, _ *cli.Command) error {
		return Run(ctx)
	},
}
