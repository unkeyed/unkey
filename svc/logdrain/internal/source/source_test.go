package source_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/logdrain/internal/source"
)

// TestAuditLogsRead_CursorBounds preserves timestamp ties across pages without
// including the cursor, the exclusive upper bound, or another workspace's rows.
func TestAuditLogsRead_CursorBounds(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	workspaceID := uid.New("workspace")
	otherWorkspaceID := uid.New("workspace")
	insertedAt := time.Now().Add(-time.Minute).UnixMilli()

	t.Cleanup(func() {
		require.NoError(t, client.Conn().Exec(ctx, `
			ALTER TABLE audit_logs_raw_v1 DELETE
			WHERE workspace_id IN (?, ?) SETTINGS mutations_sync = 1
		`, workspaceID, otherWorkspaceID))
	})
	for _, event := range []struct {
		workspaceID string
		id          string
		insertedAt  int64
	}{
		{workspaceID, "z", insertedAt - 1},
		{workspaceID, "a", insertedAt},
		{workspaceID, "b", insertedAt},
		{workspaceID, "c", insertedAt},
		{workspaceID, "d", insertedAt},
		{workspaceID, "a", insertedAt + 1},
		{workspaceID, "e", insertedAt + 2},
		{otherWorkspaceID, "z", insertedAt},
	} {
		require.NoError(t, client.Conn().Exec(ctx, `
			INSERT INTO audit_logs_raw_v1 (workspace_id, bucket, event_id, time, inserted_at)
			VALUES (?, 'audit', ?, ?, ?)
		`, event.workspaceID, event.id, event.insertedAt, event.insertedAt))
	}

	auditLogs := source.NewAuditLogs(client)
	cursor := source.Cursor{Time: insertedAt, EventID: "b"}
	toExclusive := insertedAt + 2
	firstPage, cursor, err := auditLogs.Read(ctx, workspaceID, cursor, toExclusive, 2, nil)
	require.NoError(t, err)
	require.Len(t, firstPage, 2)
	require.Equal(t, "c", firstPage[0].EventID)
	require.Equal(t, "d", firstPage[1].EventID)
	require.Equal(t, source.Cursor{Time: insertedAt, EventID: "d"}, cursor)

	secondPage, cursor, err := auditLogs.Read(ctx, workspaceID, cursor, toExclusive, 2, nil)
	require.NoError(t, err)
	require.Len(t, secondPage, 1)
	require.Equal(t, "a", secondPage[0].EventID)
	require.Equal(t, source.Cursor{Time: insertedAt + 1, EventID: "a"}, cursor)

	emptyPage, finalCursor, err := auditLogs.Read(ctx, workspaceID, cursor, toExclusive, 2, nil)
	require.NoError(t, err)
	require.Empty(t, emptyPage)
	require.Equal(t, cursor, finalCursor)
}

// TestAuditLogsRead_EventTypes filters before pagination without losing timestamp ties.
func TestAuditLogsRead_EventTypes(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := clickhouse.New(clickhouse.Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()
	workspaceID := uid.New("workspace")
	insertedAt := time.Now().Add(-time.Minute).UnixMilli()
	t.Cleanup(func() {
		require.NoError(t, client.Conn().Exec(ctx, `ALTER TABLE audit_logs_raw_v1 DELETE WHERE workspace_id = ? SETTINGS mutations_sync = 1`, workspaceID))
	})
	for _, event := range []struct{ id, eventType string }{
		{"a", "key.verify"},
		{"b", "key.create"},
		{"c", "key.verify"},
		{"d", "key.delete"},
		{"e", "key.create"},
		{"f", "key.verify"},
		{"g", `custom.'\event`},
	} {
		require.NoError(t, client.Conn().Exec(ctx, `
			INSERT INTO audit_logs_raw_v1 (workspace_id, bucket, event_id, time, inserted_at, event)
			VALUES (?, 'audit', ?, ?, ?, ?)
		`, workspaceID, event.id, insertedAt, insertedAt, event.eventType))
	}
	auditLogs := source.NewAuditLogs(client)
	from := source.Cursor{Time: insertedAt}
	filter := &logdrainv1.Config{Stream: &logdrainv1.Config_AuditLogs{AuditLogs: &logdrainv1.AuditLogStreamConfig{EventTypes: []string{"key.create", "key.delete"}}}}
	page, cursor, err := auditLogs.Read(ctx, workspaceID, from, insertedAt+1, 2, filter)
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.Equal(t, "b", page[0].EventID)
	require.Equal(t, "d", page[1].EventID)
	page, cursor, err = auditLogs.Read(ctx, workspaceID, cursor, insertedAt+1, 2, filter)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "e", page[0].EventID)
	require.Equal(t, source.Cursor{Time: insertedAt, EventID: "e"}, cursor)
	page, finalCursor, err := auditLogs.Read(ctx, workspaceID, cursor, insertedAt+1, 2, filter)
	require.NoError(t, err)
	require.Empty(t, page)
	require.Equal(t, cursor, finalCursor)

	filter.GetAuditLogs().EventTypes = []string{`custom.'\event`}
	page, _, err = auditLogs.Read(ctx, workspaceID, from, insertedAt+1, 2, filter)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, "g", page[0].EventID)
}
