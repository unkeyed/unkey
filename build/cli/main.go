package main

import (
	"context"
	"fmt"
	"os"

	"github.com/unkeyed/unkey/cmd/api"
	"github.com/unkeyed/unkey/cmd/auth"
	"github.com/unkeyed/unkey/cmd/deploy"
	"github.com/unkeyed/unkey/cmd/healthcheck"
	"github.com/unkeyed/unkey/cmd/version"
	"github.com/unkeyed/unkey/cmd/workspace"
	"github.com/unkeyed/unkey/pkg/buildinfo"
	"github.com/unkeyed/unkey/pkg/cli"
)

func main() {
	resources := api.ResourceCommands()
	for _, command := range resources {
		if command.Name == "workspace" {
			command.Commands = append(command.Commands, workspace.Cmd.Commands...)
		}
	}

	app := &cli.Command{
		Flags:       []cli.Flag{},
		Aliases:     []string{},
		Action:      nil,
		Name:        "unkey",
		Usage:       "Manage your Unkey workspace",
		Description: "Sign in with 'unkey login', then manage projects, apps, keys, and deployments.\nUse --json for scripts and --help for command options.",
		Examples:    []string{"unkey login", "unkey projects list", "unkey keys list --api-id=api_123 --json"},
		Version:     buildinfo.Version,
		Commands: append(append(resources,
			api.Cmd(),
			auth.Cmd,
			version.Cmd,
			deploy.Cmd,
			healthcheck.Cmd,
		), developmentCommands()...),
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
