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
	Description: `
This command sends an HTTP GET request to the specified URL and validates the response. 
It exits with code 0 if the server returns a 200 status code, otherwise exits with code 1.

This is useful for:
- Health monitoring in CI/CD pipelines
- Service availability checks
- Load balancer health probes
- Infrastructure monitoring scripts

## Usage

` + "`unkey healthcheck <url>`" + `

## Examples

Check if a service is healthy:
` + "`unkey healthcheck https://api.unkey.dev/health`" + `

Check local service:
` + "`unkey healthcheck http://localhost:8080/health`" + `

Use in a monitoring script:
` + "`unkey healthcheck https://example.com/api/status || echo 'Service is down!'`",
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
