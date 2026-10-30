package logs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func listAPIRequestsCmd() *cli.Command {
	return &cli.Command{
		Name:  "list-api-requests",
		Usage: "List API requests",
		Description: `List the requests that the Unkey API served for your workspace, newest first.

Filter by time, path, method, request ID, and status. In one field, values under is, contains, startsWith, and endsWith are combined with OR. Values under isNot and notContains, and the range operators gt, gte, lt, and lte, are combined with AND. Different fields are combined with AND. Every text match is exact and case-sensitive.

To get the next page, send the pagination.cursor of the previous page together with the same filters. A cursor does not expire, but rows older than the log retention leave the listing.

Unkey returns only requests inside the log retention of your workspace. The calls to this endpoint are not logged.

Required Permissions

Your root key must have the following permission:
- unkey:v1:<workspace_id>:logs/apiRequests#read

For full documentation, see https://www.unkey.com/docs/platform/cli/logs/list-api-requests` + util.Disclaimer,
		Examples: []string{
			"unkey api logs list-api-requests --filters='{\"status\":{\"gte\":400,\"lt\":600}}' --limit=20",
		},
		Flags: []cli.Flag{
			cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."),
			util.RootKeyFlag(),
			util.APIURLFlag(),
			util.ConfigFlag(),
			util.OutputFlag(),
			cli.Int64("limit", "Maximum requests per page, between 1 and 100.", cli.Default(int64(50)), cli.MutuallyExclusive("body")),
			cli.String("cursor", "Pagination cursor. Keep the same filters on every page.", cli.MutuallyExclusive("body")),
			cli.String("filters", "Request filters as a JSON object.", cli.MutuallyExclusive("body")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, err := util.CreateClient(ctx, cmd)
			if err != nil {
				return err
			}

			if cmd.FlagIsSet("body") {
				res, err := util.SendBody(ctx, client.Logs.ListAPIRequests, cmd.String("body"))
				if err != nil {
					return err
				}
				return util.Output(cmd, res.V2LogsListAPIRequestsResponseBody)
			}

			req := components.V2LogsListAPIRequestsRequestBody{
				Limit:   new(cmd.Int64("limit")),
				Cursor:  nil,
				Filters: nil,
			}
			if cmd.FlagIsSet("cursor") {
				req.Cursor = new(cmd.String("cursor"))
			}

			if cmd.FlagIsSet("filters") {
				var value components.V2LogsListAPIRequestsFilters
				if err := json.Unmarshal([]byte(cmd.String("filters")), &value); err != nil {
					return fmt.Errorf("invalid JSON for --filters: %w", err)
				}
				req.Filters = &value
			}

			res, err := client.Logs.ListAPIRequests(ctx, req)
			if err != nil {
				return fmt.Errorf("%s", util.FormatError(err))
			}
			return util.Output(cmd, res.V2LogsListAPIRequestsResponseBody)
		},
	}
}
