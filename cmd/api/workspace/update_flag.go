package workspace

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func updateFlagCmd() *cli.Command {
	return &cli.Command{
		Name:  "update-flag",
		Usage: "Update a workspace flag",
		Description: `Enable or disable a flag for your workspace. Use workspace.getFlags to
find flag slugs and check whether they can be enabled or disabled.

Required Permissions

Your root key must have the following permission:
- unkey:v1:<workspace_id>:flags#write

For full documentation, see https://www.unkey.com/docs/platform/cli/workspace/update-flag` + util.Disclaimer,
		Examples: []string{
			"unkey api workspace update-flag --slug=example_feature --enabled=false",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("slug", "Flag slug returned by workspace flags get.", cli.Required(), cli.MutuallyExclusive("body")),
			cli.Bool("enabled", "Set true to enable the flag or false to disable it.", cli.Required(), cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Workspace.UpdateFlag, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2WorkspaceUpdateFlagResponseBody)
			}

			req := components.V2WorkspaceUpdateFlagRequestBody{
				Slug:    cmd.String("slug"),
				Enabled: cmd.Bool("enabled"),
			}

			res, err := client.Workspace.UpdateFlag(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2WorkspaceUpdateFlagResponseBody)
		},
	}
}
