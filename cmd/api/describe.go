package api

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/unkeyed/unkey/cmd/api/internal/discovery"
	"github.com/unkeyed/unkey/pkg/cli"
)

// DescribeCmd returns offline, machine-readable discovery for canonical API commands.
func DescribeCmd() *cli.Command {
	return &cli.Command{
		Name:        "describe",
		Usage:       "Describe canonical API commands as JSON, without authentication or network access",
		Description: "List canonical API resource commands, or describe one command's flags and OpenAPI request schema. Excludes login and deploy orchestration.",
		Examples:    []string{"unkey describe", "unkey describe projects create", "unkey describe gateway policies update"},
		AcceptsArgs: true,
		OnError:     HandleError,
		Flags:       []cli.Flag{cli.Enum("error-format", "Error format on stderr", []string{"human", "json"}, cli.Default("human"))},
		Action: func(_ context.Context, command *cli.Command) error {
			commands := ResourceCommands()
			encoder := json.NewEncoder(os.Stdout)
			if len(command.Args()) == 0 {
				catalog, err := discovery.List(commands, resourcePaths)
				if err != nil {
					return err
				}
				return encoder.Encode(catalog)
			}
			contract, err := discovery.Describe(commands, resourcePaths, strings.Join(command.Args(), " "))
			if err != nil {
				return err
			}
			return encoder.Encode(contract)
		},
	}
}
