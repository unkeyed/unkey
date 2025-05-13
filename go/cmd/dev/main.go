package dev

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name:        "dev",
	Description: "Develop unkey",
	Action:      run,
	Commands: []*cli.Command{
		{
			Category: "migration",
			Name:     "migrate-clickhouse",
			Usage:    "migrate the clickhouse schema",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "url",
					Usage: "clickhouse url",
					Value: "clickhouse://default:password@localhost:9000?secure=false&skip_verify=true&dial_timeout=10s",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {

				return migrateClickhouse(ctx, cmd.String("url"))
			},
		},
		{
			Category: "migration",
			Name:     "migrate-mysql",
			Usage:    "migrate the mysql schema",
			Action: func(ctx context.Context, cmd *cli.Command) error {

				fmt.Println("I'm migrating mysql hard")
				return nil
			},
		},
		{
			Category: "seed",
			Name:     "seed-mysql",
			Usage:    "seed the mysql schema",
			Action: func(ctx context.Context, cmd *cli.Command) error {

				fmt.Println("I'm seeding mysql hard")
				return nil
			},
		},
		{
			Category: "seed",
			Name:     "seed-mysql",
			Usage:    "seed the mysql schema",
			Action: func(ctx context.Context, cmd *cli.Command) error {

				fmt.Println("I'm seeding mysql hard")
				return nil
			},
		},
	},
}

// nolint:gocognit
func run(ctx context.Context, cmd *cli.Command) error {

	fmt.Println("dev")

	return nil
}
