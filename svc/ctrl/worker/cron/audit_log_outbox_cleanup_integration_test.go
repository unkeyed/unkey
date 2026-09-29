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

	// The same sweep covers the back office audit outbox, which stamps
	// drained_at instead of deleted_at.
	boStale := seedBackofficeRow(t, h, sql.NullInt64{Int64: now.Add(-(cleanupRetention + 10*24*time.Hour)).UnixMilli(), Valid: true})
	boRecent := seedBackofficeRow(t, h, sql.NullInt64{Int64: now.Add(-10 * 24 * time.Hour).UnixMilli(), Valid: true})
	boPending := seedBackofficeRow(t, h, sql.NullInt64{})

	resp, err := callRunAuditLogOutboxCleanup(h)
	require.NoError(t, err)
	require.Equal(t, int64(2), resp.GetRowsDeleted(), "one stale row per outbox table is past the cutoff")

	require.False(t, outboxRowExists(t, h, stale), "stale exported row should be hard-deleted")
	require.True(t, outboxRowExists(t, h, recent), "recently-exported row should be kept for re-queue/audit")
	require.True(t, outboxRowExists(t, h, pending), "pending row (deleted_at NULL) must never be deleted")

	require.False(t, backofficeRowExists(t, h, boStale), "stale drained back office row should be hard-deleted")
	require.True(t, backofficeRowExists(t, h, boRecent), "recently-drained back office row should be kept")
	require.True(t, backofficeRowExists(t, h, boPending), "pending back office row (drained_at NULL) must never be deleted")
}

// seedBackofficeRow inserts one backoffice_audit_outbox row. A NULL
// drainedAt is a row the back office app has not copied to ClickHouse yet.
// The control plane has no insert query for this table (only the back
// office app writes it), so the test seeds it directly.
func seedBackofficeRow(t *testing.T, h *harness.Harness, drainedAt sql.NullInt64) string {
	t.Helper()
	eventID := uid.New("boal")
	_, err := h.DB.RW().ExecContext(h.Ctx,
		"INSERT INTO backoffice_audit_outbox (event_id, payload, created_at, drained_at) VALUES (?, ?, ?, ?)",
		eventID, "{}", time.Now().UnixMilli(), drainedAt,
	)
	require.NoError(t, err)
	return eventID
}

func backofficeRowExists(t *testing.T, h *harness.Harness, eventID string) bool {
	t.Helper()
	var n int
	err := h.DB.RW().QueryRowContext(h.Ctx,
		"SELECT COUNT(*) FROM backoffice_audit_outbox WHERE event_id = ?", eventID,
	).Scan(&n)
	require.NoError(t, err)
	return n > 0
}

// seedPendingRow inserts one un-exported (deleted_at NULL) outbox row under a
// fresh workspace.
func seedPendingRow(t *testing.T, h *harness.Harness) seededRow {
	t.Helper()
	row := seededRow{workspaceID: uid.New(uid.WorkspacePrefix), eventID: uid.New(uid.AuditLogPrefix)}
	payload, err := json.Marshal(struct{}{})
	require.NoError(t, err)
	err = h.DB.InsertClickhouseOutbox(h.Ctx, db.InsertClickhouseOutboxParams{
		Version:     "audit_log.v1",
		WorkspaceID: row.workspaceID,
		EventID:     row.eventID,
		Payload:     payload,
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
