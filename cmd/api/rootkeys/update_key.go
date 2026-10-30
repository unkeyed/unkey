package rootkeys

import (
	"context"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/sdks/api/go/v3/optionalnullable"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func updateKeyCmd() *cli.Command {
	return &cli.Command{
		Name:  "update-key",
		Usage: "Update root key",
		Description: `Updates a root key in the authenticated workspace.
Requires unkey:v1:<workspace_id>:rootKeys/<key_id>#write.
Use rootKeys/*#write to update any root key in the workspace.

permissions replaces the complete permission set. Each permission must be a supported URN
in the authenticated workspace and within the caller's permissions.
An expiring root key can only update root keys that expire no later than itself.

For full documentation, see https://www.unkey.com/docs/platform/cli/root-keys/update-key` + util.Disclaimer,
		Examples: []string{
			"unkey api root-keys update-key --key-id=key_example --enabled=false",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("key-id", "Root key ID.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.String("name", "New name. Use --body with name:null to remove it.", cli.MutuallyExclusive("body")),
			cli.Bool("enabled", "Whether the root key can authenticate.", cli.MutuallyExclusive("body")),
			cli.StringSlice("permissions", "Replacement comma-separated permission URNs; an empty value removes all permissions.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.RootKeys.UpdateKey, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2RootKeysUpdateKeyResponseBody)
			}

			req := components.V2RootKeysUpdateKeyRequestBody{
				KeyID:       cmd.String("key-id"),
				Name:        nil,
				Enabled:     nil,
				Permissions: nil,
			}
			if cmd.FlagIsSet("name") {
				req.Name = optionalnullable.From(new(cmd.String("name")))
			}

			if cmd.FlagIsSet("enabled") {
				req.Enabled = new(cmd.Bool("enabled"))
			}

			if cmd.FlagIsSet("permissions") {
				req.Permissions = append([]string{}, cmd.StringSlice("permissions")...)
			}

			res, err := client.RootKeys.UpdateKey(ctx, req)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2RootKeysUpdateKeyResponseBody)
		},
	}
}
