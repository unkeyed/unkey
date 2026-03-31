package customdomain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// Service orchestrates custom domain verification workflows.
//
// Service implements hydrav1.CustomDomainServiceServer with handlers for
// verifying domain ownership via DNS records. It uses a Restate virtual
// object pattern keyed by domain ID to ensure only one verification
// workflow runs per domain at any time.
//
// The verification process checks that the user has added a CNAME record
// pointing to a unique target under the configured DNS apex. For apex
// domains where CNAME is not visible (CNAME flattening, ALIAS, proxy),
// it falls back to TXT record verification using an HMAC-derived token.
//
// Verification retries every minute for approximately 24 hours before
// giving up. Once verified, the service triggers certificate issuance
// and creates a frontline route so traffic can be routed to the user's
// deployment.
type Service struct {
	hydrav1.UnimplementedCustomDomainServiceServer
	db               db.Database
	cnameDomain      string
	domainSigningKey []byte
}

var _ hydrav1.CustomDomainServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service] instance.
type Config struct {
	// DB provides database access for custom domain records.
	DB db.Database

	// CnameDomain is the base domain for custom domain CNAME targets.
	CnameDomain string

	// DomainSigningKey is used to compute HMAC-based verification tokens
	// for TXT record ownership verification. The token is derived from the
	// domain ID so it doesn't need to be stored in the database.
	DomainSigningKey string
}

// New creates a [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedCustomDomainServiceServer: hydrav1.UnimplementedCustomDomainServiceServer{},
		db:                                     cfg.DB,
		cnameDomain:                            cfg.CnameDomain,
		domainSigningKey:                       []byte(cfg.DomainSigningKey),
	}
}

// verificationToken computes a deterministic verification token for a domain
// using HMAC-SHA256. The token is derived from the domain ID so it doesn't
// need to be stored in the database.
func (s *Service) verificationToken(domainID string) string {
	mac := hmac.New(sha256.New, s.domainSigningKey)
	mac.Write([]byte(domainID))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}
