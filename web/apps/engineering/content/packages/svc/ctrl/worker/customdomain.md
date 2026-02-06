---
title: customdomain
description: "implements domain ownership verification workflows"
---

Package customdomain implements domain ownership verification workflows.

This package provides a Restate-based service for verifying custom domain ownership through DNS record validation. When a user adds a custom domain to their project, this service orchestrates the verification process that proves they control the domain.

### Verification Flow

Domain verification uses a two-step process. TXT record verification proves ownership by checking for a TXT record at \_unkey.\<domain> containing a unique token, and must complete before CNAME verification. CNAME verification enables traffic routing by checking that the domain points to a unique target subdomain under the platform's DNS apex (for example, \<random>.unkey-dns.com). Both checks must succeed before the domain is marked as verified.

### Why Restate

DNS propagation is inherently slow and unpredictable. A user may add DNS records that take anywhere from seconds to hours to propagate globally, so the workflow needs durable execution that survives restarts, a single verification attempt per domain, and a long retry window. Restate provides virtual objects keyed by domain name, durable retries every minute for up to 24 hours, and exactly-once semantics for post-verification actions such as certificate issuance and routing.

### Post-Verification

Once verified, the service triggers certificate issuance via \[certificate.Service] and creates frontline routes to enable traffic routing to the user's deployment.

### Key Types

\[Service] implements hydrav1.CustomDomainServiceServer with handlers for domain verification. Configure it via \[Config] and create instances with \[New].

### Usage

Create a custom domain service:

	svc := customdomain.New(customdomain.Config{
	    DB:          database,
	    CnameDomain: "unkey-dns.com",
	})

Register with Restate. The virtual object key is the domain name being verified:

	client := hydrav1.NewCustomDomainServiceClient(ctx, "api.example.com")
	client.VerifyDomain().Send(&hydrav1.VerifyDomainRequest{})

### Retry Behavior

The service uses a fixed 1-minute retry interval (no exponential backoff) for up to 24 hours (1440 attempts). If verification fails after this window, the domain is marked as failed and Restate terminates the invocation.

## Constants

maxVerificationDuration limits how long we retry DNS verification before marking a domain as failed.
```go
const maxVerificationDuration = 24 * time.Hour
```


## Variables

```go
var _ hydrav1.CustomDomainServiceServer = (*Service)(nil)
```

errNotVerified signals incomplete verification and triggers Restate retries.
```go
var errNotVerified = errors.New("domain not verified yet")
```


## Types

### type Config

```go
type Config struct {
	// DB provides database access for custom domain records.
	DB db.Database

	// CnameDomain is the base domain for custom domain CNAME targets.
	// Each custom domain gets a unique subdomain like "{random}.{CnameDomain}".
	// For production: "unkey-dns.com"
	// For local: "unkey.local"
	CnameDomain string
}
```

Config holds configuration for creating a \[Service] instance.

### type Service

```go
type Service struct {
	hydrav1.UnimplementedCustomDomainServiceServer
	db          db.Database
	cnameDomain string
}
```

Service orchestrates custom domain verification workflows.

Service implements hydrav1.CustomDomainServiceServer with handlers for verifying domain ownership via CNAME records. It uses a Restate virtual object pattern keyed by domain name to ensure only one verification workflow runs per domain at any time.

The verification process checks that the user has added a CNAME record pointing to a unique target under the configured DNS apex. Verification retries every minute for approximately 24 hours before giving up.

Once verified, the service triggers certificate issuance and creates a frontline route so traffic can be routed to the user's deployment.

#### func New

```go
func New(cfg Config) *Service
```

New creates a \[Service] with the given configuration.

#### func (Service) RetryVerification

```go
func (s *Service) RetryVerification(
	ctx restate.ObjectContext,
	_ *hydrav1.RetryVerificationRequest,
) (*hydrav1.RetryVerificationResponse, error)
```

RetryVerification resets a failed domain and restarts verification after the user fixes DNS configuration.

#### func (Service) VerifyDomain

```go
func (s *Service) VerifyDomain(
	ctx restate.ObjectContext,
	_ *hydrav1.VerifyDomainRequest,
) (*hydrav1.VerifyDomainResponse, error)
```

VerifyDomain performs two-step verification for a custom domain: 1. TXT record verification (proves ownership) - must verify first 2. CNAME record verification (enables routing) - only checked after TXT verified

This is a Restate virtual object handler keyed by domain name, ensuring only one verification workflow runs per domain at any time. The handler checks DNS once per invocation - Restate's retry policy handles periodic re-checks (every 1 minute for up to 24 hours).

Once both verifications succeed, the workflow: 1. Updates domain status to "verified" 2. Creates an ACME challenge record to trigger certificate issuance 3. Creates a frontline route to enable traffic routing

If verification fails after ~24 hours of retries, Restate kills the invocation.

