package domains

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func verifyDomainCmd() *cli.Command {
	return &cli.Command{Name: "verify-domain", Usage: "Restart verification for a custom domain.", Description: `Restart verification for a custom domain.

Address the domain by its ID or by its name. Names are unique per workspace, so api.acme.com is enough.

Call this after correcting DNS records for a failed domain, or to give a pending domain a new 24-hour verification period. Poll domains.getDomain for the result. A domain that is already verified returns a 412.

Required Permissions
- environment.*.verify_domain (to verify domains in any environment)
- environment.<environment_id>.verify_domain (to verify domains in a specific environment)

For full documentation, see https://www.unkey.com/docs/networking/domains` + util.Disclaimer, Examples: []string{"unkey api domains verify-domain --domain=api.acme.com"}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("domain", "Domain ID or name.", cli.Required(), cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}
		if cmd.FlagIsSet("body") {
			res, err := util.SendBody(ctx, client.Domains.VerifyDomain, cmd.String("body"))
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2DomainsVerifyDomainResponseBody)
		}
		res, err := client.Domains.VerifyDomain(ctx, components.V2DomainsVerifyDomainRequestBody{Domain: cmd.String("domain")})
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2DomainsVerifyDomainResponseBody)
	}}
}
