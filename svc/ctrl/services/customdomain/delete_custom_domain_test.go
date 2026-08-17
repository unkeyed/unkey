package customdomain

import (
	"context"
	"encoding/json"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestDeleteCustomDomainWritesAuditLog pins that the deletion and its audit entry
// commit together, mirroring the create side. A domain created and then deleted
// leaves no row but a full trail: one create entry, one delete entry.
func TestDeleteCustomDomainWritesAuditLog(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)
	svc := f.newService(t)

	created, err := svc.AddCustomDomain(ctx, f.request(f.domain))
	require.NoError(t, err)

	deleteReq := f.deleteRequest(f.domain, f.projectID)
	_, err = svc.DeleteCustomDomain(ctx, deleteReq)
	require.NoError(t, err)

	_, err = f.database.FindCustomDomainByWorkspaceAndDomain(ctx, db.FindCustomDomainByWorkspaceAndDomainParams{
		WorkspaceID: f.workspaceID,
		Domain:      f.domain,
	})
	require.True(t, db.IsNotFound(err), "the domain row must be gone, got: %v", err)

	logged := f.findAuditEvent(t, auditlog.DomainDeleteEvent)
	require.Equal(t, f.workspaceID, logged.WorkspaceID)
	require.Equal(t, string(auditlog.UserActor), logged.Actor.Type)
	require.Equal(t, deleteReq.Msg.GetActor().GetId(), logged.Actor.ID)
	require.Len(t, logged.Targets, 1)
	require.Equal(t, string(auditlog.DomainResourceType), logged.Targets[0].Type)
	require.Equal(t, created.Msg.GetDomainId(), logged.Targets[0].ID)
	require.Equal(t, f.domain, logged.Targets[0].Meta["domain"])
	require.Equal(t, f.projectID, logged.Targets[0].Meta["projectId"])
	require.Equal(t, f.appID, logged.Targets[0].Meta["appId"])
	require.Equal(t, f.environmentID, logged.Targets[0].Meta["environmentId"])
}

// TestDeleteCustomDomainAttributesMissingActorToSystem pins the same degradation
// rule as create: the actor field is new on DeleteCustomDomainRequest, so a caller
// not yet redeployed sends none, and its deletion must still be audited rather
// than rejected or silently unlogged.
func TestDeleteCustomDomainAttributesMissingActorToSystem(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)
	svc := f.newService(t)

	_, err := svc.AddCustomDomain(ctx, f.request(f.domain))
	require.NoError(t, err)

	req := connect.NewRequest(&ctrlv1.DeleteCustomDomainRequest{
		WorkspaceId: f.workspaceID,
		ProjectId:   f.projectID,
		Domain:      f.domain,
		Actor:       nil,
	})
	req.Header().Set("Authorization", "Bearer "+testBearer)

	_, err = svc.DeleteCustomDomain(ctx, req)
	require.NoError(t, err)

	logged := f.findAuditEvent(t, auditlog.DomainDeleteEvent)
	require.Equal(t, string(auditlog.SystemActor), logged.Actor.Type)
	require.Empty(t, logged.Actor.ID)
}

// TestDeleteCustomDomainRollsBackWhenAuditInsertFails pins the guarantee the shared
// transaction exists for: a deletion that cannot be audited does not happen. Without
// it a failed audit insert would leave the domain gone with no trail saying who
// removed it.
func TestDeleteCustomDomainRollsBackWhenAuditInsertFails(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)

	_, err := f.newService(t).AddCustomDomain(ctx, f.request(f.domain))
	require.NoError(t, err)

	// Same service shape, but every audit insert fails. The create above used a
	// working service, so only the delete's entry is refused.
	auditBroken := New(Config{
		Database:                   f.database,
		Restate:                    nil,
		RestateAdmin:               nil,
		Auditlogs:                  erroringAuditLogs{},
		CnameDomain:                "cname.unkey.local",
		DomainConnectPrivateKeyPEM: nil,
		Bearer:                     testBearer,
	})

	_, err = auditBroken.DeleteCustomDomain(ctx, f.deleteRequest(f.domain, f.projectID))
	require.Error(t, err)
	require.ErrorIs(t, err, errInjectedAuditInsert)
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code())

	// The rollback must restore the domain, so the caller can retry the delete.
	_, err = f.database.FindCustomDomainByWorkspaceAndDomain(ctx, db.FindCustomDomainByWorkspaceAndDomainParams{
		WorkspaceID: f.workspaceID,
		Domain:      f.domain,
	})
	require.NoError(t, err, "an unaudited deletion must roll back and leave the domain in place")

	// No delete entry was committed; only the create's remains.
	outboxRows, err := f.database.ListClickhouseOutboxByWorkspace(ctx, f.workspaceID)
	require.NoError(t, err)
	require.Len(t, outboxRows, 1)

	var logged auditlog.Event
	require.NoError(t, json.Unmarshal(outboxRows[0].Payload, &logged))
	require.Equal(t, string(auditlog.DomainCreateEvent), logged.Event)

	// The domain survived intact, so the retry deletes and audits it.
	_, err = f.newService(t).DeleteCustomDomain(ctx, f.deleteRequest(f.domain, f.projectID))
	require.NoError(t, err)
	f.findAuditEvent(t, auditlog.DomainDeleteEvent)
}

// TestDeleteCustomDomainNotFound covers every missing-domain answer: a name that
// was never attached, a project that does not own the domain, and a repeat of a
// successful delete. All three must read as NotFound, so none of them can be used
// to probe what exists.
func TestDeleteCustomDomainNotFound(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, 1)
	svc := f.newService(t)

	_, err := svc.AddCustomDomain(ctx, f.request(f.domain))
	require.NoError(t, err)

	requireNotFound := func(t *testing.T, domain, projectID string) {
		t.Helper()
		_, err := svc.DeleteCustomDomain(ctx, f.deleteRequest(domain, projectID))
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

		// The mismatch must not have deleted anything.
		_, err := f.database.FindCustomDomainByWorkspaceAndDomain(ctx, db.FindCustomDomainByWorkspaceAndDomainParams{
			WorkspaceID: f.workspaceID,
			Domain:      f.domain,
		})
		require.NoError(t, err, "a project mismatch must leave the domain in place")
	})

	t.Run("second delete", func(t *testing.T) {
		_, err := svc.DeleteCustomDomain(ctx, f.deleteRequest(f.domain, f.projectID))
		require.NoError(t, err)
		requireNotFound(t, f.domain, f.projectID)
	})
}

func (f fixture) deleteRequest(domain, projectID string) *connect.Request[ctrlv1.DeleteCustomDomainRequest] {
	req := connect.NewRequest(&ctrlv1.DeleteCustomDomainRequest{
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

// findAuditEvent returns the single outbox entry carrying event, failing the test
// when none or several match.
func (f fixture) findAuditEvent(t *testing.T, event auditlog.AuditLogEvent) auditlog.Event {
	t.Helper()

	outboxRows, err := f.database.ListClickhouseOutboxByWorkspace(context.Background(), f.workspaceID)
	require.NoError(t, err)

	var matches []auditlog.Event
	for _, row := range outboxRows {
		var logged auditlog.Event
		require.NoError(t, json.Unmarshal(row.Payload, &logged))
		if logged.Event == string(event) {
			matches = append(matches, logged)
		}
	}
	require.Len(t, matches, 1, "expected exactly one %s entry", event)
	return matches[0]
}

// erroringAuditLogs refuses every insert, standing in for an audit store that is down.
type erroringAuditLogs struct{}

func (erroringAuditLogs) Insert(context.Context, db.DBTX, []auditlog.AuditLog) error {
	return errInjectedAuditInsert
}
