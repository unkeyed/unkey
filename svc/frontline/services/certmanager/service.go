package certmanager

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/vault"
)

var _ Service = (*service)(nil)

// service provides a basic certificate manager.
type service struct {
	logger logging.Logger

	db db.Database

	vault *vault.Service

	clock clock.Clock

	cache cache.Cache[string, tls.Certificate]

	httpClient *http.Client
}

// New creates a new certificate manager.
func New(cfg Config) *service {
	return &service{
		logger: cfg.Logger,
		db:     cfg.DB,
		cache:  cfg.TLSCertificateCache,
		vault:  cfg.Vault,
		clock:  cfg.Clock,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
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
			Keyring:   "unkey_internal",
			Encrypted: bestRow.EncryptedPrivateKey,
		})
		if err != nil {
			return tls.Certificate{}, "", err
		}

		tlsCert, err := tls.X509KeyPair([]byte(bestRow.Certificate), []byte(pem.GetPlaintext()))
		if err != nil {
			return tls.Certificate{}, "", err
		}

		// Parse and set Leaf to avoid re-parsing on each TLS handshake
		// This saves ~1-2ms per handshake
		if len(tlsCert.Certificate) > 0 {
			leaf, leafErr := x509.ParseCertificate(tlsCert.Certificate[0])
			if leafErr == nil {
				tlsCert.Leaf = leaf
			}
		}

		// Handle OCSP stapling: use from DB if valid, otherwise fetch fresh
		now := s.clock.Now()
		if len(bestRow.OcspStaple) > 0 && bestRow.OcspExpiresAt.Valid &&
			time.Unix(bestRow.OcspExpiresAt.Int64, 0).After(now) {
			// Use cached OCSP from DB - saves clients 50-200ms
			tlsCert.OCSPStaple = bestRow.OcspStaple
		} else if tlsCert.Leaf != nil {
			// Fetch fresh OCSP in background, update DB for next request
			go s.refreshOCSPAsync(context.Background(), &tlsCert, bestRow.Hostname)
		}

		// Return cert and canonical key (cert's actual hostname for proper cache sharing)
		return tlsCert, bestRow.Hostname, nil
	}, caches.DefaultFindFirstOp)

	if err != nil && !db.IsNotFound(err) {
		s.logger.Error("Failed to get certificate", "error", err)
		return nil, err
	}

	if hit == cache.Null || db.IsNotFound(err) {
		return nil, errors.New("certificate not found")
	}

	return &cert, nil
}
