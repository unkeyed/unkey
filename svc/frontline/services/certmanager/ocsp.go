package certmanager

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"io"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/db"
	"golang.org/x/crypto/ocsp"
)

const (
	ocspHTTPTimeout     = 5 * time.Second
	ocspMaxResponseSize = 64 * 1024
)

// fetchOCSPStaple fetches OCSP response for the certificate.
// Returns (nil, zero) on failure - graceful degradation.
func (s *service) fetchOCSPStaple(ctx context.Context, cert *tls.Certificate) ([]byte, time.Time) {
	if cert.Leaf == nil || len(cert.Certificate) < 2 {
		return nil, time.Time{}
	}

	if len(cert.Leaf.OCSPServer) == 0 {
		return nil, time.Time{}
	}
	ocspURL := cert.Leaf.OCSPServer[0]

	// Parse issuer (second cert in chain)
	issuer, err := x509.ParseCertificate(cert.Certificate[1])
	if err != nil {
		s.logger.Warn("failed to parse issuer cert", "error", err)
		return nil, time.Time{}
	}

	// Create OCSP request
	ocspReq, err := ocsp.CreateRequest(cert.Leaf, issuer, nil)
	if err != nil {
		s.logger.Warn("failed to create OCSP request", "error", err)
		return nil, time.Time{}
	}

	// Fetch with timeout
	fetchCtx, cancel := context.WithTimeout(ctx, ocspHTTPTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(fetchCtx, http.MethodPost, ocspURL, bytes.NewReader(ocspReq))
	if err != nil {
		s.logger.Warn("failed to create OCSP HTTP request", "error", err)
		return nil, time.Time{}
	}
	httpReq.Header.Set("Content-Type", "application/ocsp-request")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		s.logger.Warn("OCSP fetch failed", "url", ocspURL, "error", err)
		return nil, time.Time{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Warn("OCSP fetch returned non-OK status", "url", ocspURL, "status", resp.StatusCode)
		return nil, time.Time{}
	}

	ocspRespBytes, err := io.ReadAll(io.LimitReader(resp.Body, ocspMaxResponseSize))
	if err != nil {
		s.logger.Warn("failed to read OCSP response", "error", err)
		return nil, time.Time{}
	}

	// Parse and validate
	ocspResp, err := ocsp.ParseResponse(ocspRespBytes, issuer)
	if err != nil {
		s.logger.Warn("failed to parse OCSP response", "error", err)
		return nil, time.Time{}
	}

	if ocspResp.Status != ocsp.Good {
		s.logger.Error("cert OCSP status not good", "status", ocspResp.Status)
	}

	return ocspRespBytes, ocspResp.NextUpdate
}

// refreshOCSPAsync fetches fresh OCSP and updates both the database and cache.
// This runs in a goroutine and should not block the main request path.
func (s *service) refreshOCSPAsync(ctx context.Context, cert *tls.Certificate, hostname string) {
	ocspStaple, expiresAt := s.fetchOCSPStaple(ctx, cert)
	if ocspStaple == nil {
		return
	}

	// Update the certificate with the OCSP staple
	cert.OCSPStaple = ocspStaple

	// Update database for persistence across restarts
	err := db.Query.UpdateCertificateOCSP(ctx, s.db.RW(), db.UpdateCertificateOCSPParams{
		OcspStaple:    ocspStaple,
		OcspExpiresAt: sql.NullInt64{Int64: expiresAt.Unix(), Valid: true},
		UpdatedAt:     sql.NullInt64{Int64: s.clock.Now().Unix(), Valid: true},
		Hostname:      hostname,
	})
	if err != nil {
		s.logger.Warn("failed to update OCSP in DB", "hostname", hostname, "error", err)
		// Continue anyway - cache update is still valuable
	}

	// Update cache directly with the cert that now has OCSP
	s.cache.Set(ctx, hostname, *cert)

	s.logger.Info("OCSP staple refreshed", "hostname", hostname, "expires_at", expiresAt)
}
