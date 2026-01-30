package customdomain

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// Service orchestrates custom domain verification workflows.
//
// Service implements hydrav1.CustomDomainServiceServer with handlers for
// verifying domain ownership via CNAME records. It uses a Restate virtual
// object pattern keyed by domain name to ensure only one verification
// workflow runs per domain at any time.
//
// The verification process checks that the user has added a CNAME record
// pointing to a unique target under the configured DNS apex. Verification
// retries every minute for approximately 24 hours before giving up.
//
// Once verified, the service triggers certificate issuance and creates
// a frontline route so traffic can be routed to the user's deployment.
type Service struct {
	hydrav1.UnimplementedCustomDomainServiceServer
	db      db.Database
	logger  logging.Logger
	dnsApex string
}

var _ hydrav1.CustomDomainServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service] instance.
type Config struct {
	// DB provides database access for custom domain records.
	DB db.Database

	// Logger receives structured log output from domain verification operations.
	Logger logging.Logger

	// DnsApex is the base domain for custom domain CNAME targets.
	// Each custom domain gets a unique subdomain like "{random}.{DnsApex}".
	// For production: "cname.unkey-dns.com"
	// For local: "cname.unkey.local"
	DnsApex string
}

// New creates a [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedCustomDomainServiceServer: hydrav1.UnimplementedCustomDomainServiceServer{},
		db:                                     cfg.DB,
		logger:                                 cfg.Logger,
		dnsApex:                                cfg.DnsApex,
	}
}
