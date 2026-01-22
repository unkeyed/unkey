// Package certificate implements ACME certificate workflows for SSL/TLS provisioning.
//
// This package orchestrates the complete certificate lifecycle using the ACME protocol,
// coordinating with certificate authorities like Let's Encrypt to validate domain
// ownership and obtain certificates. It supports both HTTP-01 challenges for regular
// domains and DNS-01 challenges for wildcard certificates.
//
// # Why Restate
//
// Certificate issuance involves multiple external dependencies (ACME servers, DNS
// propagation, database writes) that can fail independently. We use Restate for durable
// execution because ACME challenges have strict timing requirements and rate limits.
// If a challenge fails partway through, we cannot simply restart from the beginning
// without risking Let's Encrypt rate limits. Restate's exactly-once execution semantics
// allow the workflow to resume from the last completed step after crashes or network
// failures.
//
// The virtual object model keys workflows by domain name. This prevents race conditions
// where two processes might simultaneously request certificates for the same domain,
// which would trigger ACME duplicate certificate rate limits.
//
// # Key Types
//
// [Service] is the main entry point implementing hydrav1.CertificateServiceServer.
// Configure it via [Config] and create instances with [New]. The service exposes two
// primary handlers: [Service.ProcessChallenge] for obtaining certificates and
// [Service.RenewExpiringCertificates] as a self-scheduling renewal cron job.
//
// [EncryptedCertificate] holds certificate data with the private key encrypted via
// the vault service before database storage.
//
// [BootstrapConfig] configures infrastructure certificate bootstrapping for wildcard
// certificates needed by the platform itself.
//
// # Challenge Types
//
// The service automatically selects the appropriate ACME challenge type based on the
// domain. Wildcard domains (e.g., "*.example.com") require DNS-01 challenges because
// HTTP-01 cannot prove control over all possible subdomains. Regular domains use
// HTTP-01 which is faster since it avoids DNS propagation delays.
//
// # Rate Limit Handling
//
// Let's Encrypt enforces rate limits that cannot be bypassed. When rate limited,
// [Service.ProcessChallenge] uses Restate's durable sleep to wait until the retry-after
// time, then automatically retries. This prevents the workflow from consuming retry
// budget while waiting. Sleep duration is capped at 2 hours with a 1 minute buffer
// added to the retry-after time.
//
// # Security
//
// Private keys are encrypted using the vault service before storage. The encryption
// is workspace-scoped via keyring isolation. Certificates themselves are stored
// unencrypted for fast retrieval by sentinels that terminate TLS.
//
// # Error Handling
//
// The package distinguishes between retryable errors (network timeouts, temporary
// ACME server issues) and terminal errors (invalid credentials, unauthorized domains).
// Terminal errors use restate.TerminalError to prevent infinite retry loops.
// Rate limit errors are handled specially with durable sleeps rather than retries.
package certificate
