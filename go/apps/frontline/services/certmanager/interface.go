package certmanager

import (
	"context"
	"crypto/tls"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
)

type Service interface {
	// GetCertificate returns a certificate for the given domain.
	GetCertificate(ctx context.Context, domain string) (*tls.Certificate, error)
}

type Config struct {
	Logger logging.Logger

	DB db.Database

	Vault *vault.Service

	TLSCertificateCache cache.Cache[string, tls.Certificate]
}
