package domains

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func listDomainsCmd() *cli.Command {
	return &cli.Command{Name: "list-domains", Usage: "List the custom domains attached to an environment and their verification status.", Description: `List the custom domains attached to an environment and their verification status.

Results are paginated and sorted by their id. When hasMore is true, send the returned cursor to get the next page. An environment with no domains returns an empty array, not a 404.

Each domain includes its full dnsRecords, including which records Unkey has read back.

Required Permissions
- environment.*.read_domain (to read domains in any environment)
- environment.<environment_id>.read_domain (to read domains in a specific environment)

For full documentation, see https://www.unkey.com/docs/networking/domains` + util.Disclaimer, Examples: []string{"unkey api domains list-domains --project=payments --app=api --environment=production", "unkey api domains list-domains --project=payments --app=api --environment=production --limit=25 --search=acme.com"}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("app", "App ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("environment", "Environment ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.Int64("limit", "Maximum domains to return, from 1 to 100.", cli.Default(int64(100)), cli.MutuallyExclusive("body")), cli.String("cursor", "Pagination cursor from a previous response.", cli.MutuallyExclusive("body")), cli.String("search", "Case-insensitive domain ID or name filter.", cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}
		if cmd.FlagIsSet("body") {
			res, err := util.SendBody(ctx, client.Domains.ListDomains, cmd.String("body"))
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2DomainsListDomainsResponseBody)
		}
		req := components.V2DomainsListDomainsRequestBody{Project: cmd.String("project"), App: cmd.String("app"), Environment: cmd.String("environment"), Limit: ptr.P(cmd.Int64("limit")), Cursor: nil, Search: nil}
		if v := cmd.String("cursor"); v != "" {
			req.Cursor = &v
		}
		if v := cmd.String("search"); v != "" {
			req.Search = &v
		}
		res, err := client.Domains.ListDomains(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2DomainsListDomainsResponseBody)
	}}
}
