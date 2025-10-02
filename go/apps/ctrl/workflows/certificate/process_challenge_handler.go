package certificate

import (
	"database/sql"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	pdb "github.com/unkeyed/unkey/go/pkg/partition/db"
)

type EncryptedCertificate struct {
	Certificate         string
	EncryptedPrivateKey string
	ExpiresAt           int64
}

// ProcessChallenge handles the complete ACME certificate challenge flow
func (s *Service) ProcessChallenge(
	ctx restate.ObjectContext,
	req *hydrav1.ProcessChallengeRequest,
) (*hydrav1.ProcessChallengeResponse, error) {
	s.logger.Info("starting certificate challenge",
		"workspace_id", req.GetWorkspaceId(),
		"domain", req.GetDomain(),
	)

	// Step 1: Resolve domain
	dom, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.Domain, error) {
		return db.Query.FindDomainByDomain(stepCtx, s.db.RO(), req.GetDomain())
	}, restate.WithName("resolving domain"))
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
	}, restate.WithName("acquiring challenge"))
	if err != nil {
		return nil, err
	}

	// Step 3: Get or create ACME client for this workspace
	acmeClient, err := restate.Run(ctx, func(stepCtx restate.RunContext) (*certificate.Resource, error) {
		// TODO: Get ACME client for workspace
		// This requires implementing GetOrCreateUser from acme/user.go
		// and setting up challenge providers (HTTP-01, DNS-01)

		// For now, return error indicating this needs ACME client setup
		return nil, restate.TerminalError(
			err,
			500,
		)
	}, restate.WithName("setup acme client"))
	if err != nil {
		_, _ = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
			db.Query.UpdateAcmeChallengeStatus(stepCtx, s.db.RW(), db.UpdateAcmeChallengeStatusParams{
				DomainID:  dom.ID,
				Status:    db.AcmeChallengesStatusFailed,
				UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			return restate.Void{}, nil
		}, restate.WithName("mark challenge failed"))
		return &hydrav1.ProcessChallengeResponse{
			Status: "failed",
		}, nil
	}

	// Step 4: Obtain or renew certificate
	cert, err := restate.Run(ctx, func(stepCtx restate.RunContext) (EncryptedCertificate, error) {
		currCert, err := pdb.Query.FindCertificateByHostname(stepCtx, s.partitionDB.RO(), req.GetDomain())
		if err != nil && !db.IsNotFound(err) {
			return EncryptedCertificate{}, err
		}

		// TODO: Implement certificate obtain/renew logic
		// This requires the ACME client from step 3
		_ = currCert
		_ = acmeClient

		return EncryptedCertificate{}, restate.TerminalError(
			err,
			500,
		)
	}, restate.WithName("obtaining certificate"))
	if err != nil {
		return &hydrav1.ProcessChallengeResponse{
			Status: "failed",
		}, nil
	}

	// Step 5: Persist certificate to partition DB
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		now := time.Now().UnixMilli()
		return restate.Void{}, pdb.Query.InsertCertificate(stepCtx, s.partitionDB.RW(), pdb.InsertCertificateParams{
			WorkspaceID:         dom.WorkspaceID,
			Hostname:            req.GetDomain(),
			Certificate:         cert.Certificate,
			EncryptedPrivateKey: cert.EncryptedPrivateKey,
			CreatedAt:           now,
			UpdatedAt:           sql.NullInt64{Valid: true, Int64: now},
		})
	}, restate.WithName("persisting certificate"))
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
	}, restate.WithName("completing challenge"))
	if err != nil {
		return nil, err
	}

	s.logger.Info("certificate challenge completed successfully",
		"domain", req.GetDomain(),
		"expires_at", cert.ExpiresAt,
	)

	return &hydrav1.ProcessChallengeResponse{
		Status: "success",
	}, nil
}
