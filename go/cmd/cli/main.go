package main

import (
	"context"
	"fmt"
	"os"

	"github.com/unkeyed/unkey/go/cmd/cli/cli"
	"github.com/unkeyed/unkey/go/cmd/cli/commands/deploy"
	initcmd "github.com/unkeyed/unkey/go/cmd/cli/commands/init"
	"github.com/unkeyed/unkey/go/cmd/cli/commands/versions"
	"github.com/unkeyed/unkey/go/pkg/version"
)

func main() {
	app := &cli.Command{
		Name:    "unkey",
		Usage:   "Deploy and manage your API versions",
		Version: version.Version,
		Commands: []*cli.Command{
			deploy.Command,
			versions.Command,
			initcmd.Command,
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
