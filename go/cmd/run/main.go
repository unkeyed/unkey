package run

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/cmd/api"
	"github.com/unkeyed/unkey/go/cmd/ctrl"
	"github.com/unkeyed/unkey/go/cmd/gw"
	"github.com/unkeyed/unkey/go/cmd/metald"
	"github.com/unkeyed/unkey/go/pkg/cli"
)

var Cmd = &cli.Command{
	Name:  "run",
	Usage: "Run Unkey services",
	Description: `Run various Unkey services in development or production environments.

This command starts different Unkey microservices. Each service can be configured independently and runs as a standalone process.

AVAILABLE SERVICES:
- api: The main API server for validating and managing API keys
- ctrl: The control plane service for managing infrastructure and deployments
- gw: The gateway service for routing requests to the appropriate API
- metald: The VM management service for infrastructure

EXAMPLES:
unkey run api                                    # Run the API server
unkey run ctrl                                   # Run the control plane
unkey run gw                                     # Run the gateway
unkey run metald                                 # Run the VM management service
unkey run --help                                 # Show available services and their options
unkey run api --port 8080 --env production      # Run API server with custom configuration`,
	Commands: []*cli.Command{
		api.Cmd,
		ctrl.Cmd,
		gw.Cmd,
		metald.Cmd,
	},
	Action: runAction,
}

func runAction(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("Available services:")
	fmt.Println("  api   - The main API server for validating and managing API keys")
	fmt.Println("  ctrl  - The control plane service for managing infrastructure")
	fmt.Println("  gw    - The gateway service")
	fmt.Println("  metald - The VM management service")
	fmt.Println()
	fmt.Println("Use 'unkey run <service>' to start a specific service")
	fmt.Println("Use 'unkey run <service> --help' for service-specific options")
	return nil
}
