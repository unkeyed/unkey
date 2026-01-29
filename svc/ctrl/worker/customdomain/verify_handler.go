package customdomain

import (
	"database/sql"
	"errors"
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

// maxVerificationDuration is how long we keep retrying before marking as failed.
const maxVerificationDuration = 24 * time.Hour

// errNotVerified is returned when CNAME is not yet configured, triggering a Restate retry.
var errNotVerified = errors.New("CNAME not verified yet")

// VerifyDomain performs CNAME verification for a custom domain.
//
// This is a Restate virtual object handler keyed by domain name, ensuring only one
// verification workflow runs per domain at any time. The handler checks DNS once
// per invocation - Restate's retry policy handles periodic re-checks (every 1 minute
// for up to 24 hours).
//
// Once CNAME verification succeeds, the workflow:
// 1. Updates domain status to "verified"
// 2. Creates an ACME challenge record to trigger certificate issuance
// 3. Creates a frontline route to enable traffic routing
//
// If verification fails after ~24 hours of retries, Restate kills the invocation.
func (s *Service) VerifyDomain(
	ctx restate.ObjectContext,
	_ *hydrav1.VerifyDomainRequest,
) (*hydrav1.VerifyDomainResponse, error) {
	domain := restate.Key(ctx)
	dom, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.CustomDomain, error) {
		return db.Query.FindCustomDomainByDomain(stepCtx, s.db.RO(), domain)
	}, restate.WithName("fetch domain"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to fetch domain record"))
	}

	// Check if we've exceeded the verification window
	elapsed := time.Since(time.UnixMilli(dom.CreatedAt))
	if elapsed > maxVerificationDuration {
		return s.onVerificationFailed(ctx, dom, "CNAME verification timed out after 24 hours")
	}

	// Mark domain as actively being verified (idempotent)
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

	// Check CNAME
	verified, checkErr := restate.Run(ctx, func(stepCtx restate.RunContext) (bool, error) {
		return s.checkCNAME(dom.Domain, dom.TargetCname)
	}, restate.WithName("dns-check"))
	if checkErr != nil {
		s.logger.Warn("DNS check error",
			"domain", dom.Domain,
			"error", checkErr,
			"elapsed", elapsed,
		)
	}

	// Track attempt count for observability
	_, updateErr := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.UpdateCustomDomainCheckAttempt(stepCtx, s.db.RW(), db.UpdateCustomDomainCheckAttemptParams{
			ID:            dom.ID,
			CheckAttempts: dom.CheckAttempts + 1,
			LastCheckedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("update-attempt"))
	if updateErr != nil {
		s.logger.Warn("failed to update check attempt count",
			"domain", dom.Domain,
			"error", updateErr,
		)
	}

	if verified {
		return s.onVerificationSuccess(ctx, dom)
	}

	// Not verified yet - return error to trigger Restate retry
	s.logger.Info("domain not verified, will retry",
		"domain", dom.Domain,
		"attempts", dom.CheckAttempts+1,
		"elapsed", elapsed,
	)
	return nil, errNotVerified
}

// RetryVerification resets a failed domain and restarts the verification process.
func (s *Service) RetryVerification(
	ctx restate.ObjectContext,
	_ *hydrav1.RetryVerificationRequest,
) (*hydrav1.RetryVerificationResponse, error) {
	domain := restate.Key(ctx)
	s.logger.Info("retrying domain verification", "domain", domain)

	_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.ResetCustomDomainVerification(stepCtx, s.db.RW(), db.ResetCustomDomainVerificationParams{
			Domain:             domain,
			VerificationStatus: db.CustomDomainsVerificationStatusPending,
			CheckAttempts:      0,
			InvocationID:       sql.NullString{Valid: false},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("reset verification"))
	if err != nil {
		return nil, err
	}

	_, err = s.VerifyDomain(ctx, &hydrav1.VerifyDomainRequest{})
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

	// Create frontline route for traffic routing. If no deployment exists yet,
	// the route will be assigned when the first deployment happens.
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

// onVerificationFailed handles failed domain verification after timeout.
// It updates the domain status to failed and returns a terminal error to stop retries.
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

	// Return terminal error to stop Restate retries
	return nil, restate.TerminalError(fmt.Errorf("domain verification failed: %s", errorMsg), 0)
}
