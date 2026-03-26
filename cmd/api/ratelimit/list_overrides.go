package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func listOverridesCmd() *cli.Command {
	return &cli.Command{
		Name:  "list-overrides",
		Usage: "Retrieve a paginated list of all rate limit overrides in a namespace",
		Description: `Retrieve a paginated list of all rate limit overrides in a namespace.

Use this to audit rate limiting policies, build admin dashboards, or manage override configurations.

Important: Results are paginated. Use the cursor parameter to retrieve additional pages when more results are available.

Required permissions:
- ratelimit.*.read_override
- ratelimit.<namespace_id>.read_override

For full documentation, see https://www.unkey.com/docs/api-reference/v2/ratelimit/list-ratelimit-overrides` + util.Disclaimer,
		Examples: []string{
			"unkey api ratelimit list-overrides --namespace=api.requests",
			"unkey api ratelimit list-overrides --namespace=api.requests --limit=50",
			"unkey api ratelimit list-overrides --namespace=api.requests --cursor=cursor_eyJsYXN0SWQiOiJvdnJfM2RITGNOeVN6SnppRHlwMkpla2E5ciJ9",
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("namespace", "The ID or name of the rate limit namespace.", cli.Required()),
			cli.Int64("limit", "Maximum number of overrides to return per page."),
			cli.String("cursor", "Pagination cursor from a previous response."),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			req := components.V2RatelimitListOverridesRequestBody{
				Namespace: cmd.String("namespace"),
				Cursor:    nil,
				Limit:     nil,
			}

			if v := cmd.Int64("limit"); v != 0 {
				req.Limit = &v
			}

			if v := cmd.String("cursor"); v != "" {
				req.Cursor = &v
			}

			start := time.Now()
			res, err := client.Ratelimit.ListOverrides(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2RatelimitListOverridesResponseBody, time.Since(start))
		},
	}
}
