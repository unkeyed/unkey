package run

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/cmd/api"
	"github.com/unkeyed/unkey/go/cmd/ctrl"
	"github.com/unkeyed/unkey/go/cmd/gw"
	"github.com/unkeyed/unkey/go/pkg/cli"
)

var Cmd = &cli.Command{
	Name:  "run",
	Usage: "Run Unkey services",
	Description: `Run various Unkey services including:
  - api: The main API server for validating and managing API keys
  - ctrl: The control plane service for managing infrastructure
  - gw: The gateway service

EXAMPLES:
    # Run the API server
    unkey run api

    # Run the control plane
    unkey run ctrl

    # Run the gateway
    unkey run gw

    # Show available services
    unkey run --help`,
	Commands: []*cli.Command{
		api.Cmd,
		ctrl.Cmd,
		gw.Cmd,
	},
	Action: runAction,
}

func runAction(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("Available services:")
	fmt.Println("  api   - The main API server for validating and managing API keys")
	fmt.Println("  ctrl  - The control plane service for managing infrastructure")
	fmt.Println("  gw    - The gateway service")
	fmt.Println()
	fmt.Println("Use 'unkey run <service>' to start a specific service")
	fmt.Println("Use 'unkey run <service> --help' for service-specific options")
	return nil
}
