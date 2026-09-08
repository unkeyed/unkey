package domains

import (
	"context"
	"fmt"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/cli"
)

func createDomainCmd() *cli.Command {
	return &cli.Command{Name: "create-domain", Usage: "Attach a custom domain to an environment and start verifying it.", Description: `Attach a custom domain to an environment and start verifying it.

The domain is created in the pending state and does not serve traffic until verification succeeds. Verification runs in the background and polls DNS, so it is eventually consistent.

The response returns dnsRecords: every record needed to finish setup, already resolved for whether this domain is an apex or a subdomain. Create every entry exactly as given. One record establishes routing and one proves ownership, and both are needed: whether ownership can be inferred from the routing record depends on how your provider publishes it, and a name another workspace has already verified can only be claimed through the ownership record. Neither is knowable before the records exist.

When your DNS provider supports Domain Connect, the response also carries a domainConnect object; opening its url applies the same records at the provider in one step. The object is absent when the shortcut is unavailable.

Domains are unique per workspace, so the same name cannot be attached to two environments. Attaching a domain that already exists in your workspace returns a 409 conflict.

How many domains you may attach is set by your plan. Attaching one beyond that allowance returns a 403; upgrade the plan or remove a domain you no longer need.

Important: verification stops after 24 hours without the required DNS records, and the domain moves to failed.

Required Permissions
- environment.*.create_domain (to attach domains to any environment)
- environment.<environment_id>.create_domain (to attach domains to a specific environment)

For full documentation, see https://www.unkey.com/docs/networking/domains` + util.Disclaimer, Examples: []string{"unkey api domains create-domain --project=payments --app=api --environment=production --domain=api.acme.com"}, Flags: []cli.Flag{cli.String("body", "Decode this JSON as the endpoint request body. Request-building flags are mutually exclusive."), util.RootKeyFlag(), util.APIURLFlag(), util.ConfigFlag(), util.OutputFlag(), cli.String("project", "Project ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("app", "App ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("environment", "Environment ID or slug.", cli.Required(), cli.MutuallyExclusive("body")), cli.String("domain", "Fully qualified domain name to attach.", cli.Required(), cli.MutuallyExclusive("body"))}, Action: func(ctx context.Context, cmd *cli.Command) error {
		client, err := util.CreateClient(cmd)
		if err != nil {
			return err
		}
		if cmd.FlagIsSet("body") {
			res, err := util.SendBody(ctx, client.Domains.CreateDomain, cmd.String("body"))
			if err != nil {
				return err
			}
			return util.Output(cmd, res.V2DomainsCreateDomainResponseBody)
		}
		req := components.V2DomainsCreateDomainRequestBody{Project: cmd.String("project"), App: cmd.String("app"), Environment: cmd.String("environment"), Domain: cmd.String("domain")}
		res, err := client.Domains.CreateDomain(ctx, req)
		if err != nil {
			return fmt.Errorf("%s", util.FormatError(err))
		}
		return util.Output(cmd, res.V2DomainsCreateDomainResponseBody)
	}}
}
