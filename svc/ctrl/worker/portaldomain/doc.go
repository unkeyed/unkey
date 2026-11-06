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
package portaldomain
