package gdprerasure_test

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// auditActor is what an audit row says about who acted.
type auditActor struct {
	ID        string
	Name      string
	Meta      string
	RemoteIP  string
	UserAgent string
}

// TestUserErasureRemovesSessionsAndPseudonymizesAuditLogs guarantees that a
// GDPR user erasure deletes the user's sessions, device approvals and signup
// credit claims and their WorkOS account, and strips their name, metadata,
// address and user agent from audit logs, both exported and still queued,
// while keeping the events and the actor id as the customer's security record.
// Another user's rows stay intact, and a second request changes nothing.
func TestUserErasureRemovesSessionsAndPseudonymizesAuditLogs(t *testing.T) {
	h := harness.New(t)
	ws := h.Seed.CreateWorkspace(h.Ctx)

	userID := uid.New(uid.TestPrefix)
	neighbourID := uid.New(uid.TestPrefix)
	for _, id := range []string{userID, neighbourID} {
		seedUser(t, h, ws.ID, id)
	}

	require.Equal(t, db.CountUserRowsByTableRow{CliSessions: 1, DeviceAuthorizations: 1, ComputeSignupCreditClaims: 1}, countUserRows(t, h, userID))
	neighbourBefore := countUserRows(t, h, neighbourID)
	neighbourOutboxBefore := outboxActors(t, h, ws.ID, neighbourID)
	neighbourAuditBefore := auditActors(t, h, ws.ID, neighbourID)

	_, err := erase(t, h, userRequest(userID))
	require.NoError(t, err)

	require.Equal(t, db.CountUserRowsByTableRow{}, countUserRows(t, h, userID))
	pseudonym := auditActor{ID: userID, Name: "", Meta: "{}", RemoteIP: "", UserAgent: ""}
	require.Equal(t, []auditActor{pseudonym}, outboxActors(t, h, ws.ID, userID))
	require.Equal(t, []auditActor{pseudonym}, auditActors(t, h, ws.ID, userID))
	require.Equal(t, []string{userID}, h.Erased.WorkOSUserIDs())

	require.Equal(t, neighbourBefore, countUserRows(t, h, neighbourID))
	require.Equal(t, neighbourOutboxBefore, outboxActors(t, h, ws.ID, neighbourID))
	require.Equal(t, neighbourAuditBefore, auditActors(t, h, ws.ID, neighbourID))

	again, err := erase(t, h, userRequest(userID))
	require.NoError(t, err)
	requireNothingLeft(t, again)
}

func userRequest(userID string) *hydrav1.GdprErasureRequest {
	return &hydrav1.GdprErasureRequest{Subject: &hydrav1.GdprErasureRequest_User{
		User: &hydrav1.GdprErasureUser{WorkosUserId: userID},
	}}
}

// seedUser gives a dashboard user a CLI session, an approved device
// authorization, a signup credit claim, a queued audit event and an exported
// audit row, each naming them.
func seedUser(t *testing.T, h *harness.Harness, workspaceID, userID string) {
	t.Helper()
	ctx := h.Ctx
	now := time.Now()
	const name, remoteIP, userAgent = "Ada Lovelace", "203.0.113.7", "unkey-cli/1.0"

	require.NoError(t, h.DB.InsertCliSession(ctx, db.InsertCliSessionParams{
		ID:               uid.New(uid.CLISessionPrefix),
		WorkspaceID:      workspaceID,
		UserID:           userID,
		UserName:         sql.NullString{String: name, Valid: true},
		Permissions:      json.RawMessage("[]"),
		RefreshTokenHash: uid.New(uid.TestPrefix),
		RefreshFamily:    uid.New(uid.TestPrefix),
		CreatedAt:        now.UnixMilli(),
		ExpiresAt:        now.Add(time.Hour).UnixMilli(),
	}))
	require.NoError(t, h.DB.InsertDeviceAuthorization(ctx, db.InsertDeviceAuthorizationParams{
		ID:             uid.New(uid.DeviceAuthorizationPrefix),
		UserCode:       uid.New(uid.TestPrefix),
		ExpiresAt:      now.Add(time.Hour).UnixMilli(),
		WorkspaceID:    sql.NullString{String: workspaceID, Valid: true},
		ApproverUserID: sql.NullString{String: userID, Valid: true},
		ApproverName:   sql.NullString{String: name, Valid: true},
		CreatedAt:      now.UnixMilli(),
	}))
	require.NoError(t, h.DB.InsertComputeSignupCreditClaim(ctx, db.InsertComputeSignupCreditClaimParams{
		CardFingerprint:  uid.New(uid.TestPrefix),
		WorkspaceID:      uid.New(uid.WorkspacePrefix),
		StripeCustomerID: uid.New(uid.TestPrefix),
		WorkosUserID:     userID,
		AttemptID:        uid.New(uid.TestPrefix),
		CreatedAt:        now.UnixMilli(),
	}))

	payload, err := json.Marshal(auditlog.Event{
		EventID:     uid.New(uid.AuditLogPrefix),
		Time:        now.UnixMilli(),
		WorkspaceID: workspaceID,
		Bucket:      "unkey_mutations",
		Source:      auditlog.EventSourcePlatform,
		Event:       "key.create",
		Actor:       auditlog.EventActor{Type: "user", ID: userID, Name: name, Meta: map[string]any{"email": "ada@example.com"}},
		RemoteIP:    remoteIP,
		UserAgent:   userAgent,
	})
	require.NoError(t, err)
	require.NoError(t, h.DB.InsertClickhouseOutbox(ctx, db.InsertClickhouseOutboxParams{
		Version:     auditlog.OutboxVersionV1,
		WorkspaceID: workspaceID,
		EventID:     uid.New(uid.AuditLogPrefix),
		Payload:     payload,
		CreatedAt:   now.UnixMilli(),
	}))

	actorMeta, err := json.Marshal(map[string]any{"email": "ada@example.com"})
	require.NoError(t, err)
	require.NoError(t, h.ClickHouse.InsertAuditLogs(ctx, []schema.AuditLogV1{{
		EventID:     uid.New(uid.AuditLogPrefix),
		Time:        now.UnixMilli(),
		WorkspaceID: workspaceID,
		Bucket:      "unkey_mutations",
		Source:      auditlog.EventSourcePlatform,
		Event:       "key.create",
		ActorType:   "user",
		ActorID:     userID,
		ActorName:   name,
		ActorMeta:   actorMeta,
		RemoteIP:    remoteIP,
		UserAgent:   userAgent,
		Meta:        json.RawMessage("{}"),
	}}))
}

func countUserRows(t *testing.T, h *harness.Harness, userID string) db.CountUserRowsByTableRow {
	t.Helper()
	counts, err := h.DB.CountUserRowsByTable(h.Ctx, db.CountUserRowsByTableParams{
		UserID:         userID,
		ApproverUserID: sql.NullString{String: userID, Valid: true},
	})
	require.NoError(t, err)
	return counts
}

// outboxActors decodes the actor of every queued audit event in the workspace
// that userID performed.
func outboxActors(t *testing.T, h *harness.Harness, workspaceID, userID string) []auditActor {
	t.Helper()
	rows, err := h.DB.ListClickhouseOutboxByWorkspace(h.Ctx, workspaceID)
	require.NoError(t, err)

	var actors []auditActor
	for _, row := range rows {
		var event auditlog.Event
		require.NoError(t, json.Unmarshal(row.Payload, &event))
		if event.Actor.ID != userID {
			continue
		}
		meta, err := json.Marshal(event.Actor.Meta)
		require.NoError(t, err)
		if event.Actor.Meta == nil {
			meta = []byte("{}")
		}
		actors = append(actors, auditActor{
			ID:        event.Actor.ID,
			Name:      event.Actor.Name,
			Meta:      string(meta),
			RemoteIP:  event.RemoteIP,
			UserAgent: event.UserAgent,
		})
	}
	return actors
}

// auditActors reads the actor of every exported audit row in the workspace
// that userID performed. audit_logs_raw_v1 has no reader function.
func auditActors(t *testing.T, h *harness.Harness, workspaceID, userID string) []auditActor {
	t.Helper()
	rows, err := h.ClickHouseConn.Query(h.Ctx, `
		SELECT actor_id, actor_name, toJSONString(actor_meta), remote_ip, user_agent
		FROM default.audit_logs_raw_v1 FINAL
		WHERE workspace_id = ? AND actor_id = ?`, workspaceID, userID)
	require.NoError(t, err)

	var actors []auditActor
	for rows.Next() {
		var actor auditActor
		require.NoError(t, rows.Scan(&actor.ID, &actor.Name, &actor.Meta, &actor.RemoteIP, &actor.UserAgent))
		actors = append(actors, actor)
	}
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
	return actors
}
