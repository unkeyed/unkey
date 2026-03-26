package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

var deleteOverrideCmd = &cli.Command{
	Name:  "delete-override",
	Usage: "Permanently remove a rate limit override.",
	Description: `Permanently remove a rate limit override. Affected identifiers immediately revert to the namespace default.

Use this to remove temporary overrides, reset identifiers to standard limits, or clean up outdated rules.

Important: Deletion is immediate and permanent. The override cannot be recovered and must be recreated if needed again.

Required permissions:
- ratelimit.*.delete_override
- ratelimit.<namespace_id>.delete_override

For full documentation, see https://www.unkey.com/docs/api-reference/v2/ratelimit/delete-ratelimit-override` + util.Disclaimer,
	Examples: []string{
		"unkey api ratelimit delete-override --namespace=api.requests --identifier=premium_user_123",
		`unkey api ratelimit delete-override --namespace=api.requests --identifier="premium_*"`,
	},
	Flags: []cli.Flag{
		util.RootKeyFlag(),
		util.APIURLFlag(),
		util.ConfigFlag(),
		util.OutputFlag(),
		cli.String("namespace", "The id or name of the namespace containing the override.", cli.Required()),
		cli.String("identifier", "The exact identifier pattern of the override to delete.", cli.Required()),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		start := time.Now()
		res, err := client.Ratelimit.DeleteOverride(ctx, components.V2RatelimitDeleteOverrideRequestBody{
			Namespace:  cmd.String("namespace"),
			Identifier: cmd.String("identifier"),
		})
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2RatelimitDeleteOverrideResponseBody, time.Since(start))
	},
}
