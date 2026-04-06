package run

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/cmd/run/api"
	"github.com/unkeyed/unkey/cmd/run/ctrl"
	"github.com/unkeyed/unkey/cmd/run/frontline"
	"github.com/unkeyed/unkey/cmd/run/heimdall"
	"github.com/unkeyed/unkey/cmd/run/krane"
	"github.com/unkeyed/unkey/cmd/run/sentinel"
	"github.com/unkeyed/unkey/cmd/run/vault"
	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd is the run command that serves as the parent command for running Unkey services.
// Use 'unkey run <service>' to start a specific service such as api, ctrl, krane, or vault.
var Cmd = &cli.Command{
	Flags:   []cli.Flag{},
	Version: "",
	Aliases: []string{},
	Name:    "run",
	Usage:   "Run Unkey services",
	Description: `Run various Unkey services in development or production environments.

This command starts different Unkey microservices. Each service can be configured independently and runs as a standalone process.

AVAILABLE SERVICES:
- api: The main API server for validating and managing API keys
- ctrl: The control plane service for managing infrastructure and deployments
- heimdall: Per-node metering DaemonSet for cgroup + eBPF resource accounting
- krane: Deployment management service for Kubernetes
- frontline: Multi-tenant frontline service for TLS termination and routing
- sentinel: Environment tenant sentinel service for routing requests to the actual instances
- vault: Secret management service for encryption

EXAMPLES:
unkey run api                                    # Run the API server
unkey run frontline                                # Run the frontline service
unkey run sentinel                                # Run the tenant sentinel service
unkey run --help                                 # Show available services and their options
unkey run api --port 8080 --env production      # Run API server with custom configuration`,
	Commands: []*cli.Command{
		api.Cmd,
		ctrl.Cmd,
		heimdall.Cmd,
		krane.Cmd,
		frontline.Cmd,
		sentinel.Cmd,
		vault.Cmd,
	},
	Action: runAction,
}

func runAction(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("Available services:")
	fmt.Println("  api             - The main API server for validating and managing API keys")
	fmt.Println("  ctrl            - The control plane service for managing infrastructure")
	fmt.Println("  heimdall        - Per-node metering DaemonSet for cgroup + eBPF resource accounting")
	fmt.Println("  krane           - Manage containers and deployments in docker or kubernetes")
	fmt.Println("  frontline       - Multi-tenant ingress service for TLS termination and routing")
	fmt.Println("  sentinel        - Environment tenant gateway service for routing requests to the actual instances")
	fmt.Println("  vault           - Encryption service for sensitive data")
	fmt.Println()
	fmt.Println("Use 'unkey run <service>' to start a specific service")
	fmt.Println("Use 'unkey run <service> --help' for service-specific options")
	return nil
}
