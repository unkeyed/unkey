package main

import (
	"context"
	"fmt"
	"os"

	"github.com/unkeyed/unkey/go/cmd/api"
	"github.com/unkeyed/unkey/go/cmd/dev"
	"github.com/unkeyed/unkey/go/cmd/healthcheck"
	"github.com/unkeyed/unkey/go/cmd/quotacheck"
	"github.com/unkeyed/unkey/go/pkg/version"
	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:    "unkey",
		Usage:   "Run unkey ",
		Version: version.Version,

		Commands: []*cli.Command{
			api.Cmd,
			healthcheck.Cmd,
			quotacheck.Cmd,
			dev.Cmd,
		},
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Println()
		fmt.Println()
		fmt.Println(err.Error())
		fmt.Println()
		os.Exit(1)
	}
}
