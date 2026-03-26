package keys

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

var setPermissionsCmd = &cli.Command{
	Name:  "set-permissions",
	Usage: "Replace all permissions on a key with the specified set in a single atomic operation",
	Description: `Replace all permissions on a key with the specified set in a single atomic operation.

Use this to synchronize with external systems, reset permissions to a known state, or apply standardized permission templates. Permissions granted through roles remain unchanged.

Important: Changes take effect immediately with up to 30-second edge propagation.

Required permissions:

Your root key must have one of the following permissions:
- api.*.update_key (to update keys in any API)
- api.<api_id>.update_key (to update keys in a specific API)

Side Effects:

Invalidates the key cache for immediate effect, and makes permission changes available for verification within 30 seconds across all regions.

For full documentation, see https://www.unkey.com/docs/api-reference/v2/keys/set-key-permissions` + util.Disclaimer,
	Examples: []string{
		"unkey api keys set-permissions --key-id=key_1234abcd --permissions=documents.read,documents.write,settings.view",
	},
	Flags: []cli.Flag{
		util.RootKeyFlag(),
		util.APIURLFlag(),
		util.ConfigFlag(),
		util.OutputFlag(),
		cli.String("key-id", "The key ID to set permissions on.", cli.Required()),
		cli.StringSlice("permissions", "Comma-separated list of permissions. Replaces all existing permissions.", cli.Required()),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		start := time.Now()
		req := components.V2KeysSetPermissionsRequestBody{
			KeyID:       cmd.String("key-id"),
			Permissions: cmd.StringSlice("permissions"),
		}

		res, err := client.Keys.SetPermissions(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2KeysSetPermissionsResponseBody, time.Since(start))
	},
}
