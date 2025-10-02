package main

import (
	"context"
	"fmt"
	"os"

	createiceberg "github.com/unkeyed/unkey/go/cmd/create-iceberg"
	"github.com/unkeyed/unkey/go/cmd/deploy"
	gateway "github.com/unkeyed/unkey/go/cmd/gw"
	"github.com/unkeyed/unkey/go/cmd/healthcheck"
	"github.com/unkeyed/unkey/go/cmd/konsume"
	queryiceberg "github.com/unkeyed/unkey/go/cmd/query-iceberg"
	"github.com/unkeyed/unkey/go/cmd/quotacheck"
	"github.com/unkeyed/unkey/go/cmd/run"
	"github.com/unkeyed/unkey/go/cmd/version"
	"github.com/unkeyed/unkey/go/pkg/cli"
	versioncmd "github.com/unkeyed/unkey/go/pkg/version"
)

func main() {
	app := &cli.Command{
		Name:        "unkey",
		Usage:       "Run unkey",
		Description: `Unkey CLI – deploy, run and administer Unkey services.`,
		Version:     versioncmd.Version,
		Commands: []*cli.Command{
			run.Cmd,
			version.Cmd,
			deploy.Cmd,
			healthcheck.Cmd,
			quotacheck.Cmd,
			gateway.Cmd,
			konsume.Cmd,
			createiceberg.Cmd,
			queryiceberg.Cmd,
		},
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
