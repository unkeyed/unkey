package customdomain

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/domainverify"
)

// maxVerificationDuration limits how long we retry DNS verification before
// marking a domain as failed.
const maxVerificationDuration = 24 * time.Hour

// rowVisibilityGrace is how long a missing domain row stays retryable before it
// means deletion. AddCustomDomain submits this workflow inside the transaction
// that inserts the row, so the first attempts can run before the commit lands;
// treating that as terminal would kill the workflow and strand the row in
// `pending`. The race resolves in milliseconds, the window is padding.
const rowVisibilityGrace = 2 * time.Minute

// errNotVerified signals incomplete verification and triggers Restate retries.
var errNotVerified = errors.New("domain not verified yet")

// VerifyDomain verifies a custom domain for routing. The DNS rules, including
// contention with other workspaces in either domain table, live in
// [domainverify.Check].
//
// This is a Restate virtual object handler keyed by domain ID, ensuring only one
// verification workflow runs per domain at any time. The handler checks DNS once
// per invocation - Restate's retry policy handles periodic re-checks (every 1 minute
// for up to 24 hours).
//
// Once verification succeeds, the workflow:
// 1. Updates domain status to "verified"
// 2. Creates an ACME challenge record for certificate issuance
// 3. Revokes any existing verified claim from another workspace (contention case)
// 4. Creates a frontline route to enable traffic routing
// 5. Starts certificate issuance
//
// If verification fails after ~24 hours of retries, Restate kills the invocation.
func (s *Service) VerifyDomain(
	ctx restate.ObjectContext,
	_ *hydrav1.VerifyDomainRequest,
) (*hydrav1.VerifyDomainResponse, error) {
	domainID := restate.Key(ctx)

	// Journaled so every retry measures the row-visibility grace from the first
	// attempt rather than from itself, which would never let the window expire.
	startedAt, err := restateutil.Now(ctx)
	if err != nil {
		return nil, err
	}

	// Fetch domain - NOT journaled so we get fresh state on each retry
	dom, err := s.db.FindCustomDomainById(ctx, domainID)
	if err != nil {
		if db.IsNotFound(err) {
			if time.Since(startedAt) < rowVisibilityGrace {
				return nil, fault.Wrap(err, fault.Internal("domain row not visible yet"))
			}

			// The domain row was deleted (DeleteCustomDomain, environment cascade,
			// etc.) while verification was still running. Stop retrying instead of
			// surfacing a retryable internal error for up to 24 hours.
			logger.Info("domain record deleted, stopping verification workflow",
				"domain_id", domainID,
			)
			return nil, restate.ToTerminalError(fmt.Errorf("domain record not found: %s", domainID), restate.WithErrorCode(404))
		}
		return nil, fault.Wrap(err, fault.Internal("failed to fetch domain record"))
	}

	elapsed := time.Since(startedAt)
	if elapsed > maxVerificationDuration {
		return s.onVerificationFailed(ctx, dom, "domain verification timed out after 24 hours")
	}

	// Mark domain as actively being verified - NOT journaled
	err = s.db.UpdateCustomDomainVerificationStatus(ctx, db.UpdateCustomDomainVerificationStatusParams{
		ID:                 dom.ID,
		VerificationStatus: db.CustomDomainsVerificationStatusVerifying,
		UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	})
	if err != nil {
		return nil, err
	}

	outcome, err := domainverify.Check(ctx, s.resolver, s.db, domainverify.Domain{
		Hostname:          dom.Domain,
		WorkspaceID:       dom.WorkspaceID,
		TargetCname:       dom.TargetCname,
		VerificationToken: dom.VerificationToken,
	})
	if err != nil {
		return nil, err
	}

	// Update attempt count and verification flags - NOT journaled so we get fresh updates
	err = s.db.UpdateCustomDomainCheckAttempt(ctx, db.UpdateCustomDomainCheckAttemptParams{
		ID:            dom.ID,
		CheckAttempts: dom.CheckAttempts + 1,
		LastCheckedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	})
	if err != nil {
		return nil, err
	}

	err = s.db.UpdateCustomDomainOwnership(ctx, db.UpdateCustomDomainOwnershipParams{
		OwnershipVerified: outcome.TxtVerified,
		CnameVerified:     outcome.CnameVerified,
		UpdatedAt:         sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		ID:                dom.ID,
	})
	if err != nil {
		return nil, err
	}

	logger.Info("DNS verification check complete",
		"domain", dom.Domain,
		"is_apex", outcome.IsApex,
		"contested", outcome.Contested,
		"requires_txt", outcome.RequiresTxt,
		"txt_verified", outcome.TxtVerified,
		"cname_verified", outcome.CnameVerified,
		"apex_has_records", outcome.ApexHasRecords,
		"attempts", dom.CheckAttempts+1,
		"elapsed", elapsed,
	)

	if outcome.Verified {
		return s.onVerificationSuccess(ctx, dom)
	}

	// Not fully verified yet - return error to trigger Restate retry
	return nil, errNotVerified
}

// RetryVerification resets a failed domain and restarts verification after the
// user fixes DNS configuration.
func (s *Service) RetryVerification(
	ctx restate.ObjectContext,
	_ *hydrav1.RetryVerificationRequest,
) (*hydrav1.RetryVerificationResponse, error) {
	domainID := restate.Key(ctx)
	logger.Info("retrying domain verification", "domain_id", domainID)

	_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, s.db.ResetCustomDomainVerification(stepCtx, db.ResetCustomDomainVerificationParams{
			ID:                 domainID,
			VerificationStatus: db.CustomDomainsVerificationStatusPending,
			CheckAttempts:      0,
			InvocationID:       sql.NullString{Valid: false},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("reset verification"))
	if err != nil {
		return nil, err
	}

	_, verifyErr := s.VerifyDomain(ctx, &hydrav1.VerifyDomainRequest{})
	if verifyErr != nil {
		return nil, verifyErr
	}

	return &hydrav1.RetryVerificationResponse{}, nil
}

// onVerificationSuccess handles successful domain verification by updating status,
// creating an ACME challenge for certificate issuance, and setting up traffic routing.
func (s *Service) onVerificationSuccess(
	ctx restate.ObjectContext,
	dom db.CustomDomain,
) (*hydrav1.VerifyDomainResponse, error) {
	now := time.Now().UnixMilli()

	_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, s.db.UpdateCustomDomainVerificationStatus(stepCtx, db.UpdateCustomDomainVerificationStatusParams{
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
		return restate.Void{}, s.db.InsertAcmeChallenge(stepCtx, db.InsertAcmeChallengeParams{
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

	err = restate.RunVoid(ctx, func(stepCtx restate.RunContext) error {
		return domainverify.RevokeContestedClaim(stepCtx, s.db, dom.Domain, dom.WorkspaceID, now)
	}, restate.WithName("revoke contested domain"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to revoke contested domain"))
	}

	err = restate.RunVoid(ctx, func(stepCtx restate.RunContext) error {
		return s.createFrontlineRoute(stepCtx, dom, now)
	}, restate.WithName("create frontline route"))
	if err != nil {
		if restate.IsTerminalError(err) {
			return nil, err
		}
		return nil, fault.Wrap(err, fault.Internal("failed to create frontline route"))
	}

	// Sent last so issuance never sees the hostname verified in two workspaces,
	// nor starts for a domain whose route turned out to conflict.
	certClient := hydrav1.NewCertificateServiceClient(ctx, dom.Domain)
	certClient.ProcessChallenge().Send(&hydrav1.ProcessChallengeRequest{
		WorkspaceId: dom.WorkspaceID,
		Domain:      dom.Domain,
	})

	logger.Info("domain verification completed successfully",
		"domain", dom.Domain,
	)

	return &hydrav1.VerifyDomainResponse{}, nil
}

// createFrontlineRoute routes dom's hostname to its app. If no deployment
// exists yet, the route is assigned when the first deployment happens.
//
// The insert generates a fresh route id each attempt, so a retry of an insert
// that already committed hits this domain's own route, which counts as
// success. A route owned by anything else can never be inserted, so the domain
// is failed and its ACME challenge removed in this same step, and the returned
// terminal error stops retries instead of waiting out the 24-hour window.
func (s *Service) createFrontlineRoute(ctx context.Context, dom db.CustomDomain, nowMs int64) error {
	app, err := s.db.FindAppById(ctx, dom.AppID)
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to find app for frontline route"))
	}

	deploymentID := ""
	if app.CurrentDeploymentID.Valid {
		deploymentID = app.CurrentDeploymentID.String
	}

	err = s.db.InsertFrontlineRoute(ctx, db.InsertFrontlineRouteParams{
		ID:                       uid.New(uid.FrontlineRoutePrefix),
		ProjectID:                dom.ProjectID,
		AppID:                    dom.AppID,
		DeploymentID:             deploymentID,
		EnvironmentID:            dom.EnvironmentID,
		FullyQualifiedDomainName: dom.Domain,
		Sticky:                   db.FrontlineRoutesStickyLive,
		CreatedAt:                nowMs,
		UpdatedAt:                sql.NullInt64{Valid: true, Int64: nowMs},
	})
	if err == nil {
		return nil
	}
	if !db.IsDuplicateKeyError(err) {
		return fault.Wrap(err, fault.Internal("failed to insert frontline route"))
	}

	existing, err := s.db.FindFrontlineRouteByFQDN(ctx, dom.Domain)
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to read conflicting frontline route"))
	}
	if existing.ProjectID == dom.ProjectID &&
		existing.EnvironmentID == dom.EnvironmentID &&
		existing.FullyQualifiedDomainName == dom.Domain {
		return nil
	}

	const reason = "domain is already routed to another target"
	if err := s.db.UpdateCustomDomainFailed(ctx, db.UpdateCustomDomainFailedParams{
		ID:                 dom.ID,
		VerificationStatus: db.CustomDomainsVerificationStatusFailed,
		VerificationError:  sql.NullString{Valid: true, String: reason},
		UpdatedAt:          sql.NullInt64{Valid: true, Int64: nowMs},
	}); err != nil {
		return fault.Wrap(err, fault.Internal("failed to mark domain as failed"))
	}
	if err := s.db.DeleteAcmeChallengeByDomainID(ctx, dom.ID); err != nil && !db.IsNotFound(err) {
		return fault.Wrap(err, fault.Internal("failed to delete ACME challenge"))
	}

	logger.Warn("frontline route conflict, domain marked failed",
		"domain", dom.Domain,
		"domain_id", dom.ID,
		"route_id", existing.ID,
	)
	return restate.ToTerminalError(fmt.Errorf("frontline route conflict for %s: %s", dom.Domain, reason))
}

// onVerificationFailed handles failed domain verification after timeout.
// It updates the domain status to failed and returns a terminal error to stop retries.
func (s *Service) onVerificationFailed(
	ctx restate.ObjectContext,
	dom db.CustomDomain,
	errorMsg string,
) (*hydrav1.VerifyDomainResponse, error) {
	_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, s.db.UpdateCustomDomainFailed(stepCtx, db.UpdateCustomDomainFailedParams{
			ID:                 dom.ID,
			VerificationStatus: db.CustomDomainsVerificationStatusFailed,
			VerificationError:  sql.NullString{Valid: true, String: errorMsg},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("mark failed"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to mark domain as failed"))
	}

	logger.Info("domain verification failed",
		"domain", dom.Domain,
		"error", errorMsg,
	)

	// Return terminal error to stop Restate retries
	return nil, restate.ToTerminalError(fmt.Errorf("domain verification failed: %s", errorMsg))
}
