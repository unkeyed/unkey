package dev

import (
	"fmt"
	"os/exec"

	"github.com/Southclaws/fault"
	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
	Name: "dev",
	Args: true,
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:    "port",
			Aliases: []string{"p"},
			Usage:   "Run the local development server on a specific port",
			Value:   4000,

			DefaultText: "4000",
		},
	},
	Action: run,
}

func run(c *cli.Context) error {

	entrypoint := c.Args().First()
	wrangler := exec.Command("wrangler", "dev", entrypoint, fmt.Sprintf("--port=%d", c.Int("port")))

	output, err := wrangler.CombinedOutput()
	if err != nil {
		fmt.Println(err.Error())
		return fault.Wrap(err)
	}
	fmt.Println(output)
	return nil
}
