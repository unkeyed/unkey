package auditlogexport

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
)

func TestBuildCHRows(t *testing.T) {
	// Single retention so the math is checkable by eye: 30 days in ms.
	const retentionMs = int64(30 * 24 * 60 * 60 * 1000)
	retention := map[string]int64{"ws_a": retentionMs}

	t.Run("event with multiple targets folds into a single CH row with parallel arrays", func(t *testing.T) {
		t.Helper()
		events := []auditlog.Event{
			{
				EventID:     "log_1",
				Time:        1_000,
				WorkspaceID: "ws_a",
				Bucket:      "unkey_mutations",
				Source:      auditlog.EventSourcePlatform,
				Event:       "key.create",
				Description: "Created key foo",
				Actor: auditlog.EventActor{
					Type: "user",
					ID:   "user_1",
					Name: "Alice",
					Meta: map[string]any{"role": "admin"},
				},
				RemoteIP:  "1.2.3.4",
				UserAgent: "curl",
				Targets: []auditlog.EventTarget{
					{Type: "key", ID: "key_1", Name: "foo", Meta: map[string]any{"k": "v"}},
					{Type: "api", ID: "api_1", Name: "myapi"},
				},
			},
		}

		rows, err := buildCHRows(events, retention)
		require.NoError(t, err)
		require.Len(t, rows, 1, "one event = one CH row regardless of target count")

		r := rows[0]
		require.Equal(t, "log_1", r.EventID)
		require.Equal(t, "ws_a", r.WorkspaceID, "workspace_id is the real owner workspace, no platform encoding hack")
		require.Equal(t, "unkey_mutations", r.Bucket)
		require.Equal(t, "platform", r.Source)
		require.Equal(t, "key.create", r.Event)
		require.Equal(t, "Created key foo", r.Description)
		require.Equal(t, "Alice", r.ActorName)
		require.JSONEq(t, `{"role":"admin"}`, string(r.ActorMeta))
		require.Equal(t, "1.2.3.4", r.RemoteIP)

		// Targets stored as parallel arrays — same length, index-aligned.
		require.Equal(t, []string{"key", "api"}, r.TargetTypes)
		require.Equal(t, []string{"key_1", "api_1"}, r.TargetIDs)
		require.Equal(t, []string{"foo", "myapi"}, r.TargetNames)
		require.Len(t, r.TargetMetas, 2)
		require.JSONEq(t, `{"k":"v"}`, string(r.TargetMetas[0]))
		require.JSONEq(t, `{}`, string(r.TargetMetas[1]), "missing meta serializes to empty object")
	})

	t.Run("event with no targets emits one row with empty target arrays", func(t *testing.T) {
		t.Helper()
		events := []auditlog.Event{
			{
				EventID:     "log_2",
				Time:        2_000,
				WorkspaceID: "ws_a",
				Bucket:      "b",
				Source:      auditlog.EventSourcePlatform,
				Event:       "ping",
				Description: "d",
				Actor:       auditlog.EventActor{Type: "system", ID: "sys"},
			},
		}
		rows, err := buildCHRows(events, retention)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Empty(t, rows[0].TargetTypes)
		require.Empty(t, rows[0].TargetIDs)
		require.Empty(t, rows[0].TargetNames)
		require.Empty(t, rows[0].TargetMetas)
	})

	t.Run("expires_at is event_time + retention (unix-milli)", func(t *testing.T) {
		t.Helper()
		events := []auditlog.Event{
			{
				EventID:     "log_3",
				Time:        1_700_000_000_000,
				WorkspaceID: "ws_a",
				Bucket:      "b",
				Event:       "x",
				Description: "d",
				Actor:       auditlog.EventActor{Type: "u", ID: "u1"},
			},
		}
		rows, err := buildCHRows(events, retention)
		require.NoError(t, err)
		require.Equal(t, int64(1_700_000_000_000)+retentionMs, rows[0].ExpiresAt)
	})

	t.Run("blank source defaults to platform", func(t *testing.T) {
		t.Helper()
		events := []auditlog.Event{
			{EventID: "log_4", Time: 4_000, WorkspaceID: "ws_a", Bucket: "b", Event: "x", Description: "d", Actor: auditlog.EventActor{Type: "u", ID: "u1"}},
		}
		rows, err := buildCHRows(events, retention)
		require.NoError(t, err)
		require.Equal(t, auditlog.EventSourcePlatform, rows[0].Source)
	})

	t.Run("missing workspace retention falls back to zero (caller responsibility)", func(t *testing.T) {
		t.Helper()
		// If a workspace has no entry in the retention map, expires_at lands at
		// event time itself. This is intentional: the worker fails the batch
		// before this if it can't load retention, so an unmapped workspace
		// here means a programmer error and we'd rather emit visibly-bad
		// expires_at than silently apply a default.
		events := []auditlog.Event{
			{EventID: "log_5", Time: 5_000, WorkspaceID: "ws_unknown", Bucket: "b", Event: "x", Description: "d", Actor: auditlog.EventActor{Type: "u", ID: "u1"}},
		}
		rows, err := buildCHRows(events, retention)
		require.NoError(t, err)
		require.Equal(t, int64(5_000), rows[0].ExpiresAt)
	})

	t.Run("inserted_at is the build time (unix-milli), not the event time", func(t *testing.T) {
		t.Helper()
		events := []auditlog.Event{
			{EventID: "log_6", Time: 1_000, WorkspaceID: "ws_a", Bucket: "b", Event: "x", Description: "d", Actor: auditlog.EventActor{Type: "u", ID: "u1"}},
		}
		before := time.Now().UnixMilli()
		rows, err := buildCHRows(events, retention)
		after := time.Now().UnixMilli()
		require.NoError(t, err)
		require.GreaterOrEqual(t, rows[0].InsertedAt, before)
		require.LessOrEqual(t, rows[0].InsertedAt, after)
	})

	t.Run("empty meta becomes JSON empty-object {}", func(t *testing.T) {
		t.Helper()
		events := []auditlog.Event{
			{EventID: "log_7", Time: 1_000, WorkspaceID: "ws_a", Bucket: "b", Event: "x", Description: "d", Actor: auditlog.EventActor{Type: "u", ID: "u1"}},
		}
		rows, err := buildCHRows(events, retention)
		require.NoError(t, err)
		require.JSONEq(t, `{}`, string(rows[0].ActorMeta))
		require.JSONEq(t, `{}`, string(rows[0].Meta))
	})
}
