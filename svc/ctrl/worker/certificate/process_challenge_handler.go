package certificate

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/services/acme"
)

// EncryptedCertificate holds a certificate with its private key encrypted for storage.
// The private key is encrypted using the vault service with the workspace ID as the
// keyring, ensuring keys can only be decrypted by the owning workspace.
type EncryptedCertificate struct {
	// CertificateID is the unique identifier for this certificate, generated using
	// uid.New with the certificate prefix.
	CertificateID string

	// Certificate contains the PEM-encoded certificate chain including intermediates.
	Certificate string

	// EncryptedPrivateKey is the vault-encrypted PEM-encoded private key.
	EncryptedPrivateKey string

	// ExpiresAt is the certificate expiration time as Unix milliseconds. Parsed from
	// the certificate's NotAfter field; defaults to 90 days from issuance if parsing
	// fails.
	ExpiresAt int64
}

// ProcessChallenge obtains or renews an SSL/TLS certificate for a domain.
//
// This is a Restate virtual object handler keyed by domain name, ensuring only one
// certificate challenge runs per domain at any time. The workflow consists of durable
// steps that survive process restarts: domain resolution, challenge claiming, certificate
// obtainment, persistence, and verification marking.
//
// The method uses the saga pattern for error handling. If any step fails after claiming
// the challenge, a deferred compensation function marks the challenge as failed in the
// database. This prevents the challenge from being stuck in "pending" state indefinitely.
//
// Rate limit handling is special: when Let's Encrypt returns a rate limit error with a
// retry-after time, the workflow performs a Restate durable sleep until that time plus
// a 1-minute buffer (capped at 2 hours), then retries. This uses at most 3 rate limit
// retries before failing. For transient errors, Restate's standard retry with exponential
// backoff applies (30s initial, 2x factor, 5m max, 5 attempts).
//
// Returns a response with Status "success" and the certificate ID on success, or Status
// "failed" with empty certificate ID on failure. System errors return (nil, error).
func (s *Service) ProcessChallenge(
	ctx restate.ObjectContext,
	req *hydrav1.ProcessChallengeRequest,
) (resp *hydrav1.ProcessChallengeResponse, err error) {
	logger.Info("starting certificate challenge",
		"workspace_id", req.GetWorkspaceId(),
		"domain", req.GetDomain(),
	)

	// Step 1: Resolve domain
	dom, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.CustomDomain, error) {
		return db.Query.FindCustomDomainByDomain(stepCtx, s.db.RO(), req.GetDomain())
	}, restate.WithName("resolve domain"))
	if err != nil {
		return nil, err
	}

	// Step 2: Claim the challenge
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.UpdateAcmeChallengeTryClaiming(stepCtx, s.db.RW(), db.UpdateAcmeChallengeTryClaimingParams{
			DomainID:  dom.ID,
			Status:    db.AcmeChallengesStatusPending,
			UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		})
	}, restate.WithName("claim challenge"))
	if err != nil {
		return nil, err
	}

	// Compensation: if anything fails after claiming, mark challenge as failed
	defer func() {
		if err != nil || (resp != nil && resp.GetStatus() == "failed") {
			s.markChallengeFailed(ctx, dom.ID)
		}
	}()

	// Step 3: Obtain certificate via DNS-01 challenge
	// Note: ACME client creation happens inside obtainCertificate because lego.Client
	// cannot be serialized through Restate (has internal pointers)
	//
	// Retry policy:
	// - Transient errors: exponential backoff (30s → 60s → 2m → 4m → 5m), max 5 attempts
	// - Rate limits: sleep until retry-after time, then retry (max 3 rate limit retries)
	// - Bad credentials: fail immediately (terminal error)
	var cert EncryptedCertificate
	maxRateLimitRetries := 3
	for rateLimitRetry := 0; rateLimitRetry <= maxRateLimitRetries; rateLimitRetry++ {
		var obtainErr error
		cert, obtainErr = restate.Run(ctx, func(stepCtx restate.RunContext) (EncryptedCertificate, error) {
			return s.obtainCertificate(stepCtx, req.GetWorkspaceId(), dom, req.GetDomain())
		},
			restate.WithName("obtain certificate"),
			restate.WithMaxRetryAttempts(5),
			restate.WithInitialRetryInterval(30*time.Second),
			restate.WithMaxRetryInterval(5*time.Minute),
			restate.WithRetryIntervalFactor(2.0),
		)

		if obtainErr == nil {
			break // Success!
		}

		// Check if it's a rate limit error
		if rle, ok := acme.AsRateLimitError(obtainErr); ok {
			if rateLimitRetry >= maxRateLimitRetries {
				logger.Error("max rate limit retries exceeded",
					"domain", req.GetDomain(),
					"retries", rateLimitRetry,
				)
				return &hydrav1.ProcessChallengeResponse{
					CertificateId: "",
					Status:        "failed",
				}, nil
			}

			// Calculate sleep duration until retry-after (with 1 min buffer)
			sleepDuration := time.Until(rle.RetryAfter) + time.Minute
			if sleepDuration < time.Minute {
				sleepDuration = time.Minute // minimum 1 minute
			}
			if sleepDuration > 2*time.Hour {
				sleepDuration = 2 * time.Hour // cap at 2 hours
			}

			logger.Info("rate limited, sleeping until retry-after",
				"domain", req.GetDomain(),
				"retry_after", rle.RetryAfter,
				"sleep_duration", sleepDuration,
				"rate_limit_retry", rateLimitRetry+1,
			)

			// Durable sleep - Restate will wake us up
			if err := restate.Sleep(ctx, sleepDuration); err != nil {
				return nil, err
			}

			continue // Retry after sleep
		}

		// Not a rate limit error - fail
		return &hydrav1.ProcessChallengeResponse{
			CertificateId: "",
			Status:        "failed",
		}, nil
	}

	// Step 5: Persist certificate to DB
	certID, err := restate.Run(ctx, func(stepCtx restate.RunContext) (string, error) {
		return s.persistCertificate(stepCtx, dom, req.GetDomain(), cert)
	}, restate.WithName("persist certificate"))
	if err != nil {
		return nil, err
	}

	// Step 6: Mark challenge as verified
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.UpdateAcmeChallengeVerifiedWithExpiry(stepCtx, s.db.RW(), db.UpdateAcmeChallengeVerifiedWithExpiryParams{
			Status:    db.AcmeChallengesStatusVerified,
			ExpiresAt: cert.ExpiresAt,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			DomainID:  dom.ID,
		})
	}, restate.WithName("mark verified"))
	if err != nil {
		return nil, err
	}

	logger.Info("certificate challenge completed successfully",
		"domain", req.GetDomain(),
		"certificate_id", certID,
		"expires_at", cert.ExpiresAt,
	)

	return &hydrav1.ProcessChallengeResponse{
		CertificateId: certID,
		Status:        "success",
	}, nil
}

// globalAcmeUserID identifies the shared ACME account used for all certificate
// requests to avoid per-workspace account creation and stay under account limits.
const globalAcmeUserID = "acme"

// isWildcard reports whether domain is a wildcard domain pattern. Wildcard domains
// start with "*." and require DNS-01 challenges since HTTP-01 cannot validate control
// over arbitrary subdomains.
func isWildcard(domain string) bool {
	return len(domain) > 2 && domain[0] == '*' && domain[1] == '.'
}

// getOrCreateAcmeClient returns a configured ACME client for the domain's
// challenge type using the shared account.
func (s *Service) getOrCreateAcmeClient(ctx context.Context, domain string) (*lego.Client, error) {
	// Use a single global ACME user for all certificates
	client, err := acme.GetOrCreateUser(ctx, acme.UserConfig{
		DB:          s.db,
		Vault:       s.vault,
		WorkspaceID: globalAcmeUserID,
		EmailDomain: s.emailDomain,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get/create ACME user: %w", err)
	}

	// Wildcard certificates require DNS-01 challenge
	// Regular domains use HTTP-01 (faster, no DNS propagation wait)
	if isWildcard(domain) {
		if s.dnsProvider == nil {
			return nil, fmt.Errorf("DNS provider required for wildcard certificate: %s", domain)
		}
		if err := client.Challenge.SetDNS01Provider(s.dnsProvider); err != nil {
			return nil, fmt.Errorf("failed to set DNS-01 provider: %w", err)
		}
		logger.Info("using DNS-01 challenge for wildcard domain", "domain", domain)
	} else {
		if s.httpProvider == nil {
			return nil, fmt.Errorf("HTTP provider required for certificate: %s", domain)
		}
		if err := client.Challenge.SetHTTP01Provider(s.httpProvider); err != nil {
			return nil, fmt.Errorf("failed to set HTTP-01 provider: %w", err)
		}
		logger.Info("using HTTP-01 challenge for domain", "domain", domain)
	}

	return client, nil
}

// obtainCertificate requests a certificate and encrypts the private key for storage.
func (s *Service) obtainCertificate(ctx context.Context, _ string, dom db.CustomDomain, domain string) (EncryptedCertificate, error) {
	logger.Info("creating ACME client", "domain", domain)
	client, err := s.getOrCreateAcmeClient(ctx, domain)
	if err != nil {
		return EncryptedCertificate{}, fmt.Errorf("failed to create ACME client: %w", err)
	}
	logger.Info("ACME client created, requesting certificate", "domain", domain)

	// Request certificate from Let's Encrypt
	// Restate handles retries - we return TerminalError for non-retryable errors
	//nolint:exhaustruct // external library type
	request := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		parsed := acme.ParseACMEError(err)
		logger.Error("certificate obtain failed",
			"domain", domain,
			"error_type", parsed.Type,
			"message", parsed.Message,
			"is_retryable", parsed.IsRetryable,
			"retry_after", parsed.RetryAfter,
		)

		// Rate limits: return RateLimitError so the handler can sleep and retry
		if parsed.Type == acme.ACMEErrorRateLimited {
			return EncryptedCertificate{}, acme.NewRateLimitError(parsed)
		}

		// Other non-retryable errors (bad credentials): terminal error, no retry
		if !parsed.IsRetryable {
			return EncryptedCertificate{}, restate.TerminalError(
				fmt.Errorf("[%s] %s", parsed.Type, parsed.Message),
			)
		}

		// Retryable error - Restate will retry with backoff
		return EncryptedCertificate{}, fmt.Errorf("failed to obtain certificate: %w", err)
	}

	// Parse certificate to get expiration
	expiresAt, err := acme.GetCertificateExpiry(certificates.Certificate)
	if err != nil {
		logger.Warn("failed to parse certificate expiry, using default", "error", err)
		expiresAt = time.Now().Add(90 * 24 * time.Hour).UnixMilli()
	}

	// Encrypt the private key before storage
	encryptResp, err := s.vault.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: dom.WorkspaceID,
		Data:    string(certificates.PrivateKey),
	})
	if err != nil {
		return EncryptedCertificate{}, fmt.Errorf("failed to encrypt private key: %w", err)
	}

	return EncryptedCertificate{
		CertificateID:       uid.New(uid.CertificatePrefix),
		Certificate:         string(certificates.Certificate),
		EncryptedPrivateKey: encryptResp.GetEncrypted(),
		ExpiresAt:           expiresAt,
	}, nil
}

// persistCertificate stores the certificate and reuses the existing ID on renewals.
func (s *Service) persistCertificate(ctx context.Context, dom db.CustomDomain, domain string, cert EncryptedCertificate) (string, error) {
	now := time.Now().UnixMilli()

	// Check if certificate already exists for this hostname (renewal case)
	// If it does, we keep the existing ID; otherwise use the new ID
	certID := cert.CertificateID
	existingCert, err := db.Query.FindCertificateByHostname(ctx, s.db.RO(), domain)
	if err != nil && !db.IsNotFound(err) {
		return "", fmt.Errorf("failed to check for existing certificate: %w", err)
	}
	if err == nil {
		// Renewal: keep the existing certificate ID
		certID = existingCert.ID
	}

	// InsertCertificate uses ON DUPLICATE KEY UPDATE, so this handles both insert and renewal
	err = db.Query.InsertCertificate(ctx, s.db.RW(), db.InsertCertificateParams{
		ID:                  certID,
		WorkspaceID:         dom.WorkspaceID,
		Hostname:            domain,
		Certificate:         cert.Certificate,
		EncryptedPrivateKey: cert.EncryptedPrivateKey,
		CreatedAt:           now,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: now},
	})
	if err != nil {
		return "", fmt.Errorf("failed to persist certificate: %w", err)
	}

	return certID, nil
}

// markChallengeFailed marks a challenge as failed during cleanup.
func (s *Service) markChallengeFailed(ctx restate.ObjectContext, domainID string) {
	_, _ = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		if updateErr := db.Query.UpdateAcmeChallengeStatus(stepCtx, s.db.RW(), db.UpdateAcmeChallengeStatusParams{
			DomainID:  domainID,
			Status:    db.AcmeChallengesStatusFailed,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		}); updateErr != nil {
			logger.Error("failed to update challenge status", "error", updateErr, "domain_id", domainID)
		}
		return restate.Void{}, nil
	}, restate.WithName("mark failed"))
}
