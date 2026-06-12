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
// the challenge, a deferred compensation function classifies the failure:
//   - Permanent (bad ACME credentials, rate limits exhausted): marked as "failed"
//     so the user knows to intervene. Won't be retried by the cron.
//   - Transient (DNS still propagating, HTTP-01 endpoint not yet reachable): reset
//     to "waiting" so the daily renewal cron retries it automatically.
//
// The error message is recorded on custom_domains.verification_error in both cases
// so the dashboard surfaces what went wrong.
//
// Rate limit handling is special: when Let's Encrypt returns a rate limit error with a
// retry-after time, the workflow performs a Restate durable sleep until that time plus
// a 1-minute buffer (a single sleep is capped at 2 hours), then retries. The loop has
// no overall cap — Let's Encrypt told us when the window opens, so we keep going until
// it does. For transient errors, Restate's standard retry with exponential backoff
// applies (30s initial, 2x factor, 5m max, 5 attempts).
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
		dom, dbErr := db.Query.FindCustomDomainByDomain(stepCtx, s.db.RO(), req.GetDomain())
		if dbErr != nil {
			// Domain was deleted between scheduling and execution; stop retrying.
			if db.IsNotFound(dbErr) {
				return db.CustomDomain{}, restate.TerminalError(fmt.Errorf("custom domain not found: %s", req.GetDomain()), 404)
			}
			return db.CustomDomain{}, dbErr
		}
		return dom, nil
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

	// Compensation: classify the failure so we either give up (bad credentials,
	// rate limits exhausted) or reset the challenge to "waiting" for the daily
	// renewal cron to retry (transient errors like DNS propagation, HTTP-01 not
	// yet reachable). In both cases the message lands on custom_domains.verification_error
	// so the dashboard surfaces what went wrong.
	var (
		lastErrMsg string
		permanent  bool
	)
	defer func() {
		if err == nil && (resp == nil || resp.GetStatus() != "failed") {
			return
		}
		msg := lastErrMsg
		if msg == "" && err != nil {
			msg = err.Error()
		}
		if msg == "" {
			msg = "certificate issuance failed"
		}
		if permanent {
			s.markChallengeFailed(ctx, dom.ID, msg)
		} else {
			s.markChallengeForRetry(ctx, dom.ID, msg)
		}
	}()

	// Step 3: Obtain certificate via DNS-01 challenge
	// Note: ACME client creation happens inside obtainCertificate because lego.Client
	// cannot be serialized through Restate (has internal pointers)
	//
	// Retry policy:
	// - Transient errors: exponential backoff (30s → 60s → 2m → 4m → 5m), max 5 attempts.
	//   After exhaustion the challenge is reset to waiting (see compensation above) so
	//   the daily renewal cron picks it up.
	// - Rate limits: sleep until retry-after, then retry. Let's Encrypt tells us when
	//   the window opens, so we keep going as long as we keep getting rate limited.
	// - Bad credentials: fail immediately (terminal PermanentError → mark failed).
	var cert EncryptedCertificate
	for {
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

		// Permanent failure (bad credentials, etc.) - mark failed, do not retry
		if pe, ok := acme.AsPermanentError(obtainErr); ok {
			lastErrMsg = pe.Error()
			permanent = true
			return &hydrav1.ProcessChallengeResponse{
				CertificateId: "",
				Status:        "failed",
			}, nil
		}

		// Rate limit: Let's Encrypt told us exactly when to retry. Sleep durably
		// (Restate persists the timer) and try again — no artificial cap, since
		// giving up before the window opens would just defer the same retry to
		// the cron without changing the outcome.
		if rle, ok := acme.AsRateLimitError(obtainErr); ok {
			sleepDuration := time.Until(rle.RetryAfter) + time.Minute
			if sleepDuration < time.Minute {
				sleepDuration = time.Minute // minimum 1 minute
			}
			if sleepDuration > 2*time.Hour {
				sleepDuration = 2 * time.Hour // cap a single sleep at 2 hours
			}

			logger.Info("rate limited, sleeping until retry-after",
				"domain", req.GetDomain(),
				"retry_after", rle.RetryAfter,
				"sleep_duration", sleepDuration,
			)

			if err := restate.Sleep(ctx, sleepDuration); err != nil {
				return nil, err
			}

			continue
		}

		// Not a rate limit error - fail (will be retried by daily renewal cron)
		lastErrMsg = obtainErr.Error()
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

	// Step 7: Clear any verification error left over from previous failed attempts.
	// Best-effort: a failure here doesn't invalidate the certificate.
	_, _ = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.UpdateCustomDomainVerificationError(stepCtx, s.db.RW(), db.UpdateCustomDomainVerificationErrorParams{
			ID:                dom.ID,
			VerificationError: sql.NullString{Valid: false},
			UpdatedAt:         sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("clear verification error"))

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

		// Other non-retryable errors (bad credentials): terminal error, no retry.
		// Wrap in PermanentError so the orchestrator can mark the challenge as failed.
		if !parsed.IsRetryable {
			return EncryptedCertificate{}, restate.TerminalError(acme.NewPermanentError(parsed))
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

// maxVerificationErrorLength bounds error messages to the verification_error column width.
const maxVerificationErrorLength = 512

func truncateErrorMsg(msg string) string {
	if len(msg) > maxVerificationErrorLength {
		return msg[:maxVerificationErrorLength]
	}
	return msg
}

// markChallengeForRetry resets the challenge to waiting so the daily renewal cron
// retries it, and records the error message on the custom_domain for UI display.
// Used for transient ACME failures (DNS still propagating, HTTP-01 endpoint not yet
// reachable) where the issue may resolve without user intervention.
func (s *Service) markChallengeForRetry(ctx restate.ObjectContext, domainID, errorMsg string) {
	now := time.Now().UnixMilli()
	errorMsg = truncateErrorMsg(errorMsg)

	_, _ = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		if updateErr := db.Query.UpdateAcmeChallengeStatus(stepCtx, s.db.RW(), db.UpdateAcmeChallengeStatusParams{
			DomainID:  domainID,
			Status:    db.AcmeChallengesStatusWaiting,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: now},
		}); updateErr != nil {
			logger.Error("failed to reset challenge for retry", "error", updateErr, "domain_id", domainID)
		}
		if updateErr := db.Query.UpdateCustomDomainVerificationError(stepCtx, s.db.RW(), db.UpdateCustomDomainVerificationErrorParams{
			ID:                domainID,
			VerificationError: sql.NullString{Valid: true, String: errorMsg},
			UpdatedAt:         sql.NullInt64{Valid: true, Int64: now},
		}); updateErr != nil {
			logger.Error("failed to record verification error", "error", updateErr, "domain_id", domainID)
		}
		return restate.Void{}, nil
	}, restate.WithName("mark for retry"))
}

// markChallengeFailed permanently marks the challenge as failed and records the
// error. Used for errors that won't fix themselves: bad ACME credentials, or
// rate limits that explicitly told us to back off. The user must intervene
// (fix the issue and click Retry, or remove the domain).
func (s *Service) markChallengeFailed(ctx restate.ObjectContext, domainID, errorMsg string) {
	now := time.Now().UnixMilli()
	errorMsg = truncateErrorMsg(errorMsg)

	_, _ = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		if updateErr := db.Query.UpdateAcmeChallengeStatus(stepCtx, s.db.RW(), db.UpdateAcmeChallengeStatusParams{
			DomainID:  domainID,
			Status:    db.AcmeChallengesStatusFailed,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: now},
		}); updateErr != nil {
			logger.Error("failed to mark challenge failed", "error", updateErr, "domain_id", domainID)
		}
		if updateErr := db.Query.UpdateCustomDomainVerificationError(stepCtx, s.db.RW(), db.UpdateCustomDomainVerificationErrorParams{
			ID:                domainID,
			VerificationError: sql.NullString{Valid: true, String: errorMsg},
			UpdatedAt:         sql.NullInt64{Valid: true, Int64: now},
		}); updateErr != nil {
			logger.Error("failed to record verification error", "error", updateErr, "domain_id", domainID)
		}
		return restate.Void{}, nil
	}, restate.WithName("mark failed"))
}
