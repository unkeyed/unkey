package keys

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func removePermissionsCmd() *cli.Command {
	return &cli.Command{
		Name:  "remove-permissions",
		Usage: "Remove permissions from a key without affecting existing roles or other permissions",
		Description: `Remove permissions from a key without affecting existing roles or other permissions.

Use this for privilege downgrades, removing temporary access, or plan changes that revoke specific capabilities. Permissions granted through roles remain unchanged.

Important: Changes take effect immediately with up to 30-second edge propagation.

Required permissions:

Your root key must have one of the following permissions:
- api.*.update_key (to update keys in any API)
- api.<api_id>.update_key (to update keys in a specific API)

Side Effects:

Invalidates the key cache for immediate effect, and makes permission changes available for verification within 30 seconds across all regions.

For full documentation, see https://www.unkey.com/docs/api-reference/v2/keys/remove-key-permissions` + util.Disclaimer,
		Examples: []string{
			"unkey api keys remove-permissions --key-id=key_1234abcd --permissions=documents.read,documents.write",
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("key-id", "The key ID to remove permissions from.", cli.Required()),
			cli.StringSlice("permissions", "Comma-separated list of permission names to remove.", cli.Required()),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(cmd)
			if err != nil {
				return err
			}

			start := time.Now()
			req := components.V2KeysRemovePermissionsRequestBody{
				KeyID:       cmd.String("key-id"),
				Permissions: cmd.StringSlice("permissions"),
			}

			res, err := client.Keys.RemovePermissions(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2KeysRemovePermissionsResponseBody, time.Since(start))
		},
	}
}
