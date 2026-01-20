package healthcheck

import (
	"context"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/pkg/cli"
)

// Cmd is the healthcheck command that performs an HTTP GET request to a given URL
// and exits with code 0 if the server returns 200, otherwise exits with code 1.
var Cmd = &cli.Command{
	Aliases:     []string{},
	Version:     "",
	Commands:    []*cli.Command{},
	Flags:       []cli.Flag{},
	AcceptsArgs: true,
	Name:        "healthcheck",
	Usage:       "Perform an HTTP healthcheck against a given URL",
	Description: `This command sends an HTTP GET request to the specified URL and validates the response. It exits with code 0 if the server returns a 200 status code, otherwise exits with code 1.

USE CASES:
This is useful for health monitoring in CI/CD pipelines, service availability checks, load balancer health probes, and infrastructure monitoring scripts.

EXAMPLES:
unkey healthcheck https://api.unkey.dev/health    # Check if a service is healthy
unkey healthcheck http://localhost:8080/health    # Check local service
unkey healthcheck https://example.com/api/status || echo 'Service is down!'  # Use in monitoring script`,
	Action: runAction,
}

func runAction(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args()
	if len(args) == 0 {
		return fmt.Errorf("you must provide a url like so: 'unkey healthcheck <url>'")
	}

	url := args[0]
	if url == "" {
		return fmt.Errorf("you must provide a url like so: 'unkey healthcheck <url>'")
	}

	res, err := http.Get(url) //nolint: gosec
	if err != nil {
		return fmt.Errorf("failed to perform healthcheck: %w", err)
	}
	if err := res.Body.Close(); err != nil {
		return fmt.Errorf("failed to close response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck failed with status code %d", res.StatusCode)
	}

	fmt.Printf("✓ Healthcheck passed: %s returned %d\n", url, res.StatusCode)
	return nil
}
