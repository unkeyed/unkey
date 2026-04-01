// Package customdomain implements domain verification workflows.
//
// This package provides a Restate-based service for verifying custom domains
// through DNS record validation. When a user adds a custom domain to their project,
// this service orchestrates the verification process.
//
// # Verification Flow
//
// Verification has two paths:
//
//   - CNAME path (subdomains): The domain has a visible CNAME record pointing to its
//     unique target (e.g. <random>.unkey-dns.com). No TXT record needed since the
//     unique CNAME target proves routing intent.
//   - TXT path (apex domains): For apex domains using CNAME flattening (Cloudflare),
//     ALIAS/ANAME records, or Cloudflare proxy, the CNAME is not visible via DNS
//     lookup. A TXT record at _unkey.<domain> proves ownership instead.
//
// TXT is also always required when another workspace already has the same domain
// verified (contention), regardless of whether CNAME is visible. On successful
// contested verification, the old workspace's domain is revoked.
//
// # Why Restate
//
// DNS propagation is inherently slow and unpredictable. A user may add DNS records
// that take anywhere from seconds to hours to propagate globally, so the workflow
// needs durable execution that survives restarts, a single verification attempt
// per domain, and a long retry window. Restate provides virtual objects keyed by
// domain name, durable retries every minute for up to 24 hours, and exactly-once
// semantics for post-verification actions such as certificate issuance and routing.
//
// # Post-Verification
//
// Once verified, the service triggers certificate issuance via [certificate.Service]
// and creates frontline routes to enable traffic routing to the user's deployment.
//
// # Key Types
//
// [Service] implements hydrav1.CustomDomainServiceServer with handlers for domain
// verification. Configure it via [Config] and create instances with [New].
//
// # Usage
//
// Create a custom domain service:
//
//	svc := customdomain.New(customdomain.Config{
//	    DB:          database,
//	    CnameDomain: "unkey-dns.com",
//	})
//
// Register with Restate. The virtual object key is the domain name being verified:
//
//	client := hydrav1.NewCustomDomainServiceClient(ctx, "api.example.com")
//	client.VerifyDomain().Send(&hydrav1.VerifyDomainRequest{})
//
// # Retry Behavior
//
// The service uses a fixed 1-minute retry interval (no exponential backoff) for up to
// 24 hours (1440 attempts). If verification fails after this window, the domain is
// marked as failed and Restate terminates the invocation.
package customdomain
