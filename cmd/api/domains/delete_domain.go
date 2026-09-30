package domains

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func deleteDomainCmd() *cli.Command {
	return &cli.Command{Name: "delete-domain", Usage: "Delete a custom domain from your workspace.", Description: `Delete a custom domain from your workspace.

Address the domain by its id or by its name. Names are unique per workspace, so api.acme.com is enough.

Unkey stops serving the domain. Later requests fail with a certificate error. The DNS records at your provider stay in place.

Required Permissions
- environment.*.delete_domain (to delete domains in any environment)
- environment.<environment_id>.delete_domain (to delete domains in a specific environment)

For full documentation, see https://www.unkey.com/docs/networking/domains` + util.Disclaimer, Examples: []string{"unkey api domains delete-domain --domain=api.acme.com"}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("domain", "Domain ID or name.", cli.Required(), cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}
		if cmd.FlagIsSet("body") {
			res, err := util.SendBody(ctx, client.Domains.DeleteDomain, cmd.String("body"))
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2DomainsDeleteDomainResponseBody)
		}
		res, err := client.Domains.DeleteDomain(ctx, components.V2DomainsDeleteDomainRequestBody{Domain: cmd.String("domain")})
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2DomainsDeleteDomainResponseBody)
	}}
}
