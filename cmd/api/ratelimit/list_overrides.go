package ratelimit

import (
	"context"
	"fmt"

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
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("namespace", "The ID or name of the rate limit namespace.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.Int64("limit", "Maximum number of overrides to return per page.", cli.MutuallyExclusive("body")),
			cli.String("cursor", "Pagination cursor from a previous response.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				body := cmd.String("body")
				res, err := util.SendBody(ctx, client.Ratelimit.ListOverrides, body)
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2RatelimitListOverridesResponseBody)
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

			res, err := client.Ratelimit.ListOverrides(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2RatelimitListOverridesResponseBody)
		},
	}
}
