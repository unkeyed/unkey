package run

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/cmd/api"
	"github.com/unkeyed/unkey/go/cmd/ctrl"
	"github.com/unkeyed/unkey/go/pkg/cli"
)

var Cmd = &cli.Command{
	Name:  "run",
	Usage: "Run Unkey services",
	Description: `Run various Unkey services in development or production environments.

## Available Services

- ` + "`api`" + ` - The main API server for validating and managing API keys
- ` + "`ctrl`" + ` - The control plane service for managing infrastructure and deployments

## Usage

` + "`unkey run <service> [flags]`" + `

## Examples

Run the API server:
` + "`unkey run api`" + `

Run the control plane:
` + "`unkey run ctrl`" + `

Show available services and their options:
` + "`unkey run --help`" + `

Run API server with custom configuration:
` + "`unkey run api --port 8080 --env production`",
	Commands: []*cli.Command{
		api.Cmd,
		ctrl.Cmd,
	},
	Action: runAction,
}

func runAction(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("Available services:")
	fmt.Println("  api   - The main API server for validating and managing API keys")
	fmt.Println("  ctrl  - The control plane service for managing infrastructure")
	fmt.Println()
	fmt.Println("Use 'unkey run <service>' to start a specific service")
	fmt.Println("Use 'unkey run <service> --help' for service-specific options")
	return nil
}
