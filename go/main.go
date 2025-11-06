package main

import (
	"context"
	"fmt"
	"os"

	clickhouseUser "github.com/unkeyed/unkey/go/cmd/create-clickhouse-user"
	"github.com/unkeyed/unkey/go/cmd/deploy"
	dev "github.com/unkeyed/unkey/go/cmd/dev"
	"github.com/unkeyed/unkey/go/cmd/healthcheck"
	"github.com/unkeyed/unkey/go/cmd/quotacheck"
	"github.com/unkeyed/unkey/go/cmd/run"
	"github.com/unkeyed/unkey/go/cmd/version"
	"github.com/unkeyed/unkey/go/pkg/cli"
	versioncmd "github.com/unkeyed/unkey/go/pkg/version"
)

func main() {
	app := &cli.Command{
		Flags:       []cli.Flag{},
		Aliases:     []string{},
		Action:      nil,
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
			clickhouseUser.Cmd,
			dev.Cmd,
		},
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
