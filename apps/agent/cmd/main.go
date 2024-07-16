package main

import (
	"fmt"
	"os"

	"github.com/Southclaws/fault"
	"github.com/unkeyed/unkey/apps/agent/cmd/agent"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "unkey",
		Usage: "Run unkey agents",

		Commands: []*cli.Command{
			agent.Cmd,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		chain := fault.Flatten(err)

		fmt.Println()
		fmt.Println()

		fmt.Println(chain)
		fmt.Println()
		os.Exit(1)
	}
}
