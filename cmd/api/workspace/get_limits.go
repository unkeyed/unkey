package workspace

import (
	"context"

	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getLimitsCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-limits",
		Usage: "Get workspace limits",
		Description: `Get every limit of your workspace, with the current value against each
limit that has one.

Use this before an operation that a limit can reject, for example to see how
many custom domains or log drains remain, or how much CPU, memory, and disk
a new deployment can still reserve.

The request takes no parameters. Each limit has a limit, which is null
when unlimited, and a current value when the limit has one. Billable
operations count the current calendar month in UTC. The Compute limits are
present only when the workspace has a Compute plan.

Required Permissions

Your root key must have the following permission:
- unkey:v1:<workspace_id>:limits#read

For full documentation, see https://www.unkey.com/docs/platform/cli/workspace/get-limits` + util.Disclaimer,
		Examples: []string{
			"unkey api workspace get-limits",
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

			res, err := client.Workspace.GetLimits(ctx)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2WorkspaceGetLimitsResponseBody)
		},
	}
}
