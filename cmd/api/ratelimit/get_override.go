package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getOverrideCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-override",
		Usage: "Retrieve the configuration of a specific rate limit override by its identifier.",
		Description: `Retrieve the configuration of a specific rate limit override by its identifier.

Use this to inspect override configurations, audit rate limiting policies, or debug rate limiting behavior.

Important: The identifier must match exactly as specified when creating the override, including wildcard patterns.

Required permissions:
- ratelimit.*.read_override
- ratelimit.<namespace_id>.read_override

For full documentation, see https://www.unkey.com/docs/api-reference/v2/ratelimit/get-ratelimit-override` + util.Disclaimer,
		Examples: []string{
			"unkey api ratelimit get-override --namespace=api.requests --identifier=premium_user_123",
			`unkey api ratelimit get-override --namespace=api.requests --identifier="premium_*"`,
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("namespace", "The id or name of the namespace containing the override.", cli.Required()),
			cli.String("identifier", "The exact identifier pattern of the override to retrieve.", cli.Required()),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			start := time.Now()
			res, err := client.Ratelimit.GetOverride(ctx, components.V2RatelimitGetOverrideRequestBody{
				Namespace:  cmd.String("namespace"),
				Identifier: cmd.String("identifier"),
			})
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2RatelimitGetOverrideResponseBody, time.Since(start))
		},
	}
}
