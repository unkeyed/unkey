package auditlogexport

import (
	"database/sql"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
)

func TestUnkeyPlatformBucketID(t *testing.T) {
	require.Equal(t, "unkey_audit_ws_abc", unkeyPlatformBucketID("ws_abc"))
	require.Equal(t, "unkey_audit_", unkeyPlatformBucketID(""))
}

func TestBuildCHRows(t *testing.T) {
	// Single retention so the math is checkable by eye: 30 days in ms.
	const retentionMs = int64(30 * 24 * 60 * 60 * 1000)
	retention := map[string]int64{"ws_a": retentionMs}

	t.Run("event with multiple targets fans out to one row per target", func(t *testing.T) {
		t.Helper()
		events := []db.FindUnexportedAuditLogsRow{
			{
				Pk:          1,
				ID:          "log_1",
				WorkspaceID: "ws_a",
				Bucket:      "unkey_mutations",
				Event:       "key.create",
				Time:        1_000,
				Display:     "Created key foo",
				ActorType:   "user",
				ActorID:     "user_1",
				ActorName:   sql.NullString{String: "Alice", Valid: true},
				ActorMeta:   []byte(`{"role":"admin"}`),
				RemoteIp:    sql.NullString{String: "1.2.3.4", Valid: true},
				UserAgent:   sql.NullString{String: "curl", Valid: true},
			},
		}
		targets := map[string][]db.FindAuditLogTargetsForLogsRow{
			"log_1": {
				{AuditLogID: "log_1", Type: "key", ID: "key_1", Name: sql.NullString{String: "foo", Valid: true}, Meta: []byte(`{"k":"v"}`)},
				{AuditLogID: "log_1", Type: "api", ID: "api_1", Name: sql.NullString{String: "myapi", Valid: true}, Meta: nil},
			},
		}

		rows := buildCHRows(events, targets, retention)
		require.Len(t, rows, 2, "two targets should produce two CH rows")

		// Both rows share the envelope fields.
		require.Equal(t, rows[0].EventID, rows[1].EventID)
		require.Equal(t, "log_1", rows[0].EventID)
		require.Equal(t, "", rows[0].WorkspaceID, "platform events have empty workspace_id")
		require.Equal(t, "unkey_audit_ws_a", rows[0].BucketID)
		require.Equal(t, "key.create", rows[0].Event)
		require.Equal(t, "Created key foo", rows[0].Description)
		require.Equal(t, "Alice", rows[0].ActorName)
		require.Equal(t, "1.2.3.4", rows[0].RemoteIP)

		// Targets differ.
		var keys []string
		for _, r := range rows {
			keys = append(keys, r.TargetType+":"+r.TargetID)
		}
		sort.Strings(keys)
		require.Equal(t, []string{"api:api_1", "key:key_1"}, keys)
	})

	t.Run("event with no targets emits one row with empty target fields", func(t *testing.T) {
		t.Helper()
		events := []db.FindUnexportedAuditLogsRow{
			{Pk: 2, ID: "log_2", WorkspaceID: "ws_a", Bucket: "b", Event: "ping", Time: 2_000, Display: "d", ActorType: "system", ActorID: "sys"},
		}
		rows := buildCHRows(events, nil, retention)
		require.Len(t, rows, 1)
		require.Equal(t, "", rows[0].TargetType)
		require.Equal(t, "", rows[0].TargetID)
	})

	t.Run("expires_at is event_time + retention", func(t *testing.T) {
		t.Helper()
		events := []db.FindUnexportedAuditLogsRow{
			{Pk: 3, ID: "log_3", WorkspaceID: "ws_a", Bucket: "b", Event: "x", Time: 1_700_000_000_000, Display: "d", ActorType: "u", ActorID: "u1"},
		}
		rows := buildCHRows(events, nil, retention)
		expected := time.UnixMilli(1_700_000_000_000 + retentionMs)
		require.True(t, rows[0].ExpiresAt.Equal(expected),
			"expires_at should anchor to event time, got %v want %v", rows[0].ExpiresAt, expected)
	})

	t.Run("nullable string columns map empty when invalid", func(t *testing.T) {
		t.Helper()
		events := []db.FindUnexportedAuditLogsRow{
			{
				Pk: 4, ID: "log_4", WorkspaceID: "ws_a", Bucket: "b", Event: "x", Time: 4_000, Display: "d", ActorType: "u", ActorID: "u1",
				ActorName: sql.NullString{Valid: false},
				RemoteIp:  sql.NullString{Valid: false},
				UserAgent: sql.NullString{Valid: false},
			},
		}
		rows := buildCHRows(events, nil, retention)
		require.Equal(t, "", rows[0].ActorName)
		require.Equal(t, "", rows[0].RemoteIP)
		require.Equal(t, "", rows[0].UserAgent)
	})

	t.Run("missing workspace retention falls back to zero (caller responsibility)", func(t *testing.T) {
		t.Helper()
		// If a workspace has no entry in the retention map, expires_at lands at
		// event time itself. This is intentional: the worker fails the batch
		// before this if it can't load retention, so an unmapped workspace
		// here means a programmer error and we'd rather emit visibly-bad
		// expires_at than silently apply a default.
		events := []db.FindUnexportedAuditLogsRow{
			{Pk: 5, ID: "log_5", WorkspaceID: "ws_unknown", Bucket: "b", Event: "x", Time: 5_000, Display: "d", ActorType: "u", ActorID: "u1"},
		}
		rows := buildCHRows(events, nil, retention)
		require.True(t, rows[0].ExpiresAt.Equal(time.UnixMilli(5_000)))
	})
}
