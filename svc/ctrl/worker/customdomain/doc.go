// Package customdomain implements domain ownership verification workflows.
//
// This package provides a Restate-based service for verifying custom domain ownership
// through DNS record validation. When a user adds a custom domain to their project,
// this service orchestrates the verification process that proves they control the domain.
//
// # Verification Flow
//
// Domain verification uses a two-step process. TXT record verification proves
// ownership by checking for a TXT record at _unkey.<domain> containing a unique
// token, and must complete before CNAME verification. CNAME verification enables
// traffic routing by checking that the domain points to a unique target subdomain
// under the platform's DNS apex (for example, <random>.unkey-dns.com). Both checks
// must succeed before the domain is marked as verified.
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
