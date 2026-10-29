package ratelimit

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func deleteOverrideCmd() *cli.Command {
	return &cli.Command{
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
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			util.YesFlag(),
			cli.String("namespace", "The id or name of the namespace containing the override.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("identifier", "The exact identifier pattern of the override to delete.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			req, err := util.RequestBody(cmd, components.V2RatelimitDeleteOverrideRequestBody{
				Namespace:  cmd.String("namespace"),
				Identifier: cmd.String("identifier"),
			})
			if err != nil {
				return err
			}
			if err := util.Confirm(cmd, fmt.Sprintf("Delete override %q in namespace %q and restore the default limit?", req.Identifier, req.Namespace)); err != nil {
				return err
			}
			res, err := client.Ratelimit.DeleteOverride(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2RatelimitDeleteOverrideResponseBody)
		},
	}
}
