package dev

import (
	"github.com/unkeyed/unkey/go/cmd/dev/seed"
	"github.com/unkeyed/unkey/go/pkg/cli"
)

var Cmd = &cli.Command{
	Name:  "dev",
	Usage: "All of our development tools",
	Commands: []*cli.Command{
		seed.Cmd,
		// Future: apiRequestsCmd, ratelimitsCmd, etc.
	},
}
