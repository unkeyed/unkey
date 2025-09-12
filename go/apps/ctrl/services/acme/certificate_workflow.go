package acme

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	pdb "github.com/unkeyed/unkey/go/pkg/partition/db"
	"github.com/unkeyed/unkey/go/pkg/vault"
)

// CertificateChallenge tries to get a certificate from Let's Encrypt
type CertificateChallenge struct {
	db          db.Database
	partitionDB db.Database
	logger      logging.Logger
	acmeClient  *lego.Client
	vault       *vault.Service
}

type CertificateChallengeConfig struct {
	DB          db.Database
	PartitionDB db.Database
	Logger      logging.Logger
	AcmeClient  *lego.Client
	Vault       *vault.Service
}

// NewCertificateChallenge creates a new certificate challenges workflow instance
// and ensures that we have a valid ACME User
func NewCertificateChallenge(config CertificateChallengeConfig) *CertificateChallenge {
	return &CertificateChallenge{
		db:          config.DB,
		partitionDB: config.PartitionDB,
		logger:      config.Logger,
		acmeClient:  config.AcmeClient,
		vault:       config.Vault,
	}
}

// Name returns the workflow name for registration
func (w *CertificateChallenge) Name() string {
	return "certificate_challenge"
}

// CertificateChallengeRequest defines the input for the certificate challenge workflow
type CertificateChallengeRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Domain      string `json:"domain"`
}

type EncryptedCertificate struct {
	Certificate         string `json:"certificate"`
	EncryptedPrivateKey string `json:"encrypted_private_key"`
	ExpiresAt           int64  `json:"expires_at"`
}

// Run executes the complete build and deployment workflow
func (w *CertificateChallenge) Run(ctx hydra.WorkflowContext, req *CertificateChallengeRequest) error {
	w.logger.Info("starting certificate challenge", "workspace_id", req.WorkspaceID, "domain", req.Domain)

	dom, err := hydra.Step(ctx, "resolve-domain", func(stepCtx context.Context) (db.Domain, error) {
		return db.Query.FindDomainByDomain(stepCtx, w.db.RO(), req.Domain)
	})
	if err != nil {
		return err
	}

	err = hydra.StepVoid(ctx, "acquire-challenge", func(stepCtx context.Context) error {
		return db.Query.UpdateAcmeChallengeTryClaiming(stepCtx, w.db.RW(), db.UpdateAcmeChallengeTryClaimingParams{
			DomainID:  dom.ID,
			Status:    db.AcmeChallengesStatusPending,
			UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		})
	})
	if err != nil {
		return err
	}

	cert, err := hydra.Step(ctx, "obtain-certificate", func(stepCtx context.Context) (EncryptedCertificate, error) {
		// A certificate request can be either
		// A: We have a new domain WITHOUT a certificate
		// B: We have to renew a existing certificate
		// Regardless we first claim the challenge so that no-other job tries to do the same, this will just annoy acme ratelimits
		if err != nil {
			db.Query.UpdateAcmeChallengeStatus(ctx.Context(), w.db.RW(), db.UpdateAcmeChallengeStatusParams{
				DomainID:  dom.ID,
				Status:    db.AcmeChallengesStatusFailed,
				UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			return EncryptedCertificate{}, err
		}

		currCert, err := pdb.Query.FindCertificateByHostname(stepCtx, w.partitionDB.RO(), req.Domain)
		if err != nil && !db.IsNotFound(err) {
			return EncryptedCertificate{}, err
		}

		shouldRenew := !db.IsNotFound(err)
		var certificates *certificate.Resource
		if shouldRenew {
			resp, err := w.vault.Decrypt(stepCtx, &vaultv1.DecryptRequest{
				Keyring:   "unkey",
				Encrypted: string(currCert.EncryptedPrivateKey),
			})
			if err != nil {
				return EncryptedCertificate{}, err
			}

			certificates, err = w.acmeClient.Certificate.Renew(certificate.Resource{
				Domain:      req.Domain,
				PrivateKey:  []byte(resp.Plaintext),
				Certificate: []byte(currCert.Certificate),
			}, true, false, "")
		} else {
			certificates, err = w.acmeClient.Certificate.Obtain(certificate.ObtainRequest{
				Domains: []string{req.Domain},
				Bundle:  true,
			})
		}
		if err != nil {
			return EncryptedCertificate{}, err
		}

		resp, err := w.vault.Encrypt(stepCtx, &vaultv1.EncryptRequest{
			Keyring: "unkey",
			Data:    string(certificates.PrivateKey),
		})
		if err != nil {
			return EncryptedCertificate{}, err
		}

		expiresAt, err := getCertificateExpiry(string(certificates.Certificate))
		if err != nil {
			return EncryptedCertificate{}, err
		}

		return EncryptedCertificate{
			ExpiresAt:           expiresAt.UnixMilli(),
			Certificate:         string(certificates.Certificate),
			EncryptedPrivateKey: resp.Encrypted,
		}, nil
	})
	if err != nil {
		return err
	}

	err = hydra.StepVoid(ctx, "persist-certificate", func(stepCtx context.Context) error {
		now := time.Now().UnixMilli()
		return pdb.Query.InsertCertificate(stepCtx, w.partitionDB.RW(), pdb.InsertCertificateParams{
			WorkspaceID:         dom.WorkspaceID,
			Hostname:            req.Domain,
			Certificate:         cert.Certificate,
			EncryptedPrivateKey: cert.EncryptedPrivateKey,
			CreatedAt:           now,
			UpdatedAt:           sql.NullInt64{Valid: true, Int64: now},
		})
	})
	if err != nil {
		return err
	}

	err = hydra.StepVoid(ctx, "complete-challenge", func(stepCtx context.Context) error {
		return db.Query.UpdateAcmeChallengeVerifiedWithExpiry(stepCtx, w.db.RW(), db.UpdateAcmeChallengeVerifiedWithExpiryParams{
			Status:    db.AcmeChallengesStatusVerified,
			ExpiresAt: cert.ExpiresAt,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			DomainID:  dom.ID,
		})
	})
	if err != nil {
		return err
	}

	return nil
}
