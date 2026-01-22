package certmanager

import (
	"context"
	"crypto/tls"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/vault"
)

type Service interface {
	// GetCertificate returns a certificate for the given domain.
	GetCertificate(ctx context.Context, domain string) (*tls.Certificate, error)
}

type Config struct {
	Logger logging.Logger

	DB db.Database

	Vault *vault.Service

	Clock clock.Clock

	TLSCertificateCache cache.Cache[string, tls.Certificate]
}
