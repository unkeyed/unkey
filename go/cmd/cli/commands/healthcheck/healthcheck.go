package healthcheck

import (
	"context"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/go/cmd/cli/cli"
)

var Command = &cli.Command{
	Name:  "healthcheck",
	Usage: "Perform HTTP healthcheck against a URL",
	Description: `Perform an HTTP healthcheck against a given URL.
This command exits with 0 if the status code is 200, otherwise it exits with 1.

EXAMPLES:
    # Check if API is healthy
    unkey healthcheck https://api.example.com/health
    
    # Check local service
    unkey healthcheck http://localhost:8080/health`,
	Action: run,
}

func run(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args()
	if len(args) == 0 {
		return fmt.Errorf("you must provide a URL like so: 'unkey healthcheck <url>'")
	}

	url := args[0]

	// nolint:gosec
	res, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to perform healthcheck: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck failed with status code %d", res.StatusCode)
	}

	fmt.Printf("✓ Healthcheck passed for %s (status: %d)\n", url, res.StatusCode)
	return nil
}
