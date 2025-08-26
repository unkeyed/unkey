package acme

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// CertificateChallenge tries to get a certificate from Let's Encrypt
type CertificateChallenge struct {
	db          db.Database
	partitionDB db.Database
	logger      logging.Logger
	acmeClient  *lego.Client

	// DNS challenge config (optional)
	route53HostedZoneID string
	awsAccessKeyID      string
	awsSecretAccessKey  string
	awsRegion           string
}

type CertificateChallengeConfig struct {
	DB          db.Database
	PartitionDB db.Database
	Logger      logging.Logger
	AcmeClient  *lego.Client

	// DNS challenge provider configuration (optional)
	Route53HostedZoneID string // For DNS-01
	AWSAccessKeyID      string // Optional, for DNS-01
	AWSSecretAccessKey  string // Optional, for DNS-01
	AWSRegion           string // Optional, defaults to us-east-1
}

// NewCertificateChallenge creates a new certificate challenges workflow instance
// and ensures that we have a valid ACME User
func NewCertificateChallenge(config CertificateChallengeConfig) *CertificateChallenge {
	return &CertificateChallenge{
		db:          config.DB,
		partitionDB: config.PartitionDB,
		logger:      config.Logger,
		acmeClient:  config.AcmeClient,
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

// Run executes the complete build and deployment workflow
func (w *CertificateChallenge) Run(ctx hydra.WorkflowContext, req *CertificateChallengeRequest) error {
	w.logger.Info("starting lets-encrypt challenge", "workspace_id", req.WorkspaceID, "domain", req.Domain)

	// The challenge provider is already configured on the ACME client
	// Just request the certificate
	request := certificate.ObtainRequest{
		Domains: []string{req.Domain},
		Bundle:  true,
	}

	certificates, err := w.acmeClient.Certificate.Obtain(request)
	if err != nil {
		db.Query.UpdateDomainChallengeStatus(ctx.Context(), w.db.RW(), db.UpdateDomainChallengeStatusParams{
			DomainID:  req.Domain,
			Status:    db.DomainChallengesStatusFailed,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		w.logger.Error("failed to obtain certificate", "error", err)
		return err
	}

	// TODO: Implement certificate renewal logic
	// w.acmeClient.Certificate.Renew(certificate.Resource{}, bundle bool, mustStaple bool, preferredChain string)

	// Each certificate comes back with the cert bytes, the bytes of the client's
	// private key, and a certificate URL. SAVE THESE TO DISK.
	fmt.Printf("%#v\n", certificates)

	os.WriteFile("certificate.pem", certificates.Certificate, 0644)
	os.WriteFile("private_key.pem", certificates.PrivateKey, 0644)

	// // Step 1: Generate build ID
	// buildID, err := hydra.Step(ctx, "generate-build-id", func(stepCtx context.Context) (string, error) {
	// 	id := uid.New(uid.BuildPrefix)
	// 	w.logger.Info("generated build ID", "build_id", id)
	// 	return id, nil
	// })
	// if err != nil {
	// 	w.logger.Error("failed to generate build ID", "error", err)
	// 	return err
	// }

	// // Step 1: Generate build ID
	// buildID, err := hydra.Step(ctx, "generate-build-id", func(stepCtx context.Context) (string, error) {
	// 	id := uid.New(uid.BuildPrefix)
	// 	w.logger.Info("generated build ID", "build_id", id)
	// 	return id, nil
	// })
	// if err != nil {
	// 	w.logger.Error("failed to generate build ID", "error", err)
	// 	return err
	// }

	return nil
}
