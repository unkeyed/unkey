package cluster

import (
	"context"
	"database/sql"
	"time"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// infraWorkspaceID owns platform-managed wildcard certificates. Using a
// synthetic workspace keeps infra domains separate from customer domain
// records and lets them share a single ACME account.
const infraWorkspaceID = "unkey_internal"

// infraTriggerTimeout bounds the best-effort Restate trigger so a slow or hung
// ingress can never stall the caller (a Heartbeat RPC or server startup). If the
// trigger doesn't land in time the 'waiting' challenge is already durable, so the
// renewal cron issues the certificate on its next tick.
const infraTriggerTimeout = 5 * time.Second

// EnsureInfraCertificate provisions a wildcard certificate for a platform
// infrastructure domain and triggers issuance immediately.
//
// It is fully idempotent and safe to call on every Heartbeat: once the
// custom_domains + acme_challenges records exist it is a single-read no-op. The
// caller must keep calling it (not gate on a one-shot "region created" signal),
// because that repeated call is the retry mechanism: any write that fails here
// leaves no durable backstop, so the next call must re-attempt it.
//
// The 'waiting' acme_challenges row is the durable backstop the whole design
// rests on: RenewExpiringCertificates (ListExecutableChallenges matches
// status = 'waiting') re-triggers issuance regardless of whether the direct
// Restate trigger below ever succeeds. domain_id is unique on acme_challenges,
// so exactly one challenge exists per domain and it survives issuance as
// 'verified'. A non-'failed' row is the "already provisioned" signal.
//
// A 'failed' challenge is the one state the cron can't recover (it matches only
// 'waiting'/expiring-'verified'), so this resets it back to 'waiting' and
// re-triggers, bounded by the provisioned-cert cache TTL so a persistently
// failing domain retries on a sane cadence instead of on every heartbeat.
//
// This is best-effort and never returns an error: cert provisioning must not
// fail region registration (Heartbeat) or server startup. Failures are logged
// and retried on the next call.
//
// Infra wildcards are pre-verified because we control their DNS via Route53,
// so they use DNS-01 challenges and skip the customer verification flow.
func (s *Service) EnsureInfraCertificate(ctx context.Context, domain string) {
	// Cached fast path: we've already confirmed the records exist for this
	// domain, so skip the DB read entirely. This is the steady-state branch on
	// every heartbeat. Only confirmed-provisioned domains are cached below, so a
	// prior failure is never cached and still retries here.
	if _, hit := s.provisionedCerts.Get(ctx, domain); hit == cache.Hit {
		return
	}

	challenge, err := s.db.FindAcmeChallengeByDomain(ctx, domain)
	switch {
	case err == nil && challenge.Status == db.AcmeChallengesStatusFailed:
		// A prior issuance exhausted its retries and marked the challenge
		// 'failed'. Nothing else recovers it: ListExecutableChallenges skips
		// 'failed', so the renewal cron won't retry, and no path resets it, so
		// the region wildcard would stay unissued until someone edits the row by
		// hand. Flip it back to 'waiting' to restore the cron backstop, then
		// re-trigger. Cache it: the row is healthy again, and the TTL bounds how
		// often a domain that keeps failing is retried.
		if resetErr := s.db.UpdateAcmeChallengeStatus(ctx, db.UpdateAcmeChallengeStatusParams{
			Status:    db.AcmeChallengesStatusWaiting,
			UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			DomainID:  challenge.DomainID,
		}); resetErr != nil {
			logger.Error("failed to reset failed infra challenge", "error", resetErr, "domain", domain)
			return
		}
		s.provisionedCerts.Set(ctx, domain, true)
		s.triggerInfraIssuance(ctx, domain)
		return
	case err == nil:
		// waiting/pending/verified: the backstop is in place or the cert is
		// already issued, so the issuer/renewal cron owns it. Cache and return.
		s.provisionedCerts.Set(ctx, domain, true)
		return
	case !db.IsNotFound(err):
		logger.Error("failed to check for existing infra challenge", "error", err, "domain", domain)
		return
	}

	// No challenge yet: either this is the first attempt, or a previous attempt
	// failed before writing the challenge row. Create both records idempotently.
	now := time.Now().UnixMilli()

	// UpsertCustomDomain is keyed on (workspace_id, domain), so a domain row left
	// behind by a half-finished prior attempt is reused, not duplicated. The
	// generated ID is only used when this call actually inserts; on conflict the
	// existing row keeps its ID, which we read back below before inserting the
	// challenge.
	err = s.db.UpsertCustomDomain(ctx, db.UpsertCustomDomainParams{
		ID:                 uid.New(uid.DomainPrefix),
		WorkspaceID:        infraWorkspaceID,
		ProjectID:          infraWorkspaceID,
		AppID:              infraWorkspaceID,
		EnvironmentID:      infraWorkspaceID,
		Domain:             domain,
		ChallengeType:      db.CustomDomainsChallengeTypeDNS01,
		VerificationStatus: db.CustomDomainsVerificationStatusVerified, // pre-verified: we control DNS via Route53
		VerificationToken:  "",                                         // not needed for infra domains
		TargetCname:        uid.DNS1035(16),                            // unused for DNS-01 but required for uniqueness
		CreatedAt:          now,
		UpdatedAt:          sql.NullInt64{Int64: now, Valid: true},
	})
	if err != nil {
		logger.Error("failed to create infra domain", "error", err, "domain", domain)
		return
	}

	// Read back the row to get its real ID: the upsert above may have hit the
	// existing row, in which case the ID we generated was discarded.
	created, err := s.db.FindCustomDomainByDomain(ctx, domain)
	if err != nil {
		logger.Error("failed to load infra domain after upsert", "error", err, "domain", domain)
		return
	}

	// status 'waiting' so RenewExpiringCertificates picks it up as the backstop.
	// domain_id is unique, so a concurrent heartbeat that already inserted the
	// challenge makes this a duplicate-key error, which is success, not failure.
	err = s.db.InsertAcmeChallenge(ctx, db.InsertAcmeChallengeParams{
		WorkspaceID:   infraWorkspaceID,
		DomainID:      created.ID,
		Token:         "",
		Authorization: "",
		Status:        db.AcmeChallengesStatusWaiting,
		ChallengeType: db.AcmeChallengesChallengeTypeDNS01,
		CreatedAt:     now,
		UpdatedAt:     sql.NullInt64{Int64: now, Valid: true},
		ExpiresAt:     0, // set when the certificate is issued
	})
	if err != nil && !db.IsDuplicateKeyError(err) {
		logger.Error("failed to create ACME challenge for infra domain", "error", err, "domain", domain)
		return
	}

	// The backstop row is durable now (freshly inserted, or already there from a
	// racing heartbeat). Cache so later heartbeats skip the DB check.
	s.provisionedCerts.Set(ctx, domain, true)

	s.triggerInfraIssuance(ctx, domain)
	logger.Info("provisioned infra domain for certificate issuance", "domain", domain)
}

// triggerInfraIssuance fires CertificateService.ProcessChallenge for an infra
// domain so the certificate is issued within seconds rather than waiting for the
// next renewal cron tick. Best-effort and bounded: a slow ingress must not stall
// the caller, and a dropped trigger is covered by the 'waiting' challenge
// backstop (the renewal cron re-triggers it). Keyed by domain so concurrent
// triggers are serialized by Restate's virtual object model.
func (s *Service) triggerInfraIssuance(ctx context.Context, domain string) {
	triggerCtx, cancel := context.WithTimeout(ctx, infraTriggerTimeout)
	defer cancel()
	certClient := hydrav1.NewCertificateServiceIngressClient(s.restate, domain)
	if _, err := certClient.ProcessChallenge().Send(triggerCtx, &hydrav1.ProcessChallengeRequest{
		WorkspaceId: infraWorkspaceID,
		Domain:      domain,
	}); err != nil {
		logger.Warn("failed to trigger infra certificate issuance, renewal cron will retry",
			"domain", domain,
			"error", err,
		)
	}
}
