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
	return &cli.Command{Name: "list-domains", Usage: "List your custom domains with their verification status and DNS records.", Description: `List your custom domains with their verification status and DNS records.

Filter by project, app, or environment using IDs or slugs, or send {} to list domains across your workspace.

Use any filter on its own or combine filters to narrow the results. Results match all supplied filters. Omitting environment includes all matching environments.

Results include only domains you have permission to read, sorted by ID. When hasMore is true, send the returned cursor to get the next page.

status: verified means the domain is verified. Unkey has configured routing and requested a certificate. Each domain includes its full dnsRecords. Each record has a verified flag. The flag shows which records Unkey has read back, so you can see which records are still missing without a second call. Some providers hide a record from DNS lookups, for example a proxied or flattened routing record. Such a record stays false while it serves traffic.

Required Permissions

Use a root key with the environment.*.read_domain permission. A successful request returns an empty list if no matching domains are readable by your key.

For full documentation, see https://www.unkey.com/docs/networking/domains` + util.Disclaimer, Examples: []string{"unkey api domains list-domains", "unkey api domains list-domains --app=api", "unkey api domains list-domains --project=payments --app=api --environment=production --limit=25 --search=acme.com"}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug filter.", cli.MutuallyExclusive("body")), cli.String("app", "App ID or slug filter.", cli.MutuallyExclusive("body")), cli.String("environment", "Environment ID or slug filter.", cli.MutuallyExclusive("body")), cli.Int64("limit", "Maximum domains to return, from 1 to 100.", cli.Default(int64(100)), cli.MutuallyExclusive("body")), cli.String("cursor", "Pagination cursor from a previous response.", cli.MutuallyExclusive("body")), cli.String("search", "Case-insensitive domain ID or name filter.", cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
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
		req := components.V2DomainsListDomainsRequestBody{Project: nil, App: nil, Environment: nil, Limit: ptr.P(cmd.Int64("limit")), Cursor: nil, Search: nil}
		if v := cmd.String("project"); v != "" {
			req.Project = &v
		}
		if v := cmd.String("app"); v != "" {
			req.App = &v
		}
		if v := cmd.String("environment"); v != "" {
			req.Environment = &v
		}
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
