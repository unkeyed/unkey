package ctrl

import (
	"github.com/unkeyed/unkey/pkg/cli"
)

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
