package main

import (
	"context"
	"fmt"
	"os"

	"github.com/unkeyed/unkey/go/cmd/deploy"
	"github.com/unkeyed/unkey/go/cmd/healthcheck"
	initCmd "github.com/unkeyed/unkey/go/cmd/init"
	"github.com/unkeyed/unkey/go/cmd/quotacheck"
	"github.com/unkeyed/unkey/go/cmd/run"
	"github.com/unkeyed/unkey/go/cmd/version"
	"github.com/unkeyed/unkey/go/pkg/cli"
	ver "github.com/unkeyed/unkey/go/pkg/version"
)

func main() {
	app := &cli.Command{
		Name:    "unkey",
		Usage:   "Run unkey",
		Version: ver.Version,
		Commands: []*cli.Command{
			run.Cmd,
			version.Cmd,
			initCmd.Cmd,
			deploy.Cmd,
			healthcheck.Cmd,
			quotacheck.Cmd,
		},
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
