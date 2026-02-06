---
title: certificate
description: "implements ACME certificate workflows for SSL/TLS provisioning"
---

Package certificate implements ACME certificate workflows for SSL/TLS provisioning.

This package orchestrates the complete certificate lifecycle using the ACME protocol, coordinating with certificate authorities like Let's Encrypt to validate domain ownership and obtain certificates. It supports both HTTP-01 challenges for regular domains and DNS-01 challenges for wildcard certificates.

### Why Restate

Certificate issuance involves multiple external dependencies (ACME servers, DNS propagation, database writes) that can fail independently. We use Restate for durable execution because ACME challenges have strict timing requirements and rate limits. If a challenge fails partway through, we cannot simply restart from the beginning without risking Let's Encrypt rate limits. Restate's exactly-once execution semantics allow the workflow to resume from the last completed step after crashes or network failures.

The virtual object model keys workflows by domain name. This prevents race conditions where two processes might simultaneously request certificates for the same domain, which would trigger ACME duplicate certificate rate limits.

### Key Types

\[Service] is the main entry point implementing hydrav1.CertificateServiceServer. Configure it via \[Config] and create instances with \[New]. The service exposes two primary handlers: \[Service.ProcessChallenge] for obtaining certificates and \[Service.RenewExpiringCertificates] as a self-scheduling renewal cron job.

\[EncryptedCertificate] holds certificate data with the private key encrypted via the vault service before database storage.

\[BootstrapConfig] configures infrastructure certificate bootstrapping for wildcard certificates needed by the platform itself.

### Challenge Types

The service automatically selects the appropriate ACME challenge type based on the domain. Wildcard domains (e.g., "\*.example.com") require DNS-01 challenges because HTTP-01 cannot prove control over all possible subdomains. Regular domains use HTTP-01 which is faster since it avoids DNS propagation delays.

### Rate Limit Handling

Let's Encrypt enforces rate limits that cannot be bypassed. When rate limited, \[Service.ProcessChallenge] uses Restate's durable sleep to wait until the retry-after time, then automatically retries. This prevents the workflow from consuming retry budget while waiting. Sleep duration is capped at 2 hours with a 1 minute buffer added to the retry-after time.

### Security

Private keys are encrypted using the vault service before storage. The encryption is workspace-scoped via keyring isolation. Certificates themselves are stored unencrypted for fast retrieval by sentinels that terminate TLS.

### Error Handling

The package distinguishes between retryable errors (network timeouts, temporary ACME server issues) and terminal errors (invalid credentials, unauthorized domains). Terminal errors use restate.TerminalError to prevent infinite retry loops. Rate limit errors are handled specially with durable sleeps rather than retries.

## Constants

InfraWorkspaceID is the workspace ID for infrastructure certificates. Infrastructure certs are owned by this synthetic workspace rather than a customer workspace, allowing them to be managed separately and avoiding conflicts with customer domain records.
```go
const InfraWorkspaceID = "unkey_internal"
```

globalAcmeUserID identifies the shared ACME account used for all certificate requests to avoid per-workspace account creation and stay under account limits.
```go
const globalAcmeUserID = "acme"
```


## Variables

```go
var _ hydrav1.CertificateServiceServer = (*Service)(nil)
```


## Functions


## Types

### type BootstrapConfig

```go
type BootstrapConfig struct {
	// DefaultDomain is the base domain for customer deployments. If set, a wildcard
	// certificate for "*.{DefaultDomain}" is provisioned to terminate TLS for all
	// customer subdomains.
	DefaultDomain string

	// RegionalDomain is the base domain for cross-region communication between
	// frontline instances. Combined with each entry in Regions to create per-region
	// wildcard certificates.
	RegionalDomain string

	// Regions lists all deployment regions. For each region, a wildcard certificate
	// is created as "*.{region}.{RegionalDomain}" to secure inter-region traffic.
	Regions []string

	// Restate is the ingress client used to trigger [Service.ProcessChallenge] workflows.
	// Infrastructure cert bootstrapping delegates to the standard challenge flow rather
	// than implementing separate certificate logic.
	Restate *restateIngress.Client
}
```

BootstrapConfig holds configuration for \[Service.BootstrapInfraCerts].

### type Config

```go
type Config struct {
	// DB provides database access for domain, certificate, and ACME challenge records.
	DB db.Database

	// Vault encrypts private keys before database storage. Keys are encrypted using
	// the workspace ID as the keyring identifier.
	Vault vaultv1connect.VaultServiceClient

	// EmailDomain forms the email address for ACME account registration. The service
	// constructs emails as "acme@{EmailDomain}" for the global ACME account.
	EmailDomain string

	// DefaultDomain is the base domain for infrastructure wildcard certificates,
	// used by [Service.BootstrapInfraCerts] to provision platform TLS.
	DefaultDomain string

	// DNSProvider handles DNS-01 challenges required for wildcard certificates.
	// Must be set to issue wildcard certs; ignored for regular domain certificates.
	DNSProvider challenge.Provider

	// HTTPProvider handles HTTP-01 challenges for regular (non-wildcard) certificates.
	// Must be set to issue regular certs; ignored for wildcard certificates.
	HTTPProvider challenge.Provider

	// Heartbeat sends health signals after successful certificate renewal runs.
	// If nil, no heartbeat is sent.
	Heartbeat healthcheck.Heartbeat
}
```

Config holds configuration for creating a \[Service] instance.

### type EncryptedCertificate

```go
type EncryptedCertificate struct {
	// CertificateID is the unique identifier for this certificate, generated using
	// uid.New with the certificate prefix.
	CertificateID string

	// Certificate contains the PEM-encoded certificate chain including intermediates.
	Certificate string

	// EncryptedPrivateKey is the vault-encrypted PEM-encoded private key.
	EncryptedPrivateKey string

	// ExpiresAt is the certificate expiration time as Unix milliseconds. Parsed from
	// the certificate's NotAfter field; defaults to 90 days from issuance if parsing
	// fails.
	ExpiresAt int64
}
```

EncryptedCertificate holds a certificate with its private key encrypted for storage. The private key is encrypted using the vault service with the workspace ID as the keyring, ensuring keys can only be decrypted by the owning workspace.

### type Service

```go
type Service struct {
	hydrav1.UnimplementedCertificateServiceServer
	db            db.Database
	vault         vaultv1connect.VaultServiceClient
	emailDomain   string
	defaultDomain string
	dnsProvider   challenge.Provider
	httpProvider  challenge.Provider
	heartbeat     healthcheck.Heartbeat
}
```

Service orchestrates ACME certificate issuance and renewal.

Service implements hydrav1.CertificateServiceServer with two main handlers: \[Service.ProcessChallenge] for obtaining/renewing individual certificates, and \[Service.RenewExpiringCertificates] for batch renewal (called via GitHub Actions). It also provides \[Service.BootstrapInfraCerts] for provisioning infrastructure wildcard certificates at startup.

The service uses a single global ACME account (not per-workspace) to simplify key management and avoid hitting Let's Encrypt's account rate limits. Challenge type selection is automatic: wildcard domains use DNS-01, regular domains use HTTP-01 for faster issuance.

Not safe for concurrent use on the same domain. Concurrency control is handled by Restate's virtual object model which keys handlers by domain name.

#### func New

```go
func New(cfg Config) *Service
```

New creates a \[Service] with the given configuration. The returned service is ready to handle certificate requests but does not start any background processes. Call \[Service.BootstrapInfraCerts] at startup to provision infrastructure certs. \[Service.RenewExpiringCertificates] is intended to be called on a schedule via GitHub Actions.

#### func (Service) BootstrapInfraCerts

```go
func (s *Service) BootstrapInfraCerts(ctx context.Context, cfg BootstrapConfig) error
```

BootstrapInfraCerts provisions wildcard certificates for platform infrastructure.

This method ensures the platform has valid TLS certificates for its own domains before serving customer traffic. It creates database records for each infrastructure domain and triggers \[Service.ProcessChallenge] via Restate to obtain certificates.

The method is idempotent: domains with existing valid certificates are skipped, and domains with pending challenges are not re-triggered. This makes it safe to call on every service startup without risking duplicate certificate requests.

Returns nil without error if DNSProvider is not configured, since infrastructure certs require DNS-01 challenges for wildcards. Logs a warning in this case.

#### func (Service) ProcessChallenge

```go
func (s *Service) ProcessChallenge(
	ctx restate.ObjectContext,
	req *hydrav1.ProcessChallengeRequest,
) (resp *hydrav1.ProcessChallengeResponse, err error)
```

ProcessChallenge obtains or renews an SSL/TLS certificate for a domain.

This is a Restate virtual object handler keyed by domain name, ensuring only one certificate challenge runs per domain at any time. The workflow consists of durable steps that survive process restarts: domain resolution, challenge claiming, certificate obtainment, persistence, and verification marking.

The method uses the saga pattern for error handling. If any step fails after claiming the challenge, a deferred compensation function marks the challenge as failed in the database. This prevents the challenge from being stuck in "pending" state indefinitely.

Rate limit handling is special: when Let's Encrypt returns a rate limit error with a retry-after time, the workflow performs a Restate durable sleep until that time plus a 1-minute buffer (capped at 2 hours), then retries. This uses at most 3 rate limit retries before failing. For transient errors, Restate's standard retry with exponential backoff applies (30s initial, 2x factor, 5m max, 5 attempts).

Returns a response with Status "success" and the certificate ID on success, or Status "failed" with empty certificate ID on failure. System errors return (nil, error).

#### func (Service) RenewExpiringCertificates

```go
func (s *Service) RenewExpiringCertificates(
	ctx restate.ObjectContext,
	req *hydrav1.RenewExpiringCertificatesRequest,
) (*hydrav1.RenewExpiringCertificatesResponse, error)
```

RenewExpiringCertificates renews certificates before they expire.

This handler queries for certificates expiring within 30 days (based on the acme\_challenges table) and triggers \[Service.ProcessChallenge] for each one via fire-and-forget Restate calls. The ProcessChallenge handler handles actual renewal.

This handler is intended to be called on a schedule via GitHub Actions. A 100ms delay is inserted between renewal triggers to avoid overwhelming the system when many certificates need renewal simultaneously.

