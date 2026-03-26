package identities

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

var listIdentitiesCmd = &cli.Command{
	Name:  "list-identities",
	Usage: "Get a paginated list of all identities in your workspace",
	Description: `Get a paginated list of all identities in your workspace. Returns metadata and rate limit configurations.

Perfect for building management dashboards, auditing configurations, or browsing your identities.

Important: Requires identity.*.read_identity permission

For full documentation, see https://www.unkey.com/docs/api-reference/v2/identities/list-identities` + util.Disclaimer,
	Examples: []string{
		"unkey api identities list-identities",
		"unkey api identities list-identities --limit=50",
		"unkey api identities list-identities --limit=50 --cursor=cursor_eyJrZXkiOiJrZXlfMTIzNCJ9",
	},
	Flags: []cli.Flag{
		util.RootKeyFlag(),
		util.APIURLFlag(),
		util.ConfigFlag(),
		util.OutputFlag(),
		cli.Int64("limit", "Maximum number of identities to return per page."),
		cli.String("cursor", "Pagination cursor from a previous response."),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}

		req := components.V2IdentitiesListIdentitiesRequestBody{
			Limit:  nil,
			Cursor: nil,
		}

		if v := cmd.Int64("limit"); v != 0 {
			req.Limit = &v
		}

		if v := cmd.String("cursor"); v != "" {
			req.Cursor = &v
		}

		start := time.Now()
		res, err := client.Identities.ListIdentities(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2IdentitiesListIdentitiesResponseBody, time.Since(start))
	},
}
