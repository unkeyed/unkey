package dev

import (
	"github.com/unkeyed/unkey/cmd/dev/seed"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd is the dev command that provides development tools and utilities for local testing,
// including database seeding and other development workflows.
var Cmd = &cli.Command{
	Name:  "dev",
	Usage: "All of our development tools",
	Commands: []*cli.Command{
		seed.Cmd,
		generateMasterKeyCmd,
	},
}
