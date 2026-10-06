// Package portaldomain verifies the hostnames tenants attach to their portal.
//
// It mirrors the deploy customdomain workflow with its own Restate virtual object, keyed by
// portal domain id, so deploy verification is never re-keyed. The DNS and
// contention rules come from [domainverify]; this package owns the
// portal_domains reads and writes and the success steps.
//
// Every tenant's portal is served by one Unkey Deploy app, so a verified
// hostname is routed to that app's configured production environment with
// sticky=live and follows its deploys. The certificate is still the tenant's:
// issuance runs under the tenant workspace, not the portal app's.
//
// # Production rollout
//
// Portal domains need all of the following before the first one can verify:
//
//   - The portal_domains table exists on PlanetScale, created from
//     pkg/mysql/schema/portal_domains.sql before ctrl is deployed.
//   - A wildcard DNS record *.portal.unkey-dns.com points at frontline, since
//     each tenant CNAMEs its hostname to a random label under it.
//   - ctrl api sets portal_cname_domain = "portal.unkey-dns.com".
//   - ctrl api and ctrl worker both set [portal] environment_id to the portal
//     app's production environment. Without it the API refuses new domains and
//     the worker keeps retrying domains whose DNS already passes.
package portaldomain
