package main

import (
	"fmt"
	"os"

	"github.com/unkeyed/unkey/go/cmd/api"
	"github.com/unkeyed/unkey/go/cmd/healthcheck"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "unkey",
		Usage: "Run unkey ",

		Commands: []*cli.Command{
			api.Cmd,
			healthcheck.Cmd,
		},
	}

	err := app.Run(os.Args)
	if err != nil {

		fmt.Println()
		fmt.Println()

		fmt.Println(err.Error())
		fmt.Println()
		os.Exit(1)
	}
}
