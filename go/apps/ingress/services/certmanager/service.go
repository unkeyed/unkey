package certmanager

import (
	"context"
	"crypto/tls"
	"database/sql"
	"log"
	"strings"

	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
)

var _ Service = (*service)(nil)

// service provides a basic certificate manager.
type service struct {
	logger logging.Logger

	db db.Database

	vault *vault.Service

	cache cache.Cache[string, tls.Certificate]
}

// New creates a new certificate manager.
func New(cfg Config) *service {
	return &service{
		logger: cfg.Logger,
		db:     cfg.DB,
		cache:  cfg.TLSCertificateCache,
		vault:  cfg.Vault,
	}
}

func (s *service) GetCertificate(ctx context.Context, domain string) (*tls.Certificate, error) {
	cert, hit, err := s.cache.SWR(ctx, domain, func(ctx context.Context) (tls.Certificate, error) {
		// Build lookup candidates: exact match + immediate wildcard incase we have a wildcard cert available
		candidates := []string{domain}

		// Add wildcard for immediate parent level
		// e.g., api.my.unkey.local -> *.my.unkey.local
		parts := strings.SplitN(domain, ".", 2)
		if len(parts) == 2 {
			candidates = append(candidates, "*."+parts[1])
		}

		rows, err := db.Query.FindCertificatesByHostnames(ctx, s.db.RO(), candidates)
		if err != nil {
			log.Printf("certmanager: db query error domain=%s err=%v", domain, err)
			return tls.Certificate{}, err
		}

		if len(rows) == 0 {
			return tls.Certificate{}, sql.ErrNoRows
		}

		// Prefer exact match over wildcard
		var bestRow db.Certificate
		for _, row := range rows {
			if row.Hostname == domain {
				bestRow = row
				break
			}

			if bestRow.Hostname == "" {
				bestRow = row
			}
		}

		pem, err := s.vault.Decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   "unkey",
			Encrypted: bestRow.EncryptedPrivateKey,
		})
		if err != nil {
			return tls.Certificate{}, err
		}

		cert, err := tls.X509KeyPair([]byte(bestRow.Certificate), []byte(pem.GetPlaintext()))
		if err != nil {
			return tls.Certificate{}, err
		}

		return cert, nil
	}, caches.DefaultFindFirstOp)
	if err != nil {
		s.logger.Error("Failed to get certificate", "error", err)
		return nil, err
	}

	if hit == cache.Null {
		return nil, err
	}

	return &cert, nil
}
