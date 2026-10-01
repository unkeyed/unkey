package certificate

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
	"github.com/unkeyed/unkey/svc/ctrl/services/acme"
)

// EncryptedCertificate holds a certificate with its private key encrypted for storage.
type EncryptedCertificate struct {
	CertificateID       string
	Certificate         string
	EncryptedPrivateKey string
	// ExpiresAt is the certificate's NotAfter as Unix milliseconds.
	ExpiresAt int64
}

// ProcessChallenge obtains or renews an SSL/TLS certificate for a domain.
func (s *Service) ProcessChallenge(ctx restate.ObjectContext, req *hydrav1.ProcessChallengeRequest) (resp *hydrav1.ProcessChallengeResponse, err error) {
	domain, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.CustomDomain, error) {
		return s.db.FindCustomDomainByDomain(stepCtx, req.GetDomain())
	}, restate.WithName("resolve domain"))
	if err != nil {
		return nil, err
	}

	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, s.db.UpdateAcmeChallengeTryClaiming(stepCtx, db.UpdateAcmeChallengeTryClaimingParams{
			DomainID:  domain.ID,
			Status:    db.AcmeChallengesStatusPending,
			UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		})
	}, restate.WithName("claim challenge"))
	if err != nil {
		return nil, err
	}

	defer func() {
		if err == nil {
			return
		}
		ctx.Log().Error("certificate challenge failed", "domain", domain.Domain, "error", err)
		if cleanupErr := s.markChallengeFailed(ctx, domain.ID); cleanupErr != nil {
			ctx.Log().Error("failed to mark certificate challenge failed", "domain", domain.Domain, "error", cleanupErr)
		}
	}()

	cert, err := s.obtainWithRetry(ctx, domain)
	if err != nil {
		return nil, err
	}

	certID, err := restate.Run(ctx, func(stepCtx restate.RunContext) (string, error) {
		return s.persistCertificate(stepCtx, domain, cert)
	}, restate.WithName("persist certificate"))
	if err != nil {
		return nil, err
	}

	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		err := s.db.UpdateAcmeChallengeVerifiedWithExpiry(stepCtx, db.UpdateAcmeChallengeVerifiedWithExpiryParams{
			Status:    db.AcmeChallengesStatusVerified,
			ExpiresAt: cert.ExpiresAt,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			DomainID:  domain.ID,
		})
		if err != nil {
			return restate.Void{}, err
		}
		metrics.CertificateChallengesTotal.WithLabelValues("verified").Inc()
		return restate.Void{}, nil
	}, restate.WithName("mark verified"))
	if err != nil {
		return nil, err
	}

	ctx.Log().Info("certificate challenge completed",
		"domain", domain.Domain,
		"certificate_id", certID,
		"expires_at", cert.ExpiresAt,
	)

	return &hydrav1.ProcessChallengeResponse{
		CertificateId: certID,
	}, nil
}

type certificateAttempt struct {
	EncryptedCertificate
	RetryDelay time.Duration
}

func (s *Service) obtainWithRetry(ctx restate.ObjectContext, domain db.CustomDomain) (EncryptedCertificate, error) {
	const maxRateLimitRetries = 3
	for attempt := 0; ; attempt++ {
		result, err := restate.Run(ctx, func(stepCtx restate.RunContext) (certificateAttempt, error) {
			client, err := s.newACMEClient(stepCtx, domain.Domain)
			if err != nil {
				return certificateAttempt{}, err
			}
			return s.obtainCertificate(stepCtx, client, domain)
		},
			restate.WithName("obtain certificate"),
			restate.WithMaxRetryAttempts(5),
			restate.WithInitialRetryInterval(30*time.Second),
			restate.WithMaxRetryInterval(5*time.Minute),
			restate.WithRetryIntervalFactor(2.0),
		)
		if err != nil {
			return EncryptedCertificate{}, err
		}
		if result.RetryDelay == 0 {
			return result.EncryptedCertificate, nil
		}
		if attempt == maxRateLimitRetries {
			return EncryptedCertificate{}, restate.TerminalErrorf("certificate issuance remained rate limited after %d attempts", attempt+1)
		}
		if err := restate.Sleep(ctx, result.RetryDelay); err != nil {
			return EncryptedCertificate{}, err
		}
	}
}

const globalACMEUserID = "acme"

func (s *Service) newACMEClient(ctx context.Context, domain string) (*lego.Client, error) {
	client, err := acme.GetOrCreateUser(ctx, acme.UserConfig{
		DB:          s.db,
		Vault:       s.vault,
		WorkspaceID: globalACMEUserID,
		EmailDomain: s.emailDomain,
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("get or create ACME user"))
	}

	if strings.HasPrefix(domain, "*.") {
		if s.dnsProvider == nil {
			return nil, restate.TerminalErrorf("DNS provider required for wildcard certificate %q", domain)
		}
		if err := client.Challenge.SetDNS01Provider(s.dnsProvider); err != nil {
			return nil, fault.Wrap(err, fault.Internal("set DNS-01 provider"))
		}
		return client, nil
	}
	if s.httpProvider == nil {
		return nil, restate.TerminalErrorf("HTTP provider required for certificate %q", domain)
	}
	if err := client.Challenge.SetHTTP01Provider(s.httpProvider); err != nil {
		return nil, fault.Wrap(err, fault.Internal("set HTTP-01 provider"))
	}
	return client, nil
}

func (s *Service) obtainCertificate(ctx restate.RunContext, client *lego.Client, domain db.CustomDomain) (certificateAttempt, error) {
	var result certificateAttempt
	//nolint:exhaustruct // external library type
	request := certificate.ObtainRequest{
		Domains: []string{domain.Domain},
		Bundle:  true,
	}
	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		parsed := acme.ParseACMEError(err)
		metrics.CertificateIssuanceAttemptsTotal.WithLabelValues(string(parsed.Type)).Inc()
		if parsed.Type == acme.ACMEErrorRateLimited {
			result.RetryDelay = min(2*time.Hour, max(time.Minute, time.Until(parsed.RetryAfter)+time.Minute))
			ctx.Log().Warn("certificate issuance rate limited",
				"domain", domain.Domain,
				"retry_after", parsed.RetryAfter,
				"sleep_duration", result.RetryDelay,
				"error", parsed,
			)
			return result, nil
		}
		if !parsed.IsRetryable {
			return certificateAttempt{}, restate.ToTerminalError(parsed)
		}
		return certificateAttempt{}, fault.Wrap(err, fault.Internal("obtain certificate"))
	}
	metrics.CertificateIssuanceAttemptsTotal.WithLabelValues("issued").Inc()

	expiresAt, err := acme.GetCertificateExpiry(certificates.Certificate)
	if err != nil {
		return certificateAttempt{}, restate.ToTerminalError(fault.Wrap(err, fault.Internal("parse certificate expiry")))
	}

	encryptResp, err := s.vault.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: domain.WorkspaceID,
		Data:    string(certificates.PrivateKey),
	})
	if err != nil {
		return certificateAttempt{}, fault.Wrap(err, fault.Internal("encrypt private key"))
	}

	result.EncryptedCertificate = EncryptedCertificate{
		CertificateID:       uid.New(uid.CertificatePrefix),
		Certificate:         string(certificates.Certificate),
		EncryptedPrivateKey: encryptResp.GetEncrypted(),
		ExpiresAt:           expiresAt,
	}
	return result, nil
}

func (s *Service) persistCertificate(ctx context.Context, domain db.CustomDomain, cert EncryptedCertificate) (string, error) {
	certID := cert.CertificateID
	existingCert, err := s.db.FindCertificateByHostname(ctx, domain.Domain)
	if err != nil && !db.IsNotFound(err) {
		return "", fault.Wrap(err, fault.Internal("find existing certificate"))
	}
	if err == nil {
		certID = existingCert.ID
	}

	now := time.Now().UnixMilli()
	err = s.db.InsertCertificate(ctx, db.InsertCertificateParams{
		ID:                  certID,
		WorkspaceID:         domain.WorkspaceID,
		Hostname:            domain.Domain,
		Certificate:         cert.Certificate,
		EncryptedPrivateKey: cert.EncryptedPrivateKey,
		CreatedAt:           now,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: now},
	})
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("persist certificate"))
	}
	return certID, nil
}

func (s *Service) markChallengeFailed(ctx restate.ObjectContext, domainID string) error {
	_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		err := s.db.UpdateAcmeChallengeStatus(stepCtx, db.UpdateAcmeChallengeStatusParams{
			DomainID:  domainID,
			Status:    db.AcmeChallengesStatusFailed,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if err != nil {
			return restate.Void{}, err
		}
		metrics.CertificateChallengesTotal.WithLabelValues("failed").Inc()
		return restate.Void{}, nil
	}, restate.WithName("mark failed"))
	return err
}
