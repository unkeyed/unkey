package keys

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

var addRolesCmd = &cli.Command{
	Name:  "add-roles",
	Usage: "Add roles to a key without affecting existing roles or permissions",
	Description: `Add roles to a key without affecting existing roles or permissions.

Use this for privilege upgrades, enabling new feature sets, or subscription changes that grant additional role-based capabilities. Direct permissions remain unchanged.

Important: Changes take effect immediately with up to 30-second edge propagation.

Required permissions:

Your root key must have one of the following permissions:
- api.*.update_key (to update keys in any API)
- api.<api_id>.update_key (to update keys in a specific API)

Side Effects:

Invalidates the key cache for immediate effect, and makes role assignments available for verification within 30 seconds across all regions.

For full documentation, see https://www.unkey.com/docs/api-reference/v2/keys/add-key-roles` + util.Disclaimer,
	Examples: []string{
		"unkey api keys add-roles --key-id=key_1234abcd --roles=api_admin,billing_reader",
	},
	Flags: []cli.Flag{
		util.RootKeyFlag(),
		util.APIURLFlag(),
		util.ConfigFlag(),
		util.OutputFlag(),
		cli.String("key-id", "The key ID to add roles to.", cli.Required()),
		cli.StringSlice("roles", "Comma-separated list of role names to add.", cli.Required()),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		start := time.Now()
		req := components.V2KeysAddRolesRequestBody{
			KeyID: cmd.String("key-id"),
			Roles: cmd.StringSlice("roles"),
		}

		res, err := client.Keys.AddRoles(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2KeysAddRolesResponseBody, time.Since(start))
	},
}
