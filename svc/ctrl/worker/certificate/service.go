package certificate

import (
	"github.com/go-acme/lego/v4/challenge"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Service orchestrates ACME certificate issuance and renewal.
//
// Service implements hydrav1.CertificateServiceServer with two main handlers:
// [Service.ProcessChallenge] for obtaining/renewing individual certificates, and
// [Service.RenewExpiringCertificates] for batch renewal (called via GitHub Actions).
//
// The service uses a single global ACME account (not per-workspace) to simplify
// key management and avoid hitting Let's Encrypt's account rate limits. Challenge
// type selection is automatic: wildcard domains use DNS-01, regular domains use
// HTTP-01 for faster issuance.
//
// Not safe for concurrent use on the same domain. Concurrency control is handled
// by Restate's virtual object model which keys handlers by domain name.
type Service struct {
	hydrav1.UnimplementedCertificateServiceServer
	db           db.Database
	vault        vault.VaultServiceClient
	emailDomain  string
	dnsProvider  challenge.Provider
	httpProvider challenge.Provider
	heartbeat    healthcheck.Heartbeat
}

var _ hydrav1.CertificateServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service] instance.
type Config struct {
	// DB provides database access for domain, certificate, and ACME challenge records.
	DB db.Database

	// Vault encrypts private keys before database storage. Keys are encrypted using
	// the workspace ID as the keyring identifier.
	Vault vault.VaultServiceClient

	// EmailDomain forms the email address for ACME account registration. The service
	// constructs emails as "acme@{EmailDomain}" for the global ACME account.
	EmailDomain string

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

// New creates a [Service] with the given configuration. The returned service is
// ready to handle certificate requests but does not start any background processes.
// [Service.RenewExpiringCertificates] is intended to be called on a schedule via
// GitHub Actions.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedCertificateServiceServer: hydrav1.UnimplementedCertificateServiceServer{},
		db:                                    cfg.DB,
		vault:                                 cfg.Vault,
		emailDomain:                           cfg.EmailDomain,
		dnsProvider:                           cfg.DNSProvider,
		httpProvider:                          cfg.HTTPProvider,
		heartbeat:                             cfg.Heartbeat,
	}
}
