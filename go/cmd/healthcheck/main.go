package healthcheck

import (
	"context"
	"fmt"
	"net/http"

	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name: "healthcheck",
	Description: `Perform an HTTP healthcheck against a given URL.

This command exits with 0 if the status code is 200, otherwise it exits with 1.
	`,
	ArgsUsage: `<url>`,
	Action:    run,
}

// nolint:gocognit
func run(ctx context.Context, cmd *cli.Command) error {

	url := cmd.Args().First()
	if url == "" {
		return fmt.Errorf("You must provide a url like so: 'unkey healthcheck <url>'")
	}

	// nolint:gosec
	res, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to perform healthcheck: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck failed with status code %d", res.StatusCode)
	}

	return nil
}
