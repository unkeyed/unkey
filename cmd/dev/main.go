package dev

import (
	"github.com/unkeyed/unkey/cmd/dev/seed"
	"github.com/unkeyed/unkey/pkg/cli"
)

var Cmd = &cli.Command{
	Name:  "dev",
	Usage: "All of our development tools",
	Commands: []*cli.Command{
		seed.Cmd,
		// Future: apiRequestsCmd, ratelimitsCmd, etc.
	},
}
