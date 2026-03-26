package ctrl

import (
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd is the root command for the Unkey control plane. It provides subcommands
// for running the API server (api) and background worker (worker). Use this
// command as a subcommand of the main Unkey CLI binary.
var Cmd = &cli.Command{
	Version: "",
	Commands: []*cli.Command{
		apiCmd,
		workerCmd,
	},
	Aliases:     []string{},
	Description: "",
	Name:        "ctrl",
	Usage:       "Run the Unkey control plane service for managing infrastructure and services",
	Flags:       []cli.Flag{},
	Action:      nil,
	AcceptsArgs: false,
}
