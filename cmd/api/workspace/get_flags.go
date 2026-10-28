package workspace

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getFlagsCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-flags",
		Usage: "Get workspace flags",
		Description: `Get feature flags for the workspace associated with your credentials.

Each flag is keyed by its slug and includes its current value, description,
and whether it can be enabled or disabled. A workspace-specific value takes
precedence over the flag's default. If no flags are available, data is an
empty object.

This endpoint takes no request body.
Required Permissions

Your root key must have the following permission:
- unkey:v1:<workspace_id>:flags#read

For full documentation, see https://www.unkey.com/docs/platform/cli/workspace/get-flags` + util.Disclaimer,
		Examples: []string{
			"unkey api workspace get-flags",
		},
		Flags: []cli.Flag{
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			res, err := client.Workspace.GetFlags(ctx)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2WorkspaceGetFlagsResponseBody)
		},
	}
}
