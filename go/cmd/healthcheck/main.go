package healthcheck

import (
	"context"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/go/pkg/cli"
)

var Cmd = &cli.Command{
	Name:  "healthcheck",
	Usage: "Perform an HTTP healthcheck against a given URL",
	Description: `Perform an HTTP healthcheck against a given URL.

This command exits with 0 if the status code is 200, otherwise it exits with 1.

## Usage

` + "`unkey healthcheck <url>`" + `

## Examples

Check if a service is healthy:
` + "`unkey healthcheck https://api.unkey.dev/health`" + `

Check local service:
` + "`unkey healthcheck http://localhost:8080/health`",
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

	res, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to perform healthcheck: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck failed with status code %d", res.StatusCode)
	}

	fmt.Printf("✓ Healthcheck passed: %s returned %d\n", url, res.StatusCode)
	return nil
}
