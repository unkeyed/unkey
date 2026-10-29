package rootkeys

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func deleteKeyCmd() *cli.Command {
	return &cli.Command{
		Name:  "delete-key",
		Usage: "Delete root key",
		Description: `Deletes a root key in the authenticated workspace.
Requires unkey:v1:<workspace_id>:rootKeys/<key_id>#delete.
Use rootKeys/*#delete to delete any root key in the workspace.
Deleted keys cannot authenticate.

For full documentation, see https://www.unkey.com/docs/platform/cli/root-keys/delete-key` + util.Disclaimer,
		Examples: []string{
			"unkey api root-keys delete-key --key-id=key_example --yes",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			util.YesFlag(),
			cli.String("key-id", "Root key ID.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			req, err := util.RequestBody(cmd, components.V2RootKeysDeleteKeyRequestBody{
				KeyID: cmd.String("key-id"),
			})
			if err != nil {
				return err
			}
			if err := util.Confirm(cmd, fmt.Sprintf("Delete root key %q? It will no longer authenticate.", req.KeyID)); err != nil {
				return err
			}

			res, err := client.RootKeys.DeleteKey(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2RootKeysDeleteKeyResponseBody)
		},
	}
}
