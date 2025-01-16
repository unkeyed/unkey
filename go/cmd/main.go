package main

import (
	"fmt"
	"os"

	"github.com/Southclaws/fault"
	"github.com/unkeyed/unkey/go/cmd/api"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "unkey",
		Usage: "Run unkey ",

		Commands: []*cli.Command{
			api.Cmd,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		chain := fault.Flatten(err)

		fmt.Println()
		fmt.Println()

		for _, e := range chain {
			fmt.Printf(" - ")
			if e.Location != "" {
				fmt.Printf("%s\n", e.Location)
				fmt.Printf("    > ")
			}
			fmt.Printf("%s\n", e.Message)
		}
		fmt.Println()
		os.Exit(1)
	}
}
