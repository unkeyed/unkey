package rootkeys

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/sdks/api/go/v3/optionalnullable"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func createKeyCmd() *cli.Command {
	return &cli.Command{
		Name:  "create-key",
		Usage: "Create root key",
		Description: `Creates a root key for the authenticated workspace.
Requires unkey:v1:<workspace_id>:rootKeys/*#write.
The created root key cannot have more permissions than the caller.

For full documentation, see https://www.unkey.com/docs/platform/cli/root-keys/create-key` + util.Disclaimer,
		Examples: []string{
			"unkey api root-keys create-key --name=automation --permissions=unkey:v1:ws_example:rootKeys/*#read",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("name", "Optional name for the root key.", cli.MutuallyExclusive("body")),
			cli.StringSlice("permissions", "Comma-separated permission URNs. Use --body to create an empty permission set.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.Int64("expires", "Expiration as Unix milliseconds. Omit for no expiration.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.RootKeys.CreateKey, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2RootKeysCreateKeyResponseBody)
			}

			req := components.V2RootKeysCreateKeyRequestBody{
				Name:        nil,
				Permissions: cmd.StringSlice("permissions"),
				Expires:     nil,
			}
			if cmd.FlagIsSet("name") {
				req.Name = new(cmd.String("name"))
			}

			if cmd.FlagIsSet("expires") {
				req.Expires = optionalnullable.From(new(cmd.Int64("expires")))
			}

			res, err := client.RootKeys.CreateKey(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2RootKeysCreateKeyResponseBody)
		},
	}
}
