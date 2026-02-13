package certmanager

import (
	"context"
	"crypto/tls"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/rpc/vault"
)

type Service interface {
	// GetCertificate returns a certificate for the given domain.
	GetCertificate(ctx context.Context, domain string) (*tls.Certificate, error)
}

type Config struct {
	DB db.Database

	Vault vault.VaultServiceClient

	TLSCertificateCache cache.Cache[string, tls.Certificate]
}
