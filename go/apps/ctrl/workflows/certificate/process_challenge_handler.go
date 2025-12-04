package certificate

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/go/apps/ctrl/services/acme"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/retry"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// EncryptedCertificate holds a certificate and its encrypted private key.
type EncryptedCertificate struct {
	CertificateID       string
	Certificate         string
	EncryptedPrivateKey string
	ExpiresAt           int64
}

// ProcessChallenge handles the complete ACME certificate challenge flow.
//
// This method implements a multi-step durable workflow using Restate to obtain or renew
// an SSL/TLS certificate for a domain. Each step is wrapped in restate.Run for durability,
// allowing the workflow to resume from the last completed step if interrupted.
//
// Uses the saga pattern: if any step fails after claiming the challenge, the deferred
// compensation marks the challenge as failed.
func (s *Service) ProcessChallenge(
	ctx restate.ObjectContext,
	req *hydrav1.ProcessChallengeRequest,
) (resp *hydrav1.ProcessChallengeResponse, err error) {
	s.logger.Info("starting certificate challenge",
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
		if err != nil || (resp != nil && resp.Status == "failed") {
			s.markChallengeFailed(ctx, dom.ID)
		}
	}()

	// Step 3: Obtain certificate via DNS-01 challenge
	// Note: ACME client creation happens inside obtainCertificate because lego.Client
	// cannot be serialized through Restate (has internal pointers)
	cert, err := restate.Run(ctx, func(stepCtx restate.RunContext) (EncryptedCertificate, error) {
		return s.obtainCertificate(stepCtx, req.GetWorkspaceId(), dom, req.GetDomain())
	}, restate.WithName("obtain certificate"))
	if err != nil {
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

	s.logger.Info("certificate challenge completed successfully",
		"domain", req.GetDomain(),
		"certificate_id", certID,
		"expires_at", cert.ExpiresAt,
	)

	return &hydrav1.ProcessChallengeResponse{
		CertificateId: certID,
		Status:        "success",
	}, nil
}

// globalAcmeUserID is the fixed user ID for the single global ACME account
const globalAcmeUserID = "acme"

func (s *Service) getOrCreateAcmeClient(ctx context.Context, _ string) (*lego.Client, error) {
	// Use a single global ACME user for all certificates
	client, err := acme.GetOrCreateUser(ctx, acme.UserConfig{
		DB:          s.db,
		Logger:      s.logger,
		Vault:       s.vault,
		WorkspaceID: globalAcmeUserID,
		EmailDomain: s.emailDomain,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get/create ACME user: %w", err)
	}

	if s.dnsProvider != nil {
		if err := client.Challenge.SetDNS01Provider(s.dnsProvider); err != nil {
			return nil, fmt.Errorf("failed to set DNS-01 provider: %w", err)
		}
	}

	return client, nil
}

func (s *Service) obtainCertificate(ctx context.Context, workspaceID string, dom db.CustomDomain, domain string) (EncryptedCertificate, error) {
	// Create ACME client inside this function because lego.Client cannot be
	// serialized through Restate (contains internal pointers that become nil)
	s.logger.Info("creating ACME client", "domain", domain)
	client, err := s.getOrCreateAcmeClient(ctx, workspaceID)
	if err != nil {
		return EncryptedCertificate{}, fmt.Errorf("failed to create ACME client: %w", err)
	}
	s.logger.Info("ACME client created, requesting certificate", "domain", domain)

	// Request certificate from Let's Encrypt with retry and exponential backoff
	request := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}

	var certificates *certificate.Resource
	retrier := retry.New(
		retry.Attempts(3),
		retry.Backoff(func(attempt int) time.Duration {
			// Exponential backoff: 30s, 60s, 120s (capped at 5min)
			delay := time.Duration(30<<attempt) * time.Second
			if delay > 5*time.Minute {
				delay = 5 * time.Minute
			}
			return delay
		}),
	)

	err = retrier.Do(func() error {
		var obtainErr error
		certificates, obtainErr = client.Certificate.Obtain(request)
		return obtainErr
	})
	if err != nil {
		return EncryptedCertificate{}, fmt.Errorf("failed to obtain certificate after retries: %w", err)
	}

	// Parse certificate to get expiration
	expiresAt, err := acme.GetCertificateExpiry(certificates.Certificate)
	if err != nil {
		s.logger.Warn("failed to parse certificate expiry, using default", "error", err)
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

func (s *Service) persistCertificate(ctx context.Context, dom db.CustomDomain, domain string, cert EncryptedCertificate) (string, error) {
	now := time.Now().UnixMilli()

	// InsertCertificate uses ON DUPLICATE KEY UPDATE, so this handles both insert and renewal
	err := db.Query.InsertCertificate(ctx, s.db.RW(), db.InsertCertificateParams{
		ID:                  cert.CertificateID,
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

	return cert.CertificateID, nil
}

func (s *Service) markChallengeFailed(ctx restate.ObjectContext, domainID string) {
	_, _ = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		if updateErr := db.Query.UpdateAcmeChallengeStatus(stepCtx, s.db.RW(), db.UpdateAcmeChallengeStatusParams{
			DomainID:  domainID,
			Status:    db.AcmeChallengesStatusFailed,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		}); updateErr != nil {
			s.logger.Error("failed to update challenge status", "error", updateErr, "domain_id", domainID)
		}
		return restate.Void{}, nil
	}, restate.WithName("mark failed"))
}
