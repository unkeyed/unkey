package domains

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func getDomainCmd() *cli.Command {
	return &cli.Command{Name: "get-domain", Usage: "Retrieve a custom domain and its verification status.", Description: `Retrieve a custom domain and its verification status.

Address the domain by its id or by its name. Names are unique per workspace, so api.acme.com is sufficient. You do not need to supply a project, app, or environment.

Use this endpoint to poll after domains.createDomain. Verification runs in the background and checks DNS approximately each minute.

status: verified means the domain is verified. Each DNS record's verified flag shows which records Unkey has read back. verificationError gives the reason for the last failed attempt.

Important: verification stops 24 hours after the domain was created, and the status becomes failed.

Required Permissions
- environment.*.read_domain (to read domains in any environment)
- environment.<environment_id>.read_domain (to read domains in a specific environment)

For full documentation, see https://www.unkey.com/docs/networking/domains` + util.Disclaimer, Examples: []string{"unkey api domains get-domain --domain=api.acme.com"}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("domain", "Domain ID or name.", cli.Required(), cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}
		if cmd.FlagIsSet("body") {
			res, err := util.SendBody(ctx, client.Domains.GetDomain, cmd.String("body"))
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2DomainsGetDomainResponseBody)
		}
		res, err := client.Domains.GetDomain(ctx, components.V2DomainsGetDomainRequestBody{Domain: cmd.String("domain")})
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2DomainsGetDomainResponseBody)
	}}
}
