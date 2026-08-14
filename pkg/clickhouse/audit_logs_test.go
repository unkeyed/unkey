package clickhouse_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

const testBucket = "unkey_mutations"

// auditRow builds an AuditLogV1 with the given identifiers and timestamps. All
// other fields carry deterministic, non-empty values so reads can be asserted.
func auditRow(workspaceID, eventID string, timeMs, insertedAtMs int64) schema.AuditLogV1 {
	return schema.AuditLogV1{
		EventID:       eventID,
		Time:          timeMs,
		InsertedAt:    insertedAtMs,
		WorkspaceID:   workspaceID,
		Bucket:        testBucket,
		Source:        "platform",
		Event:         "key.create",
		Description:   "created a key",
		ActorType:     "root_key",
		ActorID:       "root_" + eventID,
		ActorName:     "root key",
		ActorMeta:     json.RawMessage(`{"role":"admin"}`),
		RemoteIP:      "1.2.3.4",
		UserAgent:     "unkey-test/1.0",
		Meta:          json.RawMessage(`{"foo":"bar"}`),
		TargetTypes:   []string{"key"},
		TargetIDs:     []string{"key_" + eventID},
		TargetNames:   []string{"a key"},
		TargetMetas:   []json.RawMessage{json.RawMessage(`{"k":"v"}`)},
		CorrelationID: "corr_" + eventID,
	}
}

func newAuditTestClient(t *testing.T) (*clickhouse.Client, ch.Conn, string) {
	t.Helper()
	chCfg := containers.ClickHouse(t)

	client, err := clickhouse.New(clickhouse.Config{URL: chCfg.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	opts, err := ch.ParseDSN(chCfg.DSN)
	require.NoError(t, err)
	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return client, conn, uid.New(uid.WorkspacePrefix)
}

// seed inserts rows and waits until they are all visible in the raw table.
func seed(t *testing.T, client *clickhouse.Client, conn ch.Conn, workspaceID string, rows []schema.AuditLogV1) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, client.InsertAuditLogs(ctx, rows))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var count uint64
		err := conn.QueryRow(ctx, "SELECT count() FROM default.audit_logs_raw_v1 WHERE workspace_id = ?", workspaceID).Scan(&count)
		require.NoError(c, err)
		require.Equal(c, uint64(len(rows)), count)
	}, time.Minute, time.Second)
}

func eventIDs(rows []schema.AuditLogV1) []string {
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.EventID
	}
	return ids
}

func TestListAuditLogs(t *testing.T) {
	t.Parallel()
	client, conn, workspaceID := newAuditTestClient(t)
	ctx := context.Background()

	base := time.Now().Add(-time.Hour).UnixMilli()
	// Three rows at distinct inserted_at, plus two sharing an inserted_at to
	// exercise the (inserted_at, event_id) tiebreak.
	rows := []schema.AuditLogV1{
		auditRow(workspaceID, "evt_a", base+10, base+100),
		auditRow(workspaceID, "evt_b", base+20, base+200),
		auditRow(workspaceID, "evt_c", base+30, base+200), // shares inserted_at with evt_b
		auditRow(workspaceID, "evt_d", base+40, base+300),
	}
	seed(t, client, conn, workspaceID, rows)

	window := clickhouse.ListAuditLogsRequest{
		WorkspaceID: workspaceID,
		Bucket:      testBucket,
		StartMs:     base,
		EndMs:       base + 100000,
		Limit:       100,
	}

	t.Run("orders ascending by (inserted_at, event_id)", func(t *testing.T) {
		got, err := client.ListAuditLogs(ctx, window)
		require.NoError(t, err)
		require.Equal(t, []string{"evt_a", "evt_b", "evt_c", "evt_d"}, eventIDs(got))
	})

	t.Run("keyset boundary skips no rows sharing an inserted_at", func(t *testing.T) {
		req := window
		req.Limit = 2
		page1, err := client.ListAuditLogs(ctx, req)
		require.NoError(t, err)
		require.Equal(t, []string{"evt_a", "evt_b"}, eventIDs(page1))

		last := page1[len(page1)-1]
		req.AfterInsertedAtMs = last.InsertedAt
		req.AfterEventID = last.EventID
		page2, err := client.ListAuditLogs(ctx, req)
		require.NoError(t, err)
		// evt_c shares evt_b's inserted_at; it must appear, not be skipped.
		require.Equal(t, []string{"evt_c", "evt_d"}, eventIDs(page2))
	})

	t.Run("limit+1 fetch reveals more", func(t *testing.T) {
		req := window
		req.Limit = 3 // 3 requested; 4 exist, so limit+1 pattern would return 4
		got, err := client.ListAuditLogs(ctx, req)
		require.NoError(t, err)
		require.Len(t, got, 3)
	})

	t.Run("preserves JSON meta round-trip", func(t *testing.T) {
		got, err := client.ListAuditLogs(ctx, window)
		require.NoError(t, err)
		require.NotEmpty(t, got)
		first := got[0]

		var actorMeta, meta, targetMeta map[string]string
		require.NoError(t, json.Unmarshal(first.ActorMeta, &actorMeta))
		require.Equal(t, "admin", actorMeta["role"])
		require.NoError(t, json.Unmarshal(first.Meta, &meta))
		require.Equal(t, "bar", meta["foo"])
		require.Len(t, first.TargetMetas, 1)
		require.NoError(t, json.Unmarshal(first.TargetMetas[0], &targetMeta))
		require.Equal(t, "v", targetMeta["k"])

		require.Equal(t, []string{"key"}, first.TargetTypes)
		require.Equal(t, "1.2.3.4", first.RemoteIP)
	})

	t.Run("empty range returns empty slice", func(t *testing.T) {
		req := window
		req.StartMs = base + 500000
		req.EndMs = base + 600000
		got, err := client.ListAuditLogs(ctx, req)
		require.NoError(t, err)
		require.Empty(t, got)
	})
}

func TestListAuditLogs_Filters(t *testing.T) {
	t.Parallel()
	client, conn, workspaceID := newAuditTestClient(t)
	ctx := context.Background()

	base := time.Now().Add(-time.Hour).UnixMilli()
	r1 := auditRow(workspaceID, "evt_1", base+10, base+100)
	r1.Event = "key.create"
	r1.ActorID = "root_alpha"
	r2 := auditRow(workspaceID, "evt_2", base+20, base+200)
	r2.Event = "api.delete"
	r2.ActorID = "root_beta"
	r2.TargetTypes = []string{"api"}
	seed(t, client, conn, workspaceID, []schema.AuditLogV1{r1, r2})

	window := clickhouse.ListAuditLogsRequest{
		WorkspaceID: workspaceID, Bucket: testBucket, StartMs: base, EndMs: base + 100000, Limit: 100,
	}

	t.Run("event filter", func(t *testing.T) {
		req := window
		req.Events = []string{"api.delete"}
		got, err := client.ListAuditLogs(ctx, req)
		require.NoError(t, err)
		require.Equal(t, []string{"evt_2"}, eventIDs(got))
	})

	t.Run("actorId filter", func(t *testing.T) {
		req := window
		req.ActorID = "root_alpha"
		got, err := client.ListAuditLogs(ctx, req)
		require.NoError(t, err)
		require.Equal(t, []string{"evt_1"}, eventIDs(got))
	})

	t.Run("resourceType filter", func(t *testing.T) {
		req := window
		req.ResourceType = "api"
		got, err := client.ListAuditLogs(ctx, req)
		require.NoError(t, err)
		require.Equal(t, []string{"evt_2"}, eventIDs(got))
	})
}

func TestListAuditLogs_WorkspaceIsolation(t *testing.T) {
	t.Parallel()
	client, conn, workspaceID := newAuditTestClient(t)
	ctx := context.Background()
	other := uid.New(uid.WorkspacePrefix)

	base := time.Now().Add(-time.Hour).UnixMilli()
	mine := auditRow(workspaceID, "evt_mine", base+10, base+100)
	theirs := auditRow(other, "evt_theirs", base+20, base+200)
	// Insert both; wait for each workspace's rows to be visible.
	require.NoError(t, client.InsertAuditLogs(ctx, []schema.AuditLogV1{mine, theirs}))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var count uint64
		err := conn.QueryRow(ctx, "SELECT count() FROM default.audit_logs_raw_v1 WHERE workspace_id IN (?, ?)", workspaceID, other).Scan(&count)
		require.NoError(c, err)
		require.Equal(c, uint64(2), count)
	}, time.Minute, time.Second)

	got, err := client.ListAuditLogs(ctx, clickhouse.ListAuditLogsRequest{
		WorkspaceID: workspaceID, Bucket: testBucket, StartMs: base, EndMs: base + 100000, Limit: 100,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"evt_mine"}, eventIDs(got))
}

// TestListAuditLogs_LateDrainNoGap is the load-bearing regression for the
// gap-free-sync guarantee: a row that drains late (older event time, but a
// newer inserted_at than the consumer's current watermark) MUST still be
// returned on the next page. Keying the cursor on event time instead of
// inserted_at would silently drop it.
func TestListAuditLogs_LateDrainNoGap(t *testing.T) {
	t.Parallel()
	client, conn, workspaceID := newAuditTestClient(t)
	ctx := context.Background()

	base := time.Now().Add(-time.Hour).UnixMilli()
	// A consumer has already paged through this row and holds its watermark.
	early := auditRow(workspaceID, "evt_early", base+50, base+100)
	seed(t, client, conn, workspaceID, []schema.AuditLogV1{early})

	window := clickhouse.ListAuditLogsRequest{
		WorkspaceID: workspaceID, Bucket: testBucket, StartMs: base, EndMs: base + 100000, Limit: 100,
	}
	page1, err := client.ListAuditLogs(ctx, window)
	require.NoError(t, err)
	require.Equal(t, []string{"evt_early"}, eventIDs(page1))
	watermark := page1[len(page1)-1]

	// Now a late-committing event arrives: its event time is OLDER than the
	// watermark's event time, but its inserted_at is NEWER (it drained after).
	late := auditRow(workspaceID, "evt_late", base+10, base+200)
	require.Less(t, late.Time, watermark.Time, "late event has older event time")
	require.Greater(t, late.InsertedAt, watermark.InsertedAt, "late event has newer inserted_at")
	seed(t, client, conn, workspaceID, []schema.AuditLogV1{early, late})

	req := window
	req.AfterInsertedAtMs = watermark.InsertedAt
	req.AfterEventID = watermark.EventID
	page2, err := client.ListAuditLogs(ctx, req)
	require.NoError(t, err)
	require.Equal(t, []string{"evt_late"}, eventIDs(page2), "late-drained row must not be skipped")
}
