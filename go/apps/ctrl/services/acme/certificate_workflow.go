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
	ID          uint64 `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Domain      string `json:"domain"`
}

type EncryptedCertificate struct {
	Certificate         string `json:"certificate"`
	EncryptedPrivateKey string `json:"encrypted_private_key"`
}

// Run executes the complete build and deployment workflow
func (w *CertificateChallenge) Run(ctx hydra.WorkflowContext, req *CertificateChallengeRequest) error {
	w.logger.Info("starting lets-encrypt challenge", "workspace_id", req.WorkspaceID, "domain", req.Domain)

	dom, err := hydra.Step(ctx, "find-domain", func(stepCtx context.Context) (db.Domain, error) {
		return db.Query.FindDomainByDomain(ctx.Context(), w.db.RO(), req.Domain)
	})
	if err != nil {
		w.logger.Error("failed to find domain", "error", err)
		return err
	}

	err = hydra.StepVoid(ctx, "claim-challenge", func(stepCtx context.Context) error {
		return db.Query.UpdateDomainChallengeTryClaiming(stepCtx, w.db.RW(), db.UpdateDomainChallengeTryClaimingParams{
			DomainID:  dom.ID,
			Status:    db.DomainChallengesStatusPending,
			UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		})
	})
	if err != nil {
		w.logger.Error("failed to claim challenge", "error", err)
		return err
	}

	cert, err := hydra.Step(ctx, "get-and-encrypt-cert", func(stepCtx context.Context) (EncryptedCertificate, error) {
		// The challenge provider is already configured on the ACME client
		// Just request the certificate
		certificates, err := w.acmeClient.Certificate.Obtain(certificate.ObtainRequest{
			Domains: []string{req.Domain},
			Bundle:  true,
		})

		if err != nil {
			db.Query.UpdateDomainChallengeStatus(ctx.Context(), w.db.RW(), db.UpdateDomainChallengeStatusParams{
				DomainID:  dom.ID,
				Status:    db.DomainChallengesStatusFailed,
				UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			w.logger.Error("failed to obtain certificate", "error", err)
			return EncryptedCertificate{}, err
		}

		// TODO: Implement certificate renewal logic
		// w.acmeClient.Certificate.Renew(certificate.Resource{}, bundle bool, mustStaple bool, preferredChain string)
		resp, err := w.vault.Encrypt(stepCtx, &vaultv1.EncryptRequest{
			Keyring: "unkey",
			Data:    string(certificates.PrivateKey),
		})

		if err != nil {
			return EncryptedCertificate{}, err
		}

		return EncryptedCertificate{Certificate: string(certificates.Certificate), EncryptedPrivateKey: resp.Encrypted}, nil
	})
	if err != nil {
		w.logger.Error("failed to store cert in vaults", "error", err)
		return err
	}

	err = hydra.StepVoid(ctx, "store-cert", func(stepCtx context.Context) error {
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
		w.logger.Error("failed to store cert in vaults", "error", err)
		return err
	}

	return nil
}
