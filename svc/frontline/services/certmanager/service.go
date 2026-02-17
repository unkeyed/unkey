package certmanager

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"strings"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

var _ Service = (*service)(nil)

// service provides a basic certificate manager.
type service struct {
	db db.Database

	vault vault.VaultServiceClient

	cache cache.Cache[string, tls.Certificate]
}

// New creates a new certificate manager.
func New(cfg Config) *service {
	return &service{
		db:    cfg.DB,
		cache: cfg.TLSCertificateCache,
		vault: cfg.Vault,
	}
}

func (s *service) GetCertificate(ctx context.Context, domain string) (*tls.Certificate, error) {
	// Build lookup candidates: exact match + immediate wildcard
	// e.g., api.example.com -> [api.example.com, *.example.com]
	candidates := []string{domain}
	parts := strings.SplitN(domain, ".", 2)
	if len(parts) == 2 {
		candidates = append(candidates, "*."+parts[1])
	}

	// SWRWithFallback checks all candidates, returns first hit, and on miss
	// fetches from origin and caches under the canonical key (cert's actual hostname)
	cert, hit, err := s.cache.SWRWithFallback(ctx, candidates, func(ctx context.Context) (tls.Certificate, string, error) {

		rows, err := db.Query.FindCertificatesByHostnames(ctx, s.db.RO(), candidates)
		if err != nil {
			return tls.Certificate{}, "", err
		}

		if len(rows) == 0 {
			return tls.Certificate{}, domain, sql.ErrNoRows
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
			Keyring:   bestRow.WorkspaceID,
			Encrypted: bestRow.EncryptedPrivateKey,
		})
		if err != nil {
			return tls.Certificate{}, "", err
		}

		tlsCert, err := tls.X509KeyPair([]byte(bestRow.Certificate), []byte(pem.GetPlaintext()))
		if err != nil {
			return tls.Certificate{}, "", err
		}

		// Return cert and canonical key (cert's actual hostname for proper cache sharing)
		return tlsCert, bestRow.Hostname, nil
	}, caches.DefaultFindFirstOp)

	if err != nil && !db.IsNotFound(err) {
		logger.Error("Failed to get certificate", "error", err)
		return nil, err
	}

	if hit == cache.Null || db.IsNotFound(err) {
		return nil, errors.New("certificate not found")
	}

	return &cert, nil
}
