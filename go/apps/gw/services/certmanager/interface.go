package certmanager

import (
	"context"
	"crypto/tls"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/partition/db"
)

type Service interface {
	// GetCertificate returns a certificate for the given domain.
	GetCertificate(ctx context.Context, domain string) (*tls.Certificate, error)
}

type Config struct {
	// Logger is the logger used to log messages.
	Logger logging.Logger

	// DB is the database used to store certificates.
	DB db.Database

	TLSCertificateCache cache.Cache[string, tls.Certificate]

	// DefaultCertDomain is the domain to use for fallback certificate
	// When a domain doesn't have a certificate, use this domain's cert instead
	// This allows handling HTTPS requests for unknown domains
	// Optional - if empty, TLS handshake fails for unknown domains
	DefaultCertDomain string
}
