package keys

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func setRolesCmd() *cli.Command {
	return &cli.Command{
		Name:  "set-roles",
		Usage: "Replace all roles on a key with the specified set in a single atomic operation",
		Description: `Replace all roles on a key with the specified set in a single atomic operation.

Use this to synchronize with external systems, reset roles to a known state, or apply standardized role templates. Direct permissions are never affected.

Important: Changes take effect immediately with up to 30-second edge propagation.

Required permissions:

Your root key must have one of the following permissions:
- api.*.update_key (to update keys in any API)
- api.<api_id>.update_key (to update keys in a specific API)

Side Effects:

Invalidates the key cache for immediate effect, and makes role changes available for verification within 30 seconds across all regions.

For full documentation, see https://www.unkey.com/docs/api-reference/v2/keys/set-key-roles` + util.Disclaimer,
		Examples: []string{
			"unkey api keys set-roles --key-id=key_1234abcd --roles=api_admin,billing_reader",
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("key-id", "The key ID to set roles on.", cli.Required()),
			cli.StringSlice("roles", "Comma-separated list of roles. Replaces all existing roles.", cli.Required()),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			start := time.Now()
			req := components.V2KeysSetRolesRequestBody{
				KeyID: cmd.String("key-id"),
				Roles: cmd.StringSlice("roles"),
			}

			res, err := client.Keys.SetRoles(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2KeysSetRolesResponseBody, time.Since(start))
		},
	}
}
