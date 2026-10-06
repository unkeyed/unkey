package portaldomain

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
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/domainverify"
)

// maxVerificationDuration limits how long DNS verification retries before the
// row is marked failed.
const maxVerificationDuration = 24 * time.Hour

// rowVisibilityGrace is how long a missing row stays retryable before it means
// deletion. AddPortalDomain submits this workflow inside the transaction that
// inserts the row, so the first attempts can run before the commit lands.
const rowVisibilityGrace = 2 * time.Minute

// errNotVerified signals incomplete verification and triggers Restate retries.
var errNotVerified = errors.New("domain not verified yet")

// VerifyDomain checks a portal domain's DNS once per invocation; Restate's
// retry policy re-runs it every minute for up to 24 hours. The DNS and
// contention rules are shared with deploy domains through [domainverify.Check].
func (s *Service) VerifyDomain(
	ctx restate.ObjectContext,
	_ *hydrav1.VerifyPortalDomainRequest,
) (*hydrav1.VerifyPortalDomainResponse, error) {
	domainID := restate.Key(ctx)

	// Journaled so every retry measures from the first attempt.
	startedAt, err := restateutil.Now(ctx)
	if err != nil {
		return nil, err
	}

	// Not journaled, so each retry reads fresh state.
	dom, err := s.db.FindPortalDomainById(ctx, domainID)
	if err != nil {
		if db.IsNotFound(err) {
			if time.Since(startedAt) < rowVisibilityGrace {
				return nil, fault.Wrap(err, fault.Internal("portal domain row not visible yet"))
			}
			logger.Info("portal domain record deleted, stopping verification workflow",
				"domain_id", domainID,
			)
			return nil, restate.ToTerminalError(fmt.Errorf("portal domain record not found: %s", domainID), restate.WithErrorCode(404))
		}
		return nil, fault.Wrap(err, fault.Internal("failed to fetch portal domain record"))
	}

	elapsed := time.Since(startedAt)
	if elapsed > maxVerificationDuration {
		return s.onVerificationFailed(ctx, dom, "domain verification timed out after 24 hours")
	}

	err = s.db.UpdatePortalDomainVerificationStatus(ctx, db.UpdatePortalDomainVerificationStatusParams{
		ID:                 dom.ID,
		VerificationStatus: db.PortalDomainsVerificationStatusVerifying,
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

	now := time.Now().UnixMilli()
	err = s.db.UpdatePortalDomainCheckAttempt(ctx, db.UpdatePortalDomainCheckAttemptParams{
		ID:            dom.ID,
		CheckAttempts: dom.CheckAttempts + 1,
		LastCheckedAt: sql.NullInt64{Valid: true, Int64: now},
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: now},
	})
	if err != nil {
		return nil, err
	}

	err = s.db.UpdatePortalDomainOwnership(ctx, db.UpdatePortalDomainOwnershipParams{
		OwnershipVerified: outcome.TxtVerified,
		CnameVerified:     outcome.CnameVerified,
		UpdatedAt:         sql.NullInt64{Valid: true, Int64: now},
		ID:                dom.ID,
	})
	if err != nil {
		return nil, err
	}

	logger.Info("portal domain DNS verification check complete",
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

	return nil, errNotVerified
}

// RetryVerification resets a portal domain and restarts verification.
func (s *Service) RetryVerification(
	ctx restate.ObjectContext,
	_ *hydrav1.RetryPortalDomainVerificationRequest,
) (*hydrav1.RetryPortalDomainVerificationResponse, error) {
	domainID := restate.Key(ctx)
	logger.Info("retrying portal domain verification", "domain_id", domainID)

	err := restate.RunVoid(ctx, func(stepCtx restate.RunContext) error {
		return s.db.ResetPortalDomainVerification(stepCtx, db.ResetPortalDomainVerificationParams{
			ID:                 domainID,
			VerificationStatus: db.PortalDomainsVerificationStatusPending,
			CheckAttempts:      0,
			InvocationID:       sql.NullString{Valid: false, String: ""},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("reset verification"))
	if err != nil {
		return nil, err
	}

	if _, err := s.VerifyDomain(ctx, &hydrav1.VerifyPortalDomainRequest{}); err != nil {
		return nil, err
	}

	return &hydrav1.RetryPortalDomainVerificationResponse{}, nil
}

// onVerificationSuccess marks the row verified, queues its ACME challenge,
// revokes any other workspace's claim, routes the hostname to the portal app,
// and starts issuance last so it never sees two verified rows or a domain
// whose route conflicted.
func (s *Service) onVerificationSuccess(
	ctx restate.ObjectContext,
	dom db.PortalDomain,
) (*hydrav1.VerifyPortalDomainResponse, error) {
	// Checked before marking the row verified so a misconfigured worker leaves
	// it retryable rather than verified with no route.
	if s.environmentID == "" {
		return nil, fault.New("portal environment_id is not configured")
	}

	now := time.Now().UnixMilli()

	err := restate.RunVoid(ctx, func(stepCtx restate.RunContext) error {
		return s.db.UpdatePortalDomainVerificationStatus(stepCtx, db.UpdatePortalDomainVerificationStatusParams{
			ID:                 dom.ID,
			VerificationStatus: db.PortalDomainsVerificationStatusVerified,
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: now},
		})
	}, restate.WithName("mark verified"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to mark portal domain as verified"))
	}

	// Token and authorization come from the ACME server during the challenge.
	err = restate.RunVoid(ctx, func(stepCtx restate.RunContext) error {
		return s.db.InsertAcmeChallenge(stepCtx, db.InsertAcmeChallengeParams{
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

	// Every tenant's route shares the portal project and environment, so the
	// replay check in CreateRoute cannot tell tenants apart. That is safe only
	// because the revoke above already removed any other workspace's route.
	err = restate.RunVoid(ctx, func(stepCtx restate.RunContext) error {
		return s.createFrontlineRoute(stepCtx, dom, now)
	}, restate.WithName("create frontline route"))
	if err != nil {
		if restate.IsTerminalError(err) {
			return nil, err
		}
		return nil, fault.Wrap(err, fault.Internal("failed to create frontline route"))
	}

	// The certificate belongs to the tenant, so it is issued under the tenant
	// workspace, never the portal app's.
	hydrav1.NewCertificateServiceClient(ctx, dom.Domain).ProcessChallenge().Send(&hydrav1.ProcessChallengeRequest{
		WorkspaceId: dom.WorkspaceID,
		Domain:      dom.Domain,
	})

	logger.Info("portal domain verification completed successfully", "domain", dom.Domain)

	return &hydrav1.VerifyPortalDomainResponse{}, nil
}

// createFrontlineRoute routes dom's hostname to the portal app's live
// deployment in the configured environment.
func (s *Service) createFrontlineRoute(ctx context.Context, dom db.PortalDomain, nowMs int64) error {
	env, err := s.db.FindEnvironmentById(ctx, s.environmentID)
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to find portal environment"))
	}

	app, err := s.db.FindAppById(ctx, env.AppID)
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to find portal app"))
	}

	deploymentID := ""
	if app.CurrentDeploymentID.Valid {
		deploymentID = app.CurrentDeploymentID.String
	}

	return domainverify.CreateRoute(ctx, s.db, domainverify.Route{
		DomainID:      dom.ID,
		Hostname:      dom.Domain,
		ProjectID:     env.ProjectID,
		AppID:         env.AppID,
		EnvironmentID: env.ID,
		DeploymentID:  deploymentID,
	}, nowMs, func(ctx context.Context, reason string) error {
		return s.db.UpdatePortalDomainFailed(ctx, db.UpdatePortalDomainFailedParams{
			ID:                 dom.ID,
			VerificationStatus: db.PortalDomainsVerificationStatusFailed,
			VerificationError:  sql.NullString{Valid: true, String: reason},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: nowMs},
		})
	})
}

// onVerificationFailed marks the row failed and stops retries.
func (s *Service) onVerificationFailed(
	ctx restate.ObjectContext,
	dom db.PortalDomain,
	errorMsg string,
) (*hydrav1.VerifyPortalDomainResponse, error) {
	err := restate.RunVoid(ctx, func(stepCtx restate.RunContext) error {
		return s.db.UpdatePortalDomainFailed(stepCtx, db.UpdatePortalDomainFailedParams{
			ID:                 dom.ID,
			VerificationStatus: db.PortalDomainsVerificationStatusFailed,
			VerificationError:  sql.NullString{Valid: true, String: errorMsg},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("mark failed"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to mark portal domain as failed"))
	}

	logger.Info("portal domain verification failed", "domain", dom.Domain, "error", errorMsg)

	return nil, restate.ToTerminalError(fmt.Errorf("portal domain verification failed: %s", errorMsg))
}
