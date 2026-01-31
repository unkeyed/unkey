package main

import (
	"context"
	"fmt"
	"os"

	billingjob "github.com/unkeyed/unkey/cmd/billing-job"
	"github.com/unkeyed/unkey/cmd/deploy"
	dev "github.com/unkeyed/unkey/cmd/dev"
	"github.com/unkeyed/unkey/cmd/frontline"
	"github.com/unkeyed/unkey/cmd/healthcheck"
	"github.com/unkeyed/unkey/cmd/quotacheck"
	"github.com/unkeyed/unkey/cmd/run"
	"github.com/unkeyed/unkey/cmd/version"
	"github.com/unkeyed/unkey/pkg/cli"
	versioncmd "github.com/unkeyed/unkey/pkg/version"
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
			frontline.Cmd,
			dev.Cmd,
			billingjob.Cmd,
		},
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
