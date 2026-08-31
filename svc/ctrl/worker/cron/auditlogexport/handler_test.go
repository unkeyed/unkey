package auditlogexport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type failingClickHouse struct {
	clickhouse.ClickHouse
	err  error
	rows []schema.AuditLogV1
}

func (f *failingClickHouse) InsertAuditLogs(_ context.Context, rows []schema.AuditLogV1) error {
	f.rows = append(f.rows, rows...)
	return f.err
}

// TestExportBatch_ClickHouseFailureLeavesOutboxRowPending protects at-least-once
// delivery by marking an outbox row exported only after ClickHouse acknowledges
// it. A failed insert must leave the row pending so a later attempt can retry it.
func TestExportBatch_ClickHouseFailureLeavesOutboxRowPending(t *testing.T) {
	ctx := context.Background()
	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	// A reused container can carry pending outbox rows from earlier runs.
	// The export batch reads the whole table (ORDER BY pk LIMIT batchLimit),
	// so enough leftovers would evict this test's freshly-seeded row from the
	// window and break the Contains assertion below. Drain to a clean slate
	// first with a noop ClickHouse; this only marks already-pending rows
	// deleted, so it can't race other processes that share the container.
	drainer := &Handler{db: database, clickhouse: clickhouse.NewNoop()}
	for {
		drained, drainErr := drainer.exportBatch(ctx)
		require.NoError(t, drainErr)
		if drained.EventsExported < batchLimit {
			break
		}
	}

	event := auditlog.Event{
		EventID:     uid.New("evt"),
		Time:        time.Now().UnixMilli(),
		WorkspaceID: uid.New("ws"),
		Bucket:      "audit",
		Source:      auditlog.EventSourcePlatform,
		Event:       "test.export",
		Description: "test export",
		Actor:       auditlog.EventActor{Type: "system", ID: "test"},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)
	require.NoError(t, database.InsertClickhouseOutbox(ctx, db.InsertClickhouseOutboxParams{
		Version:     auditlog.OutboxVersionV1,
		WorkspaceID: event.WorkspaceID,
		EventID:     event.EventID,
		Payload:     payload,
		CreatedAt:   event.Time,
	}))

	insertErr := errors.New("clickhouse unavailable")
	failingCH := &failingClickHouse{
		ClickHouse: clickhouse.NewNoop(),
		err:        insertErr,
	}
	handler := &Handler{db: database, clickhouse: failingCH}

	result, err := handler.exportBatch(ctx)
	require.ErrorIs(t, err, insertErr)
	require.Zero(t, result.EventsExported)

	exportedEventIDs := make([]string, 0, len(failingCH.rows))
	for _, row := range failingCH.rows {
		exportedEventIDs = append(exportedEventIDs, row.EventID)
	}
	require.Contains(t, exportedEventIDs, event.EventID, "the failing insert must include the seeded outbox row")

	var deletedAt sql.NullInt64
	err = database.RW().QueryRowContext(ctx,
		"SELECT deleted_at FROM clickhouse_outbox WHERE workspace_id = ? AND event_id = ?",
		event.WorkspaceID,
		event.EventID,
	).Scan(&deletedAt)
	require.NoError(t, err)
	require.False(t, deletedAt.Valid, "a ClickHouse failure must not mark the outbox row exported")

	handler.clickhouse = clickhouse.NewNoop()
	retryResult, err := handler.exportBatch(ctx)
	require.NoError(t, err)
	require.Positive(t, retryResult.EventsExported)

	err = database.RW().QueryRowContext(ctx,
		"SELECT deleted_at FROM clickhouse_outbox WHERE workspace_id = ? AND event_id = ?",
		event.WorkspaceID,
		event.EventID,
	).Scan(&deletedAt)
	require.NoError(t, err)
	require.True(t, deletedAt.Valid, "the event must be exported and marked for deletion")
}
