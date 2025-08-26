package certmanager

import (
	"context"
	"crypto/tls"
	"strings"

	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/partition/db"
	"github.com/unkeyed/unkey/go/pkg/vault"
)

var _ Service = (*service)(nil)

// service provides a basic certificate manager.
type service struct {
	// Logger is the logger used to log messages.
	logger logging.Logger

	// DB is the database used to store certificates.
	db db.Database

	// Vault is the vault service used to store certificates.
	vault *vault.Service

	cache cache.Cache[string, tls.Certificate]

	// DefaultCertDomain is the domain to use for fallback certificate
	// When a domain has no cert, use this domain's cert instead
	defaultCertDomain string
}

// New creates a new certificate manager.
func New(cfg Config) *service {
	return &service{
		logger:            cfg.Logger,
		db:                cfg.DB,
		cache:             cfg.TLSCertificateCache,
		defaultCertDomain: cfg.DefaultCertDomain,
		vault:             cfg.Vault,
	}
}

// GetCertificate implements the CertManager interface.
func (s *service) GetCertificate(ctx context.Context, domain string) (*tls.Certificate, error) {
	if strings.HasSuffix(domain, s.defaultCertDomain) && domain != "*."+s.defaultCertDomain {
		return s.GetCertificate(ctx, "*."+s.defaultCertDomain)
	}

	cert, hit, err := s.cache.SWR(ctx, domain, func(ctx context.Context) (tls.Certificate, error) {
		row, err := db.Query.FindCertificateByHostname(ctx, s.db.RO(), domain)
		if err != nil {
			return tls.Certificate{}, err
		}

		// todo: handle vault

		// For non production use aka development
		cert, err := tls.X509KeyPair([]byte(row.Certificate), []byte(row.EncryptedPrivateKey))

		return cert, nil
	}, caches.DefaultFindFirstOp)
	if err != nil {
		// if db.IsNotFound(err) {
		// // If we have a default cert domain configured, try to fetch that cert
		// if s.defaultCertDomain != "" && domain != s.defaultCertDomain {
		// 	s.logger.Warn("certificate not found, trying default certificate",
		// 		"domain", domain,
		// 		"defaultDomain", s.defaultCertDomain,
		// 	)
		// 	// Recursively call GetCertificate with the default domain
		// 	// This will use caching properly and fetch from DB if needed
		// 	return s.GetCertificate(ctx, s.defaultCertDomain)
		// }

		// No default cert domain or the default domain itself has no cert
		// 	return nil, err
		// }

		return nil, err
	}

	if hit == cache.Null {
		// todo: return wrapped 404
		return nil, err
	}

	return &cert, nil
}
