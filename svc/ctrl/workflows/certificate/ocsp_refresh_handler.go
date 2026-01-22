package certificate

import (
	"bytes"
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"io"
	"net/http"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"golang.org/x/crypto/ocsp"
)

const (
	// ocspRefreshInterval is how often the OCSP refresh check runs
	ocspRefreshInterval = 24 * time.Hour

	// ocspHTTPTimeout is the timeout for OCSP HTTP requests
	ocspHTTPTimeout = 5 * time.Second

	ocspMaxResponseSize = 64 * 1024
)

// RefreshExpiringOCSPStaples refreshes OCSP staples for certificates with expiring OCSP.
// This is a self-scheduling Restate cron job - after completing, it schedules itself
// to run again after ocspRefreshInterval (24 hours).
//
// To start the cron job, call this handler once with key "global". It will then
// automatically reschedule itself forever.
func (s *Service) RefreshExpiringOCSPStaples(
	ctx restate.ObjectContext,
	req *hydrav1.RefreshExpiringOCSPStaplesRequest,
) (*hydrav1.RefreshExpiringOCSPStaplesResponse, error) {
	s.logger.Info("starting OCSP staple refresh check")

	// Default to 7 days before expiry
	daysBeforeExpiry := req.GetDaysBeforeExpiry()
	if daysBeforeExpiry <= 0 {
		daysBeforeExpiry = 7
	}

	// Find certificates with OCSP expiring soon (or missing)
	expiryThreshold := time.Now().Add(time.Duration(daysBeforeExpiry) * 24 * time.Hour).Unix()
	certs, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]db.FindCertificatesWithExpiringOCSPRow, error) {
		return db.Query.FindCertificatesWithExpiringOCSP(stepCtx, s.db.RO(), sql.NullInt64{
			Int64: expiryThreshold,
			Valid: true,
		})
	}, restate.WithName("list expiring OCSP"))
	if err != nil {
		return nil, err
	}

	s.logger.Info("found certificates needing OCSP refresh", "count", len(certs))

	var failedHostnames []string
	refreshesCompleted := int32(0)

	for _, cert := range certs {
		// Fetch and update OCSP for this certificate
		success, fetchErr := restate.Run(ctx, func(stepCtx restate.RunContext) (bool, error) {
			return s.refreshOCSPForCertificate(stepCtx, cert)
		}, restate.WithName("refresh OCSP for "+cert.Hostname))

		if fetchErr != nil || !success {
			s.logger.Warn("failed to refresh OCSP",
				"hostname", cert.Hostname,
				"error", fetchErr,
			)
			failedHostnames = append(failedHostnames, cert.Hostname)
		} else {
			refreshesCompleted++
			s.logger.Info("OCSP refreshed", "hostname", cert.Hostname)
		}

		// Small delay between requests to avoid overwhelming OCSP servers
		if err := restate.Sleep(ctx, 100*time.Millisecond); err != nil {
			return nil, err
		}
	}

	s.logger.Info("OCSP staple refresh check completed",
		"checked", len(certs),
		"refreshed", refreshesCompleted,
		"failed", len(failedHostnames),
	)

	// Schedule next run - this creates the Restate cron pattern
	// Use "global" key for singleton cron jobs (same key as certificate renewal)
	nextRunDate := time.Now().Add(ocspRefreshInterval).Format("2006-01-02")
	selfClient := hydrav1.NewCertificateServiceClient(ctx, "global")
	selfClient.RefreshExpiringOCSPStaples().Send(
		&hydrav1.RefreshExpiringOCSPStaplesRequest{
			DaysBeforeExpiry: 7,
		},
		restate.WithDelay(ocspRefreshInterval),
		restate.WithIdempotencyKey("ocsp-refresh-"+nextRunDate),
	)

	s.logger.Info("scheduled next OCSP refresh check", "delay", ocspRefreshInterval)

	return &hydrav1.RefreshExpiringOCSPStaplesResponse{
		CertificatesChecked: int32(len(certs)),
		RefreshesCompleted:  refreshesCompleted,
		FailedHostnames:     failedHostnames,
	}, nil
}

// refreshOCSPForCertificate fetches a fresh OCSP staple and updates the database.
func (s *Service) refreshOCSPForCertificate(ctx context.Context, certRow db.FindCertificatesWithExpiringOCSPRow) (bool, error) {
	// Parse the PEM certificate chain to get leaf and issuer
	// We don't need the private key for OCSP, just the certificate chain
	derCerts, _ := decodePEMCertificateChain([]byte(certRow.Certificate))
	if len(derCerts) < 2 {
		s.logger.Warn("certificate chain too short for OCSP", "hostname", certRow.Hostname)
		return false, nil
	}

	leaf, err := x509.ParseCertificate(derCerts[0])
	if err != nil {
		s.logger.Warn("failed to parse leaf certificate", "hostname", certRow.Hostname, "error", err)
		return false, err
	}

	if len(leaf.OCSPServer) == 0 {
		// No OCSP server specified - this is fine, not all CAs provide OCSP
		return true, nil
	}

	issuer, err := x509.ParseCertificate(derCerts[1])
	if err != nil {
		s.logger.Warn("failed to parse issuer certificate", "hostname", certRow.Hostname, "error", err)
		return false, err
	}

	ocspStaple, expiresAt := s.fetchOCSPStaple(ctx, leaf, issuer)
	if ocspStaple == nil {
		return false, nil
	}

	// Update database
	return true, db.Query.UpdateCertificateOCSP(ctx, s.db.RW(), db.UpdateCertificateOCSPParams{
		OcspStaple:    ocspStaple,
		OcspExpiresAt: sql.NullInt64{Int64: expiresAt.Unix(), Valid: true},
		UpdatedAt:     sql.NullInt64{Int64: time.Now().Unix(), Valid: true},
		Hostname:      certRow.Hostname,
	})
}

// fetchOCSPStaple fetches OCSP response for the certificate.
// Returns (nil, zero) on failure - graceful degradation.
func (s *Service) fetchOCSPStaple(ctx context.Context, leaf, issuer *x509.Certificate) ([]byte, time.Time) {
	if len(leaf.OCSPServer) == 0 {
		return nil, time.Time{}
	}
	ocspURL := leaf.OCSPServer[0]

	// Create OCSP request
	ocspReq, err := ocsp.CreateRequest(leaf, issuer, nil)
	if err != nil {
		s.logger.Warn("failed to create OCSP request", "error", err)
		return nil, time.Time{}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: ocspHTTPTimeout,
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

	resp, err := client.Do(httpReq)
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

// decodePEMCertificateChain parses a PEM-encoded certificate chain and returns
// the DER-encoded certificates.
func decodePEMCertificateChain(pemData []byte) ([][]byte, error) {
	var certs [][]byte
	rest := pemData

	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			certs = append(certs, block.Bytes)
		}
	}

	return certs, nil
}
