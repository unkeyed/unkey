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
	Usage:       "Check an HTTP endpoint",
	Description: "Send an HTTP GET request. Exit with status 0 for HTTP 200, or status 1 otherwise.",
	Examples:    []string{"unkey healthcheck https://api.unkey.com/v2/liveness", "unkey healthcheck http://localhost:8080/health"},
	Action:      runAction,
}

func runAction(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args()
	if len(args) != 1 || args[0] == "" {
		return fmt.Errorf("usage: unkey healthcheck <url>")
	}

	url := args[0]
	res, err := http.Get(url) //nolint: gosec
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}
	if err := res.Body.Close(); err != nil {
		return fmt.Errorf("failed to close response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck failed: HTTP %d from %s", res.StatusCode, url)
	}

	_, err = fmt.Printf("Healthy: HTTP %d from %s\n", res.StatusCode, url)
	return err
}
