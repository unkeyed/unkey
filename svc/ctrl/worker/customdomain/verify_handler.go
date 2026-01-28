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
// If verification fails after all retry attempts, the domain status is set to "failed".
func (s *Service) VerifyDomain(
	ctx restate.ObjectContext,
	req *hydrav1.VerifyDomainRequest,
) (*hydrav1.VerifyDomainResponse, error) {
	s.logger.Info("starting domain verification",
		"domain", req.GetDomain(),
		"workspace_id", req.GetWorkspaceId(),
	)

	// Step 1: Fetch domain record
	dom, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.CustomDomain, error) {
		return db.Query.FindCustomDomainByDomain(stepCtx, s.db.RO(), req.GetDomain())
	}, restate.WithName("fetch domain"))
	if err != nil {
		return nil, err
	}

	// Step 2: Update status to verifying
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

	// Step 3: Verification loop with exponential backoff
	maxAttempts := len(verificationBackoff)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Check DNS CNAME record
		verified, checkErr := restate.Run(ctx, func(stepCtx restate.RunContext) (bool, error) {
			return s.checkCNAME(req.GetDomain(), dom.TargetCname)
		}, restate.WithName(fmt.Sprintf("dns-check-%d", attempt)))
		if checkErr != nil {
			s.logger.Warn("DNS check error",
				"domain", req.GetDomain(),
				"error", checkErr,
				"attempt", attempt,
			)
			// Continue to next attempt on error
		}

		// Update check count
		_, _ = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.UpdateCustomDomainCheckAttempt(stepCtx, s.db.RW(), db.UpdateCustomDomainCheckAttemptParams{
				ID:            dom.ID,
				CheckAttempts: int32(attempt + 1),
				LastCheckedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
		}, restate.WithName(fmt.Sprintf("update-attempt-%d", attempt)))

		if verified {
			// Success! Trigger certificate issuance and routing setup
			return s.onVerificationSuccess(ctx, dom)
		}

		// Not verified yet - sleep and retry
		if attempt < maxAttempts-1 {
			sleepDuration := verificationBackoff[attempt]
			s.logger.Info("domain not verified, sleeping before retry",
				"domain", req.GetDomain(),
				"attempt", attempt,
				"next_check_in", sleepDuration,
			)
			if err := restate.Sleep(ctx, sleepDuration); err != nil {
				return nil, err
			}
		}
	}

	// Max attempts reached - mark as failed
	return s.onVerificationFailed(ctx, dom, "CNAME verification failed after maximum retry attempts")
}

// RetryVerification resets a failed domain and restarts the verification process.
func (s *Service) RetryVerification(
	ctx restate.ObjectContext,
	req *hydrav1.RetryVerificationRequest,
) (*hydrav1.RetryVerificationResponse, error) {
	s.logger.Info("retrying domain verification",
		"domain", req.GetDomain(),
		"workspace_id", req.GetWorkspaceId(),
	)

	// Reset verification state
	_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.ResetCustomDomainVerification(stepCtx, s.db.RW(), db.ResetCustomDomainVerificationParams{
			Domain:             req.GetDomain(),
			VerificationStatus: db.CustomDomainsVerificationStatusPending,
			CheckAttempts:      0,
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("reset verification"))
	if err != nil {
		return nil, err
	}

	// Trigger a new verification workflow
	resp, err := s.VerifyDomain(ctx, &hydrav1.VerifyDomainRequest{
		WorkspaceId: req.GetWorkspaceId(),
		Domain:      req.GetDomain(),
	})
	if err != nil {
		return nil, err
	}

	return &hydrav1.RetryVerificationResponse{
		Status: resp.GetStatus(),
	}, nil
}

// checkCNAME verifies that the domain has a CNAME record pointing to the expected target.
func (s *Service) checkCNAME(domain, expectedCname string) (bool, error) {
	cname, err := net.LookupCNAME(domain)
	if err != nil {
		// DNS lookup failed - could be NXDOMAIN, no CNAME, or network error
		return false, nil
	}

	// Normalize: remove trailing dot if present
	cname = strings.TrimSuffix(cname, ".")
	expectedCname = strings.TrimSuffix(expectedCname, ".")

	return strings.EqualFold(cname, expectedCname), nil
}

// onVerificationSuccess handles successful domain verification.
// It updates the domain status, creates an ACME challenge for certificate issuance,
// and creates a frontline route for traffic routing.
func (s *Service) onVerificationSuccess(
	ctx restate.ObjectContext,
	dom db.CustomDomain,
) (*hydrav1.VerifyDomainResponse, error) {
	now := time.Now().UnixMilli()

	// Step 1: Update status to verified
	_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.UpdateCustomDomainVerified(stepCtx, s.db.RW(), db.UpdateCustomDomainVerifiedParams{
			ID:                 dom.ID,
			VerificationStatus: db.CustomDomainsVerificationStatusVerified,
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: now},
		})
	}, restate.WithName("mark verified"))
	if err != nil {
		return nil, err
	}

	// Step 2: Create ACME challenge record to trigger certificate issuance
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.InsertAcmeChallenge(stepCtx, s.db.RW(), db.InsertAcmeChallengeParams{
			DomainID:      dom.ID,
			WorkspaceID:   dom.WorkspaceID,
			Token:         "", // Will be populated by ACME client
			ChallengeType: db.AcmeChallengesChallengeTypeHTTP01,
			Authorization: "",
			Status:        db.AcmeChallengesStatusWaiting,
			ExpiresAt:     time.Now().Add(30 * 24 * time.Hour).UnixMilli(),
			CreatedAt:     now,
		})
	}, restate.WithName("create acme challenge"))
	if err != nil {
		return nil, err
	}

	// Step 3: Trigger certificate issuance workflow
	certClient := hydrav1.NewCertificateServiceClient(ctx, dom.Domain)
	certClient.ProcessChallenge().Send(&hydrav1.ProcessChallengeRequest{
		WorkspaceId: dom.WorkspaceID,
		Domain:      dom.Domain,
	})

	// Step 4: Create frontline route for the custom domain
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		// Get the project to check for live deployment
		project, findErr := db.Query.FindProjectById(stepCtx, s.db.RO(), dom.ProjectID)
		if findErr != nil {
			return restate.Void{}, fmt.Errorf("failed to find project: %w", findErr)
		}

		// Use live deployment if available
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
		})
	}, restate.WithName("create frontline route"))
	if err != nil {
		return nil, err
	}

	s.logger.Info("domain verification completed successfully",
		"domain", dom.Domain,
	)

	return &hydrav1.VerifyDomainResponse{
		Status: "verified",
	}, nil
}

// onVerificationFailed handles failed domain verification after all retries exhausted.
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
		return nil, err
	}

	s.logger.Info("domain verification failed",
		"domain", dom.Domain,
		"error", errorMsg,
	)

	return &hydrav1.VerifyDomainResponse{
		Status: "failed",
		Error:  errorMsg,
	}, nil
}
