package workspace

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getUsageCmd() *cli.Command {
	return &cli.Command{
		Name:  "get-usage",
		Usage: "Get workspace usage",
		Description: `Get the usage of your workspace for one calendar month (UTC): billable API
operations, compute usage per environment, and active keys per app.

Use this to track usage during the month or to look back at a past
month. The response has no prices. Your invoice is the source of truth.
Compute usage can be up to 15 minutes late, and it can change for up to 7
days.

Send {} for the current month to date, or
{"period": {"year": 2026, "month": 9}} for a full past month. The month
must start within the last 90 days.

Required Permissions

Your root key must have this permission:
- unkey:v1:<workspace_id>:usage#read

For full documentation, see https://www.unkey.com/docs/platform/cli/workspace/get-usage` + util.Disclaimer,
		Examples: []string{
			"unkey api workspace get-usage --period='{\"year\":2026,\"month\":9}'",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.String("period", "Calendar month as a JSON object with year and month. Omit for the current month.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Workspace.GetUsage, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2WorkspaceGetUsageResponseBody)
			}

			req := components.V2WorkspaceGetUsageRequestBody{
				Period: nil,
			}
			if cmd.FlagIsSet("period") {
				var value components.V2WorkspaceGetUsageRequestPeriod
				if err := json.Unmarshal([]byte(cmd.String("period")), &value); err != nil {
					return fmt.Errorf("invalid JSON for --period: %w", err)
				}
				req.Period = &value
			}

			res, err := client.Workspace.GetUsage(ctx, req)
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2WorkspaceGetUsageResponseBody)
		},
	}
}
