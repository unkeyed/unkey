package main

import (
	"context"
	"fmt"
	"os"

	"github.com/unkeyed/unkey/cmd/api"
	"github.com/unkeyed/unkey/cmd/auth"
	"github.com/unkeyed/unkey/cmd/deploy"
	dev "github.com/unkeyed/unkey/cmd/dev"
	"github.com/unkeyed/unkey/cmd/healthcheck"
	"github.com/unkeyed/unkey/cmd/preflight"
	"github.com/unkeyed/unkey/cmd/run"
	"github.com/unkeyed/unkey/cmd/version"
	"github.com/unkeyed/unkey/pkg/buildinfo"
	"github.com/unkeyed/unkey/pkg/cli"
)

func main() {
	app := &cli.Command{
		Flags:       []cli.Flag{},
		Aliases:     []string{},
		Action:      nil,
		Name:        "unkey",
		Usage:       "Run unkey",
		Description: `Unkey CLI – deploy, run and administer Unkey services.`,
		Version:     buildinfo.Version,
		Commands: []*cli.Command{
			api.Cmd(),
			auth.Cmd,
			run.Cmd,
			version.Cmd,
			deploy.Cmd,
			healthcheck.Cmd,
			dev.Cmd,
			preflight.Cmd,
		},
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
