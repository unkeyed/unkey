package certmanager

import (
	"context"
	"crypto/tls"

	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/partition/db"
)

var _ Service = (*service)(nil)

// service provides a basic certificate manager.
type service struct {
	// Logger is the logger used to log messages.
	logger logging.Logger

	// DB is the database used to store certificates.
	db db.Database

	cache cache.Cache[string, tls.Certificate]
}

// New creates a new certificate manager.
func New(cfg Config) *service {
	return &service{
		logger: cfg.Logger,
		db:     cfg.DB,
		cache:  cfg.TLSCertificateCache,
	}
}

// GetCertificate implements the CertManager interface.
func (s *service) GetCertificate(ctx context.Context, domain string) (*tls.Certificate, error) {
	cert, hit, err := s.cache.SWR(ctx, domain, func(ctx context.Context) (tls.Certificate, error) {
		return tls.Certificate{}, nil
	}, caches.DefaultFindFirstOp)
	if err != nil {
		if db.IsNotFound(err) {
			// todo: return wrapped 404
			return nil, err
		}

		return nil, err
	}

	if hit == cache.Null {
		// todo: return wrapped 404
		return nil, err
	}

	return &cert, nil
}
