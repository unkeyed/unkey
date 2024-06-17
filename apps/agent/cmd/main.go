package main

import (
	"fmt"
	"os"

	"github.com/unkeyed/unkey/apps/agent/cmd/agent"
	"github.com/unkeyed/unkey/apps/agent/cmd/agent/deploy"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "unkey",
		Usage: "Run unkey agents",

		Commands: []*cli.Command{
			agent.Cmd,
			deploy.Cmd,
		},
	}

	fmt.Println(os.Environ())
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println()
		fmt.Println()
		fmt.Println(err)
		fmt.Println()
		os.Exit(1)
	}
}
