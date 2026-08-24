//go:build !cli_release

package main

import (
	dev "github.com/unkeyed/unkey/cmd/dev"
	"github.com/unkeyed/unkey/pkg/cli"
)

func developmentCommands() []*cli.Command {
	return []*cli.Command{dev.Cmd}
}
