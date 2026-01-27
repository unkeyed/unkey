package certmanager

import (
	"context"
	"crypto/tls"

	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

type Service interface {
	// GetCertificate returns a certificate for the given domain.
	GetCertificate(ctx context.Context, domain string) (*tls.Certificate, error)
}

type Config struct {
	Logger logging.Logger

	DB db.Database

	Vault vaultv1connect.VaultServiceClient

	TLSCertificateCache cache.Cache[string, tls.Certificate]
}
