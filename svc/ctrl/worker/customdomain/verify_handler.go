package customdomain

import (
	"database/sql"
	"fmt"
	"net"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
)

// verificationBackoff defines the sleep durations between DNS verification attempts.
// Total verification window is approximately 24 hours before giving up.
var verificationBackoff = []time.Duration{
	1 * time.Minute,  // Check 1: 1 minute after start
	5 * time.Minute,  // Check 2: 6 minutes total
	15 * time.Minute, // Check 3: 21 minutes total
	30 * time.Minute, // Check 4: 51 minutes total
	1 * time.Hour,    // Check 5: ~2 hours total
	2 * time.Hour,    // Check 6: ~4 hours total
	4 * time.Hour,    // Check 7: ~8 hours total
	6 * time.Hour,    // Check 8: ~14 hours total
	6 * time.Hour,    // Check 9: ~20 hours total
	4 * time.Hour,    // Check 10: ~24 hours total
}

// VerifyDomain performs CNAME verification for a custom domain.
//
// This is a Restate virtual object handler keyed by domain name, ensuring only one
// verification workflow runs per domain at any time. The workflow checks DNS records
// with exponential backoff over approximately 24 hours.
//
// Once CNAME verification succeeds, the workflow:
// 1. Updates domain status to "verified"
// 2. Creates an ACME challenge record to trigger certificate issuance
// 3. Creates a frontline route to enable traffic routing
//
// If verification fails after all retry attempts, an error is returned so Restate
// marks the workflow as failed.
func (s *Service) VerifyDomain(
	ctx restate.ObjectContext,
	req *hydrav1.VerifyDomainRequest,
) (*hydrav1.VerifyDomainResponse, error) {
	dom, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.CustomDomain, error) {
		return db.Query.FindCustomDomainByDomain(stepCtx, s.db.RO(), req.GetDomain())
	}, restate.WithName("fetch domain"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to fetch domain record"))
	}

	s.logger.Info("starting domain verification",
		"domain", dom.Domain,
		"workspace_id", dom.WorkspaceID,
	)

	// Mark domain as actively being verified so the UI can show progress.
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.UpdateCustomDomainVerificationStatus(stepCtx, s.db.RW(), db.UpdateCustomDomainVerificationStatusParams{
			ID:                 dom.ID,
			VerificationStatus: db.CustomDomainsVerificationStatusVerifying,
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("mark verifying"))
	if err != nil {
		return nil, err
	}

	// Poll DNS until the CNAME is configured or we exhaust retries.
	maxAttempts := len(verificationBackoff)
	for attempt := range maxAttempts {
		verified, checkErr := restate.Run(ctx, func(stepCtx restate.RunContext) (bool, error) {
			return s.checkCNAME(dom.Domain, dom.TargetCname)
		}, restate.WithName(fmt.Sprintf("dns-check-%d", attempt)))
		if checkErr != nil {
			s.logger.Warn("DNS check error",
				"domain", dom.Domain,
				"error", checkErr,
				"attempt", attempt,
			)
		}

		// Track attempt count for observability. Failure here is non-fatal since
		// it's just metadata - the verification can still proceed.
		_, updateErr := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.UpdateCustomDomainCheckAttempt(stepCtx, s.db.RW(), db.UpdateCustomDomainCheckAttemptParams{
				ID:            dom.ID,
				CheckAttempts: int32(attempt + 1),
				LastCheckedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
		}, restate.WithName(fmt.Sprintf("update-attempt-%d", attempt)))
		if updateErr != nil {
			s.logger.Warn("failed to update check attempt count",
				"domain", dom.Domain,
				"error", updateErr,
				"attempt", attempt,
			)
		}

		if verified {
			return s.onVerificationSuccess(ctx, dom)
		}

		// Not verified yet - sleep and retry
		if attempt < maxAttempts-1 {
			sleepDuration := verificationBackoff[attempt]
			s.logger.Info("domain not verified, sleeping before retry",
				"domain", dom.Domain,
				"attempt", attempt,
				"next_check_in", sleepDuration,
			)
			if err := restate.Sleep(ctx, sleepDuration); err != nil {
				return nil, err
			}
		}
	}

	// Max attempts reached - mark as failed and return error
	return s.onVerificationFailed(ctx, dom, "CNAME verification failed after maximum retry attempts")
}

// RetryVerification resets a failed domain and restarts the verification process.
func (s *Service) RetryVerification(
	ctx restate.ObjectContext,
	req *hydrav1.RetryVerificationRequest,
) (*hydrav1.RetryVerificationResponse, error) {
	s.logger.Info("retrying domain verification", "domain", req.GetDomain())

	_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.ResetCustomDomainVerification(stepCtx, s.db.RW(), db.ResetCustomDomainVerificationParams{
			Domain:             req.GetDomain(),
			VerificationStatus: db.CustomDomainsVerificationStatusPending,
			CheckAttempts:      0,
			InvocationID:       sql.NullString{Valid: false},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("reset verification"))
	if err != nil {
		return nil, err
	}

	_, err = s.VerifyDomain(ctx, &hydrav1.VerifyDomainRequest{
		Domain: req.GetDomain(),
	})
	if err != nil {
		return nil, err
	}

	return &hydrav1.RetryVerificationResponse{}, nil
}

// checkCNAME verifies that the domain has a CNAME record pointing to the expected target.
// Returns (true, nil) if verified, (false, nil) if CNAME doesn't match, or (false, err)
// if DNS lookup failed due to network issues or misconfiguration.
func (s *Service) checkCNAME(domain, expectedCname string) (bool, error) {
	cname, err := net.LookupCNAME(domain)
	if err != nil {
		// Return the error so caller can log it and distinguish network failures
		// from "CNAME not configured yet".
		return false, err
	}

	// Normalize by removing trailing dots for comparison.
	cname = strings.TrimSuffix(cname, ".")
	expectedCname = strings.TrimSuffix(expectedCname, ".")

	return strings.EqualFold(cname, expectedCname), nil
}

// onVerificationSuccess handles successful domain verification by updating status,
// creating an ACME challenge for certificate issuance, and setting up traffic routing.
func (s *Service) onVerificationSuccess(
	ctx restate.ObjectContext,
	dom db.CustomDomain,
) (*hydrav1.VerifyDomainResponse, error) {
	now := time.Now().UnixMilli()

	_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.UpdateCustomDomainVerificationStatus(stepCtx, s.db.RW(), db.UpdateCustomDomainVerificationStatusParams{
			ID:                 dom.ID,
			VerificationStatus: db.CustomDomainsVerificationStatusVerified,
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: now},
		})
	}, restate.WithName("mark verified"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to mark domain as verified"))
	}

	// Create a placeholder ACME challenge record. Token and Authorization are empty
	// because they're provided by the ACME server during the challenge flow, not by us.
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.InsertAcmeChallenge(stepCtx, s.db.RW(), db.InsertAcmeChallengeParams{
			DomainID:      dom.ID,
			WorkspaceID:   dom.WorkspaceID,
			Token:         "",
			ChallengeType: db.AcmeChallengesChallengeTypeHTTP01,
			Authorization: "",
			Status:        db.AcmeChallengesStatusWaiting,
			ExpiresAt:     time.Now().Add(30 * 24 * time.Hour).UnixMilli(),
			CreatedAt:     now,
			UpdatedAt:     sql.NullInt64{Valid: true, Int64: now},
		})
	}, restate.WithName("create acme challenge"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to create ACME challenge record"))
	}

	// Kick off certificate issuance asynchronously via Restate.
	certClient := hydrav1.NewCertificateServiceClient(ctx, dom.Domain)
	certClient.ProcessChallenge().Send(&hydrav1.ProcessChallengeRequest{
		WorkspaceId: dom.WorkspaceID,
		Domain:      dom.Domain,
	})

	// Create frontline route so traffic can be routed to this domain once the cert is ready.
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		project, findErr := db.Query.FindProjectById(stepCtx, s.db.RO(), dom.ProjectID)
		if findErr != nil {
			return restate.Void{}, fault.Wrap(findErr, fault.Internal("failed to find project for frontline route"))
		}

		deploymentID := ""
		if project.LiveDeploymentID.Valid {
			deploymentID = project.LiveDeploymentID.String
		}

		return restate.Void{}, db.Query.InsertFrontlineRoute(stepCtx, s.db.RW(), db.InsertFrontlineRouteParams{
			ID:                       uid.New(uid.FrontlineRoutePrefix),
			ProjectID:                dom.ProjectID,
			DeploymentID:             deploymentID,
			EnvironmentID:            dom.EnvironmentID,
			FullyQualifiedDomainName: dom.Domain,
			Sticky:                   db.FrontlineRoutesStickyLive,
			CreatedAt:                now,
			UpdatedAt:                sql.NullInt64{Valid: true, Int64: now},
		})
	}, restate.WithName("create frontline route"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to create frontline route"))
	}

	s.logger.Info("domain verification completed successfully",
		"domain", dom.Domain,
	)

	return &hydrav1.VerifyDomainResponse{}, nil
}

// onVerificationFailed handles failed domain verification after all retries exhausted.
// It updates the domain status to failed and returns an error so Restate marks the workflow as failed.
func (s *Service) onVerificationFailed(
	ctx restate.ObjectContext,
	dom db.CustomDomain,
	errorMsg string,
) (*hydrav1.VerifyDomainResponse, error) {
	_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.UpdateCustomDomainFailed(stepCtx, s.db.RW(), db.UpdateCustomDomainFailedParams{
			ID:                 dom.ID,
			VerificationStatus: db.CustomDomainsVerificationStatusFailed,
			VerificationError:  sql.NullString{Valid: true, String: errorMsg},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("mark failed"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to mark domain as failed"))
	}

	s.logger.Info("domain verification failed",
		"domain", dom.Domain,
		"error", errorMsg,
	)

	return nil, fault.New("domain verification failed",
		fault.Internal(errorMsg),
	)
}
