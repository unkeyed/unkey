package customdomain

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestRetryVerificationResetsAndWritesAuditLog pins that the reset and its audit
// entry commit together: the row returns to pending with a fresh invocation id and
// zeroed attempts, and exactly one domain.verify entry attributes the retry.
func TestRetryVerificationResetsAndWritesAuditLog(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)
	svc := f.newService(t)

	_, err := svc.AddCustomDomain(ctx, f.request(f.domain))
	require.NoError(t, err)
	f.markDomainStatus(t, db.CustomDomainsVerificationStatusFailed)

	retryReq := f.retryRequest(f.domain, f.projectID)
	res, err := svc.RetryVerification(ctx, retryReq)
	require.NoError(t, err)
	require.Equal(t, ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_PENDING, res.Msg.GetStatus())

	row, err := f.database.FindCustomDomainByWorkspaceAndDomain(ctx, db.FindCustomDomainByWorkspaceAndDomainParams{
		WorkspaceID: f.workspaceID,
		Domain:      f.domain,
	})
	require.NoError(t, err)
	require.Equal(t, db.CustomDomainsVerificationStatusPending, row.VerificationStatus)
	require.True(t, row.InvocationID.Valid)
	require.NotEmpty(t, row.InvocationID.String)

	logged := f.findAuditEvent(t, auditlog.DomainVerifyEvent)
	require.Equal(t, f.workspaceID, logged.WorkspaceID)
	require.Equal(t, string(auditlog.UserActor), logged.Actor.Type)
	require.Equal(t, retryReq.Msg.GetActor().GetId(), logged.Actor.ID)
	require.Len(t, logged.Targets, 1)
	require.Equal(t, string(auditlog.DomainResourceType), logged.Targets[0].Type)
	require.Equal(t, row.ID, logged.Targets[0].ID)
	require.Equal(t, f.domain, logged.Targets[0].Meta["domain"])
	require.Equal(t, f.projectID, logged.Targets[0].Meta["projectId"])
	require.Equal(t, f.appID, logged.Targets[0].Meta["appId"])
	require.Equal(t, f.environmentID, logged.Targets[0].Meta["environmentId"])
}

// A domain that gave up long ago carries residue: the timeout message, the last
// check stamp, and a spent attempt count. The retry has to clear all of it, or the
// caller polls a row that still reads as the previous failure. created_at is aged
// well past the verification window on purpose: ctrl must not gate on row age, and
// the worker anchors its 24h window to the invocation instead (pinned by
// TestVerifyDomainOldRowDoesNotTimeOut).
func TestRetryVerificationRestartsLongFailedDomain(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)
	svc := f.newService(t)

	_, err := svc.AddCustomDomain(ctx, f.request(f.domain))
	require.NoError(t, err)

	found, err := f.database.FindCustomDomainByWorkspaceAndDomain(ctx, db.FindCustomDomainByWorkspaceAndDomainParams{
		WorkspaceID: f.workspaceID,
		Domain:      f.domain,
	})
	require.NoError(t, err)
	created, err := f.database.FindCustomDomainById(ctx, found.ID)
	require.NoError(t, err)

	_, err = f.database.RW().ExecContext(ctx,
		"UPDATE custom_domains SET created_at = ? WHERE id = ?",
		time.Now().Add(-30*24*time.Hour).UnixMilli(), created.ID)
	require.NoError(t, err)

	require.NoError(t, f.database.UpdateCustomDomainCheckAttempt(ctx, db.UpdateCustomDomainCheckAttemptParams{
		ID:            created.ID,
		CheckAttempts: 1439,
		LastCheckedAt: sql.NullInt64{Valid: true, Int64: time.Now().Add(-29 * 24 * time.Hour).UnixMilli()},
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}))
	require.NoError(t, f.database.UpdateCustomDomainFailed(ctx, db.UpdateCustomDomainFailedParams{
		ID:                 created.ID,
		VerificationStatus: db.CustomDomainsVerificationStatusFailed,
		VerificationError:  sql.NullString{Valid: true, String: "domain verification timed out after 24 hours"},
		UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}))

	_, err = svc.RetryVerification(ctx, f.retryRequest(f.domain, f.projectID))
	require.NoError(t, err)

	row, err := f.database.FindCustomDomainById(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, db.CustomDomainsVerificationStatusPending, row.VerificationStatus)
	require.Zero(t, row.CheckAttempts, "a retry starts the attempt budget over")
	require.False(t, row.VerificationError.Valid, "the previous timeout message must not survive the retry")
	require.False(t, row.LastCheckedAt.Valid, "the previous check stamp must not survive the retry")
	require.True(t, row.InvocationID.Valid)
	require.NotEmpty(t, row.InvocationID.String, "the retry must be tracked by its own invocation")

	// The customer already published these records, and the retry exists because
	// they just fixed their DNS. Rotating either one would silently invalidate the
	// setup they are retrying against.
	require.Equal(t, created.VerificationToken, row.VerificationToken, "a retry must not rotate the TXT token")
	require.Equal(t, created.TargetCname, row.TargetCname, "a retry must not rotate the CNAME target")
}

// TestRetryVerificationRejectsVerifiedDomain pins the server-side guard. The dashboard
// only offers retry for failed domains, but that is a UX gate; this one protects a
// serving domain from any caller, and nothing may be written when it fires.
func TestRetryVerificationRejectsVerifiedDomain(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)
	svc := f.newService(t)

	_, err := svc.AddCustomDomain(ctx, f.request(f.domain))
	require.NoError(t, err)
	f.markDomainStatus(t, db.CustomDomainsVerificationStatusVerified)

	_, err = svc.RetryVerification(ctx, f.retryRequest(f.domain, f.projectID))
	require.Error(t, err)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeFailedPrecondition, connectErr.Code())
	require.Contains(t, connectErr.Message(), "already verified")

	row, err := f.database.FindCustomDomainByWorkspaceAndDomain(ctx, db.FindCustomDomainByWorkspaceAndDomainParams{
		WorkspaceID: f.workspaceID,
		Domain:      f.domain,
	})
	require.NoError(t, err)
	require.Equal(t, db.CustomDomainsVerificationStatusVerified, row.VerificationStatus,
		"a rejected retry must leave the row untouched")

	// Only the create's entry exists; the rejection wrote nothing.
	f.findAuditEvent(t, auditlog.DomainCreateEvent)
	f.requireNoAuditEvent(t, auditlog.DomainVerifyEvent)
}

// TestRetryVerificationRollsBackWhenAuditInsertFails pins the guarantee the shared
// transaction exists for: a retry that cannot be audited does not happen, so the row
// keeps its failed state and the caller can try again.
func TestRetryVerificationRollsBackWhenAuditInsertFails(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)

	_, err := f.newService(t).AddCustomDomain(ctx, f.request(f.domain))
	require.NoError(t, err)
	f.markDomainStatus(t, db.CustomDomainsVerificationStatusFailed)

	// Same service shape, but every audit insert fails. The create above used a
	// working service, so only the retry's entry is refused.
	auditBroken := New(Config{
		Database:                   f.database,
		Restate:                    nil,
		RestateAdmin:               nil,
		Auditlogs:                  erroringAuditLogs{},
		CnameDomain:                "cname.unkey.local",
		DomainConnectPrivateKeyPEM: nil,
		Bearer:                     testBearer,
	})

	_, err = auditBroken.RetryVerification(ctx, f.retryRequest(f.domain, f.projectID))
	require.Error(t, err)
	require.ErrorIs(t, err, errInjectedAuditInsert)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code())

	row, err := f.database.FindCustomDomainByWorkspaceAndDomain(ctx, db.FindCustomDomainByWorkspaceAndDomainParams{
		WorkspaceID: f.workspaceID,
		Domain:      f.domain,
	})
	require.NoError(t, err)
	require.Equal(t, db.CustomDomainsVerificationStatusFailed, row.VerificationStatus,
		"an unaudited retry must roll back and leave the row failed")
	f.requireNoAuditEvent(t, auditlog.DomainVerifyEvent)
}

func TestRetryVerificationAttributesMissingActorToSystem(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)
	svc := f.newService(t)

	_, err := svc.AddCustomDomain(ctx, f.request(f.domain))
	require.NoError(t, err)
	f.markDomainStatus(t, db.CustomDomainsVerificationStatusFailed)

	req := connect.NewRequest(&ctrlv1.RetryVerificationRequest{
		WorkspaceId: f.workspaceID,
		ProjectId:   f.projectID,
		Domain:      f.domain,
		Actor:       nil,
	})
	req.Header().Set("Authorization", "Bearer "+testBearer)

	_, err = svc.RetryVerification(ctx, req)
	require.NoError(t, err)

	logged := f.findAuditEvent(t, auditlog.DomainVerifyEvent)
	require.Equal(t, string(auditlog.SystemActor), logged.Actor.Type)
	require.Empty(t, logged.Actor.ID)
}

// TestRetryVerificationNotFound covers every missing-domain answer: a name that was
// never attached and a project that does not own the domain. Both must read as
// NotFound, so neither can be used to probe what exists.
func TestRetryVerificationNotFound(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)
	svc := f.newService(t)

	_, err := svc.AddCustomDomain(ctx, f.request(f.domain))
	require.NoError(t, err)

	requireNotFound := func(t *testing.T, domain, projectID string) {
		t.Helper()
		_, err := svc.RetryVerification(ctx, f.retryRequest(domain, projectID))
		require.Error(t, err)
		var connectErr *connect.Error
		require.ErrorAs(t, err, &connectErr)
		require.Equal(t, connect.CodeNotFound, connectErr.Code())
	}

	t.Run("never attached", func(t *testing.T) {
		requireNotFound(t, randomDomain(), f.projectID)
	})

	t.Run("project mismatch", func(t *testing.T) {
		requireNotFound(t, f.domain, uid.New(uid.ProjectPrefix))
	})
}

func (f fixture) retryRequest(domain, projectID string) *connect.Request[ctrlv1.RetryVerificationRequest] {
	req := connect.NewRequest(&ctrlv1.RetryVerificationRequest{
		WorkspaceId: f.workspaceID,
		ProjectId:   projectID,
		Domain:      domain,
		Actor: &ctrlv1.ActorInfo{
			Id:        uid.New("user"),
			Name:      "Test User",
			Type:      ctrlv1.ActorType_ACTOR_TYPE_USER,
			RemoteIp:  "127.0.0.1",
			UserAgent: "test-agent",
			Meta:      map[string]string{},
		},
	})
	req.Header().Set("Authorization", "Bearer "+testBearer)
	return req
}

// markDomainStatus moves f's domain into status directly, standing in for the
// verification workflow outcome the service under test reacts to.
func (f fixture) markDomainStatus(t *testing.T, status db.CustomDomainsVerificationStatus) {
	t.Helper()

	row, err := f.database.FindCustomDomainByWorkspaceAndDomain(context.Background(), db.FindCustomDomainByWorkspaceAndDomainParams{
		WorkspaceID: f.workspaceID,
		Domain:      f.domain,
	})
	require.NoError(t, err)
	require.NoError(t, f.database.UpdateCustomDomainVerificationStatus(context.Background(), db.UpdateCustomDomainVerificationStatusParams{
		ID:                 row.ID,
		VerificationStatus: status,
		UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}))
}

// requireNoAuditEvent fails when any outbox entry carries event.
func (f fixture) requireNoAuditEvent(t *testing.T, event auditlog.AuditLogEvent) {
	t.Helper()

	outboxRows, err := f.database.ListClickhouseOutboxByWorkspace(context.Background(), f.workspaceID)
	require.NoError(t, err)
	for _, row := range outboxRows {
		require.NotContains(t, string(row.Payload), `"event":"`+string(event)+`"`)
	}
}
