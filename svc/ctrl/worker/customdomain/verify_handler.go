package customdomain

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/dns"
	"github.com/unkeyed/unkey/pkg/dns/domainconnect"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
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

// VerifyDomain verifies a custom domain for routing.
//
// Verification has two paths:
//   - CNAME path: the domain has a visible CNAME record pointing to its unique target
//     (e.g. <random>.unkey-dns.com). This works for subdomains. No TXT record needed.
//   - TXT path: for apex domains using CNAME flattening, ALIAS/ANAME records, or
//     Cloudflare proxy, the CNAME is not visible via DNS lookup. In this case, a TXT
//     record at _unkey.<domain> proves ownership, and the apex must additionally
//     resolve to at least one A/AAAA address (sanity check that the alias record is
//     actually configured — TXT alone proves ownership but not routing).
//
// TXT is also always required when another workspace already has the same domain
// verified (contention), regardless of whether CNAME is visible.
//
// This is a Restate virtual object handler keyed by domain name, ensuring only one
// verification workflow runs per domain at any time. The handler checks DNS once
// per invocation - Restate's retry policy handles periodic re-checks (every 1 minute
// for up to 24 hours).
//
// Once verification succeeds, one transaction:
// 1. Revokes any existing verified domain from another workspace (contention case)
// 2. Updates domain status to "verified"
// 3. Creates an ACME challenge record to trigger certificate issuance
// 4. Creates a frontline route to enable traffic routing
//
// If verification is still incomplete after 24 hours, the domain is marked failed.
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

	// The row, the clock, and DNS all change between attempts. Branching on them
	// outside a step makes replays diverge from the journal, so the whole check is
	// one step that Restate retries until it journals an outcome.
	check, err := restate.Run(ctx, func(runCtx restate.RunContext) (domainCheck, error) {
		return s.checkDomain(runCtx, domainID, startedAt)
	}, restate.WithName("check domain"))
	if err != nil {
		return nil, err
	}

	if check.TimedOut {
		return s.onVerificationFailed(ctx, check.Domain, "domain verification timed out after 24 hours")
	}
	return s.onVerificationSuccess(ctx, check.Domain, check.OwnershipProven)
}

// domainCheck is the journaled outcome of [Service.checkDomain]. Without
// TimedOut, Domain passed verification.
type domainCheck struct {
	Domain   db.CustomDomain
	TimedOut bool
	// OwnershipProven is set when the TXT record was checked and matched.
	OwnershipProven bool
}

// checkDomain runs one verification attempt. It returns errNotVerified while
// DNS is not ready, which fails the step so Restate retries the invocation.
func (s *Service) checkDomain(ctx context.Context, domainID string, startedAt time.Time) (domainCheck, error) {
	dom, err := s.db.FindCustomDomainById(ctx, domainID)
	if err != nil {
		if db.IsNotFound(err) {
			if time.Since(startedAt) < rowVisibilityGrace {
				return domainCheck{}, fault.Wrap(err, fault.Internal("domain row not visible yet"))
			}

			// The domain row was deleted (DeleteCustomDomain, environment cascade,
			// etc.) while verification was still running. Stop retrying instead of
			// surfacing a retryable internal error for up to 24 hours.
			logger.Info("domain record deleted, stopping verification workflow",
				"domain_id", domainID,
			)
			return domainCheck{}, restate.ToTerminalError(fmt.Errorf("domain record not found: %s", domainID), restate.WithErrorCode(404))
		}
		return domainCheck{}, fault.Wrap(err, fault.Internal("failed to fetch domain record"))
	}

	elapsed := time.Since(startedAt)
	if elapsed > maxVerificationDuration {
		return domainCheck{Domain: dom, TimedOut: true, OwnershipProven: false}, nil
	}

	err = s.db.UpdateCustomDomainVerificationStatus(ctx, db.UpdateCustomDomainVerificationStatusParams{
		ID:                 dom.ID,
		VerificationStatus: db.CustomDomainsVerificationStatusVerifying,
		UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	})
	if err != nil {
		return domainCheck{}, err
	}

	// Step 1: Try CNAME verification. This works for subdomains with a real CNAME record.
	// Apex domains (CNAME flattening, ALIAS/ANAME, CF proxy) won't have a visible CNAME.
	cnameVerified, cnameErr := s.checkCNAME(ctx, dom.Domain, dom.TargetCname)
	if cnameErr != nil {
		logger.Warn("CNAME check error",
			"domain", dom.Domain,
			"error", cnameErr,
			"elapsed", elapsed,
		)
	}

	// Step 2: Check if TXT verification is needed.
	// TXT is required when:
	//   a) CNAME lookup failed (apex/flattened/proxied domains)
	//   b) Another workspace already has this domain verified (contention)
	contested := false
	requiresTxt := !cnameVerified

	if !requiresTxt {
		// Even if CNAME passed, check for contention — TXT is required to claim
		// a domain from another workspace
		_, contestErr := s.db.FindVerifiedCustomDomainByDomainExcludingWorkspace(ctx, db.FindVerifiedCustomDomainByDomainExcludingWorkspaceParams{
			Domain:      dom.Domain,
			WorkspaceID: dom.WorkspaceID,
		})
		if contestErr != nil && !db.IsNotFound(contestErr) {
			return domainCheck{}, fault.Wrap(contestErr, fault.Internal("failed to check domain contention"))
		}
		if contestErr == nil {
			contested = true
			requiresTxt = true
		}
	}

	txtVerified := true // default: not required
	if requiresTxt {
		var txtErr error
		txtVerified, txtErr = s.checkTXTRecord(ctx, dom.Domain, dom.VerificationToken)
		if txtErr != nil {
			logger.Warn("TXT check error",
				"domain", dom.Domain,
				"error", txtErr,
				"elapsed", elapsed,
			)
		}
	}

	// Apex domains can't expose a verifiable CNAME (ALIAS/ANAME, CNAME flattening,
	// or CF proxy hide it from the resolver), so they fall through to the TXT path.
	// TXT alone proves ownership but not that the user actually configured the
	// alias record needed for routing — require the apex to resolve to at least
	// one A/AAAA address as a sanity check.
	isApex := domainconnect.IsApexDomain(dom.Domain)
	apexHasRecords := true
	if isApex {
		var recordsErr error
		apexHasRecords, recordsErr = s.hasAddressRecords(ctx, dom.Domain)
		if recordsErr != nil {
			logger.Warn("apex DNS lookup error",
				"domain", dom.Domain,
				"error", recordsErr,
				"elapsed", elapsed,
			)
		}
	}

	err = s.db.UpdateCustomDomainCheckAttempt(ctx, db.UpdateCustomDomainCheckAttemptParams{
		ID:            dom.ID,
		CheckAttempts: dom.CheckAttempts + 1,
		LastCheckedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	})
	if err != nil {
		return domainCheck{}, err
	}

	err = s.db.UpdateCustomDomainOwnership(ctx, db.UpdateCustomDomainOwnershipParams{
		OwnershipVerified: txtVerified,
		CnameVerified:     cnameVerified,
		UpdatedAt:         sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		ID:                dom.ID,
	})
	if err != nil {
		return domainCheck{}, err
	}

	// Log current status
	logger.Info("DNS verification check complete",
		"domain", dom.Domain,
		"is_apex", isApex,
		"contested", contested,
		"requires_txt", requiresTxt,
		"txt_verified", txtVerified,
		"cname_verified", cnameVerified,
		"apex_has_records", apexHasRecords,
		"attempts", dom.CheckAttempts+1,
		"elapsed", elapsed,
	)

	// Verified when: CNAME matches OR TXT proves ownership (for apex/flattened/proxied)
	// If contested, both CNAME (or TXT) AND TXT ownership must pass
	// For apex: also require the apex to resolve to A/AAAA records
	verified := cnameVerified || txtVerified
	if contested {
		verified = txtVerified // TXT is the gate for contention
	}
	if isApex {
		verified = verified && apexHasRecords
	}
	if !verified {
		return domainCheck{}, errNotVerified
	}
	return domainCheck{Domain: dom, TimedOut: false, OwnershipProven: requiresTxt && txtVerified}, nil
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

// checkTXTRecord verifies that the domain has a TXT record proving ownership.
// The TXT record should be at _unkey.<domain> with value "unkey-domain-verify=<token>".
// Returns (true, nil) if verified, (false, nil) if TXT doesn't exist or doesn't match.
func (s *Service) checkTXTRecord(ctx context.Context, domain, expectedToken string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, dns.DefaultTimeout)
	defer cancel()

	txtRecords, err := dns.LookupTXT(ctx, dns.OwnershipTXTName(domain))
	if err != nil {
		if dns.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}

	expected := dns.OwnershipTXTValue(expectedToken)
	for _, txt := range txtRecords {
		if strings.EqualFold(txt, expected) {
			return true, nil
		}
	}
	return false, nil
}

// checkCNAME verifies that the domain has a CNAME record pointing to the expected target.
// Returns (true, nil) if verified, (false, nil) if CNAME doesn't exist or doesn't match.
//
// Note: This only works for subdomains with a real CNAME record. Apex domains using
// CNAME flattening, ALIAS/ANAME records, or Cloudflare proxy will not have a visible
// CNAME — those domains fall back to TXT-based ownership verification combined with
// hasAddressRecords to prove DNS is configured.
func (s *Service) checkCNAME(ctx context.Context, domain, expectedCname string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, dns.DefaultTimeout)
	defer cancel()

	// LookupCNAME already normalizes, just need to normalize expected
	expectedCname = strings.TrimSuffix(expectedCname, ".")
	expectedCname = strings.ToLower(expectedCname)

	cname, err := dns.LookupCNAME(ctx, domain)
	if err != nil {
		if dns.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}

	// LookupCNAME returns the domain itself when no CNAME record exists
	if cname == strings.ToLower(strings.TrimSuffix(domain, ".")) {
		return false, nil
	}

	return cname == expectedCname, nil
}

// hasAddressRecords returns true if the domain resolves to at least one A or
// AAAA address. Used as an apex-domain sanity check: TXT alone proves ownership
// but not that the user configured the actual ALIAS/ANAME/A record needed for
// routing. We don't compare addresses to a target — proxies (Cloudflare) and
// shared CDN IPs make that unreliable — we only confirm DNS is set up.
func (s *Service) hasAddressRecords(ctx context.Context, domain string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, dns.DefaultTimeout)
	defer cancel()

	addrs, err := dns.LookupHost(ctx, domain)
	if err != nil {
		if dns.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return len(addrs) > 0, nil
}

// verifyOutcome is the journaled result of [Service.completeVerification].
type verifyOutcome string

const (
	verifyOutcomeRouted   verifyOutcome = "routed"
	verifyOutcomeConflict verifyOutcome = "conflict"
	verifyOutcomeDeleted  verifyOutcome = "deleted"
)

// onVerificationSuccess marks a verified domain, routes it, and starts
// certificate issuance.
func (s *Service) onVerificationSuccess(
	ctx restate.ObjectContext,
	dom db.CustomDomain,
	ownershipProven bool,
) (*hydrav1.VerifyDomainResponse, error) {
	now := time.Now().UnixMilli()

	outcome, err := restate.Run(ctx, func(stepCtx restate.RunContext) (verifyOutcome, error) {
		return s.completeVerification(stepCtx, dom, ownershipProven, now)
	}, restate.WithName("complete verification"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to complete domain verification"))
	}

	switch outcome {
	case verifyOutcomeDeleted:
		logger.Info("domain record deleted, stopping verification workflow", "domain_id", dom.ID)
		return nil, restate.ToTerminalError(fmt.Errorf("domain record not found: %s", dom.ID), restate.WithErrorCode(404))
	case verifyOutcomeConflict:
		return s.onVerificationFailed(ctx, dom, "domain is already routed to another app")
	case verifyOutcomeRouted:
	}

	hydrav1.NewCertificateServiceClient(ctx, dom.Domain).ProcessChallenge().Send(&hydrav1.ProcessChallengeRequest{
		WorkspaceId: dom.WorkspaceID,
		Domain:      dom.Domain,
	})

	logger.Info("domain verification completed successfully",
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

// completeVerification applies every write for a verified domain in one
// transaction, so a retry after an unjournaled commit finds its own work and
// a concurrent delete or claim sees all of it or none of it.
//
// The share lock on the domain row makes DeleteCustomDomain wait for this
// transaction, or makes this one see the row gone. Two workspaces verifying
// the same FQDN at once serialize on the unique route: the loser's insert
// fails, the step retries, and it then sees the winner's committed claim. When
// another workspace has verified the domain, the later verifier takes it over
// only with TXT proof. The DNS check is journaled and does not rerun when this
// step retries, so a claim that commits in between must be checked here.
func (s *Service) completeVerification(ctx context.Context, dom db.CustomDomain, ownershipProven bool, now int64) (verifyOutcome, error) {
	app, err := s.db.FindAppById(ctx, dom.AppID)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("failed to find app for frontline route"))
	}

	deploymentID := ""
	if app.CurrentDeploymentID.Valid {
		deploymentID = app.CurrentDeploymentID.String
	}

	return db.TxWithResultRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) (verifyOutcome, error) {
		q := db.NewQueries(tx)

		if _, lockErr := q.LockCustomDomain(txCtx, dom.ID); lockErr != nil {
			if db.IsNotFound(lockErr) {
				return verifyOutcomeDeleted, nil
			}
			return "", lockErr
		}

		contested, contestErr := q.FindVerifiedCustomDomainByDomainExcludingWorkspace(txCtx, db.FindVerifiedCustomDomainByDomainExcludingWorkspaceParams{
			Domain:      dom.Domain,
			WorkspaceID: dom.WorkspaceID,
		})
		hasContested := contestErr == nil
		if contestErr != nil && !db.IsNotFound(contestErr) {
			return "", contestErr
		}
		if hasContested && !ownershipProven {
			return verifyOutcomeConflict, nil
		}

		routeExists := false
		route, routeErr := q.FindFrontlineRouteByFQDN(txCtx, dom.Domain)
		switch {
		case routeErr == nil:
			routeExists = true
			ours := route.AppID == dom.AppID && route.EnvironmentID == dom.EnvironmentID
			takeover := hasContested && route.ProjectID == contested.ProjectID
			if !ours && !takeover {
				return verifyOutcomeConflict, nil
			}
		case !db.IsNotFound(routeErr):
			return "", routeErr
		}

		if hasContested {
			logger.Info("revoking domain from previous workspace",
				"domain", dom.Domain,
				"old_workspace", contested.WorkspaceID,
				"new_workspace", dom.WorkspaceID,
			)
			if err := q.UpdateCustomDomainFailed(txCtx, db.UpdateCustomDomainFailedParams{
				ID:                 contested.ID,
				VerificationStatus: db.CustomDomainsVerificationStatusFailed,
				VerificationError:  sql.NullString{Valid: true, String: "domain claimed by another workspace"},
				UpdatedAt:          sql.NullInt64{Valid: true, Int64: now},
			}); err != nil {
				return "", err
			}
			if err := q.DeleteAcmeChallengeByDomainID(txCtx, contested.ID); err != nil && !db.IsNotFound(err) {
				return "", err
			}
			if routeExists && route.ProjectID == contested.ProjectID {
				if err := q.DeleteFrontlineRouteByFQDNAndProject(txCtx, db.DeleteFrontlineRouteByFQDNAndProjectParams{
					Fqdn:      dom.Domain,
					ProjectID: contested.ProjectID,
				}); err != nil && !db.IsNotFound(err) {
					return "", err
				}
				routeExists = false
			}
		}

		if err := q.UpdateCustomDomainVerificationStatus(txCtx, db.UpdateCustomDomainVerificationStatusParams{
			ID:                 dom.ID,
			VerificationStatus: db.CustomDomainsVerificationStatusVerified,
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: now},
		}); err != nil {
			return "", err
		}

		// Token and Authorization stay empty: the ACME server provides them
		// during the challenge flow. domain_id is unique, so an existing row
		// from an earlier attempt already satisfies this.
		err := q.InsertAcmeChallenge(txCtx, db.InsertAcmeChallengeParams{
			DomainID:      dom.ID,
			WorkspaceID:   dom.WorkspaceID,
			Token:         "",
			ChallengeType: db.AcmeChallengesChallengeTypeHTTP01,
			Authorization: "",
			Status:        db.AcmeChallengesStatusWaiting,
			ExpiresAt:     time.UnixMilli(now).Add(30 * 24 * time.Hour).UnixMilli(),
			CreatedAt:     now,
			UpdatedAt:     sql.NullInt64{Valid: true, Int64: now},
		})
		if err != nil && !db.IsDuplicateKeyError(err) {
			return "", err
		}

		if routeExists {
			return verifyOutcomeRouted, nil
		}
		return verifyOutcomeRouted, q.InsertFrontlineRoute(txCtx, db.InsertFrontlineRouteParams{
			ID:                       uid.New(uid.FrontlineRoutePrefix),
			ProjectID:                dom.ProjectID,
			AppID:                    dom.AppID,
			DeploymentID:             deploymentID,
			EnvironmentID:            dom.EnvironmentID,
			FullyQualifiedDomainName: dom.Domain,
			Sticky:                   db.FrontlineRoutesStickyLive,
			CreatedAt:                now,
			UpdatedAt:                sql.NullInt64{Valid: true, Int64: now},
		})
	})
}
