package cron_test

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// The cron handler keeps exported rows for 30 days before sweeping them, so
// the test seeds deleted_at offsets safely on either side of that boundary.
const cleanupRetention = 30 * 24 * time.Hour

// seededRow identifies a clickhouse_outbox row by its (unique) workspace and
// event id so the test can look it up via ListClickhouseOutboxByWorkspace.
type seededRow struct {
	workspaceID string
	eventID     string
}

func TestRunAuditLogOutboxCleanup_Integration(t *testing.T) {
	h := harness.New(t)

	now := time.Now()

	// stale: exported well before the retention cutoff -> must be deleted.
	stale := seedExportedRow(t, h, now.Add(-(cleanupRetention + 10*24*time.Hour)).UnixMilli())
	// recent: exported within the retention window -> must survive.
	recent := seedExportedRow(t, h, now.Add(-10*24*time.Hour).UnixMilli())
	// pending: never exported (deleted_at IS NULL) -> must survive.
	pending := seedPendingRow(t, h)

	resp, err := callRunAuditLogOutboxCleanup(h)
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.GetRowsDeleted(), "only the stale exported row is past the cutoff")

	require.False(t, outboxRowExists(t, h, stale), "stale exported row should be hard-deleted")
	require.True(t, outboxRowExists(t, h, recent), "recently-exported row should be kept for re-queue/audit")
	require.True(t, outboxRowExists(t, h, pending), "pending row (deleted_at NULL) must never be deleted")
}

// seedPendingRow inserts one un-exported (deleted_at NULL) outbox row under a
// fresh workspace.
func seedPendingRow(t *testing.T, h *harness.Harness) seededRow {
	t.Helper()
	row := seededRow{workspaceID: uid.New("ws"), eventID: uid.New("evt")}
	err := h.DB.InsertClickhouseOutbox(h.Ctx, db.InsertClickhouseOutboxParams{
		Version:     "audit_log.v1",
		WorkspaceID: row.workspaceID,
		EventID:     row.eventID,
		Payload:     json.RawMessage(`{}`),
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)
	return row
}

// seedExportedRow inserts a row and stamps its deleted_at to deletedAtMs,
// simulating a row the drainer already exported to ClickHouse.
func seedExportedRow(t *testing.T, h *harness.Harness, deletedAtMs int64) seededRow {
	t.Helper()
	row := seedPendingRow(t, h)

	rows, err := h.DB.ListClickhouseOutboxByWorkspace(h.Ctx, row.workspaceID)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	err = h.DB.MarkClickhouseOutboxBatchDeleted(h.Ctx, db.MarkClickhouseOutboxBatchDeletedParams{
		DeletedAt: sql.NullInt64{Int64: deletedAtMs, Valid: true},
		Pks:       []uint64{rows[0].Pk},
	})
	require.NoError(t, err)
	return row
}

func outboxRowExists(t *testing.T, h *harness.Harness, row seededRow) bool {
	t.Helper()
	rows, err := h.DB.ListClickhouseOutboxByWorkspace(h.Ctx, row.workspaceID)
	require.NoError(t, err)
	for _, r := range rows {
		if r.EventID == row.eventID {
			return true
		}
	}
	return false
}

func callRunAuditLogOutboxCleanup(h *harness.Harness) (*hydrav1.RunAuditLogOutboxCleanupResponse, error) {
	client := hydrav1.NewCronServiceIngressClient(h.Restate, "audit-log-outbox-cleanup")
	return client.RunAuditLogOutboxCleanup().Request(h.Ctx, &hydrav1.RunAuditLogOutboxCleanupRequest{})
}
