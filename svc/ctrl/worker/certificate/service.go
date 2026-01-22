package certificate

import (
	"github.com/go-acme/lego/v4/challenge"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/vault"
)

// Service handles ACME certificate challenge workflows.
//
// This service orchestrates the complete certificate issuance process including
// domain validation, challenge claiming, ACME protocol communication, and certificate
// storage. It implements the hydrav1.CertificateServiceServer interface.
//
// The service uses Restate virtual objects keyed by domain name to ensure that only
// one certificate challenge runs per domain at any time, preventing duplicate requests
// and rate limit violations.
type Service struct {
	hydrav1.UnimplementedCertificateServiceServer
	db            db.Database
	vault         *vault.Service
	logger        logging.Logger
	emailDomain   string
	defaultDomain string
	dnsProvider   challenge.Provider // For DNS-01 challenges (wildcard certs)
	httpProvider  challenge.Provider // For HTTP-01 challenges (regular certs)
}

var _ hydrav1.CertificateServiceServer = (*Service)(nil)

// Config holds the configuration for creating a certificate service.
type Config struct {
	// DB is the main database connection for workspace and domain data.
	DB db.Database

	// Vault provides encryption services for private key storage.
	Vault *vault.Service

	// Logger for structured logging.
	Logger logging.Logger

	// EmailDomain is the domain used for ACME account emails (workspace_id@domain)
	EmailDomain string

	// DefaultDomain is the base domain for wildcard certificates
	DefaultDomain string

	// DNSProvider is the challenge provider for DNS-01 challenges (wildcard certs)
	DNSProvider challenge.Provider

	// HTTPProvider is the challenge provider for HTTP-01 challenges (regular certs)
	HTTPProvider challenge.Provider
}

// New creates a new certificate service instance.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedCertificateServiceServer: hydrav1.UnimplementedCertificateServiceServer{},
		db:                                    cfg.DB,
		vault:                                 cfg.Vault,
		logger:                                cfg.Logger,
		emailDomain:                           cfg.EmailDomain,
		defaultDomain:                         cfg.DefaultDomain,
		dnsProvider:                           cfg.DNSProvider,
		httpProvider:                          cfg.HTTPProvider,
	}
}
