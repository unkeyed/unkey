package auditlogexport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type recordingClickHouse struct {
	clickhouse.ClickHouse
	err      error
	onInsert func(ctx context.Context)

	mu   sync.Mutex
	rows []schema.AuditLogV1
}

func (f *recordingClickHouse) InsertAuditLogs(ctx context.Context, rows []schema.AuditLogV1) error {
	f.mu.Lock()
	f.rows = append(f.rows, rows...)
	f.mu.Unlock()
	if f.onInsert != nil {
		f.onInsert(ctx)
	}
	return f.err
}

func (f *recordingClickHouse) eventIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := make([]string, 0, len(f.rows))
	for _, row := range f.rows {
		ids = append(ids, row.EventID)
	}
	return ids
}

// poolDatabase is a db.Database whose every query runs on a pool the test
// owns, so the test can read sql.DBStats and prove which connections the
// exporter holds at a given moment. db.New hides its *sql.DB.
type poolDatabase struct {
	*db.Queries
	rw *db.Replica
}

func (p *poolDatabase) Conn() *db.Replica     { return p.rw }
func (p *poolDatabase) RW() *db.Replica       { return p.rw }
func (p *poolDatabase) RO() *db.Replica       { return p.rw }
func (p *poolDatabase) Bulk() *db.BulkQueries { return db.NewBulkQueries(p.rw) }
func (p *poolDatabase) Close() error          { return p.rw.Close() }

func newPoolDatabase(t *testing.T, dsn string) (*poolDatabase, *sql.DB) {
	t.Helper()
	sqlDB, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	replica := mysql.NewReplicaFromDB(sqlDB, "rw", sqlcomment.Disabled())
	pool := &poolDatabase{Queries: db.NewQueries(replica), rw: replica}
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	return pool, sqlDB
}

func newDatabase(t *testing.T, dsn string) db.Database {
	t.Helper()
	database, err := db.New(dsn, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	return database
}

func drainOutbox(t *testing.T, ctx context.Context, database db.Database) {
	t.Helper()
	drainer := &Handler{db: database, clickhouse: clickhouse.NewNoop()}
	for {
		drained, err := drainer.exportBatch(ctx)
		require.NoError(t, err)
		if drained.EventsExported < batchLimit {
			return
		}
	}
}

func newOutboxEvent(event string) auditlog.Event {
	return auditlog.Event{
		EventID:     uid.New("evt"),
		Time:        time.Now().UnixMilli(),
		WorkspaceID: uid.New("ws"),
		Bucket:      "audit",
		Source:      auditlog.EventSourcePlatform,
		Event:       event,
		Description: "test export",
		Actor:       auditlog.EventActor{Type: "system", ID: "test"},
	}
}

func seedOutboxEvent(ctx context.Context, database db.Database, event auditlog.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return database.InsertClickhouseOutbox(ctx, db.InsertClickhouseOutboxParams{
		Version:     auditlog.OutboxVersionV1,
		WorkspaceID: event.WorkspaceID,
		EventID:     event.EventID,
		Payload:     payload,
		CreatedAt:   event.Time,
	})
}

func outboxDeletedAt(t *testing.T, ctx context.Context, database db.Database, event auditlog.Event) sql.NullInt64 {
	t.Helper()
	var deletedAt sql.NullInt64
	err := database.RW().QueryRowContext(ctx,
		"SELECT deleted_at FROM clickhouse_outbox WHERE workspace_id = ? AND event_id = ?",
		event.WorkspaceID,
		event.EventID,
	).Scan(&deletedAt)
	require.NoError(t, err)
	return deletedAt
}

// TestExportBatch_ClickHouseFailureLeavesOutboxRowPending protects at-least-once
// delivery by marking an outbox row exported only after ClickHouse acknowledges
// it. A failed insert must leave the row pending so a later attempt can retry it.
func TestExportBatch_ClickHouseFailureLeavesOutboxRowPending(t *testing.T) {
	ctx := context.Background()
	database := newDatabase(t, containers.MySQL(t).DSN)
	drainOutbox(t, ctx, database)

	event := newOutboxEvent("test.export")
	require.NoError(t, seedOutboxEvent(ctx, database, event))

	insertErr := errors.New("clickhouse unavailable")
	failingCH := &recordingClickHouse{ClickHouse: clickhouse.NewNoop(), err: insertErr}
	handler := &Handler{db: database, clickhouse: failingCH}

	result, err := handler.exportBatch(ctx)
	require.ErrorIs(t, err, insertErr)
	require.Zero(t, result.EventsExported)
	require.Contains(t, failingCH.eventIDs(), event.EventID, "the failing insert must include the seeded outbox row")
	require.False(t, outboxDeletedAt(t, ctx, database, event).Valid, "a ClickHouse failure must not mark the outbox row exported")

	handler.clickhouse = clickhouse.NewNoop()
	retryResult, err := handler.exportBatch(ctx)
	require.NoError(t, err)
	require.Positive(t, retryResult.EventsExported)
	require.True(t, outboxDeletedAt(t, ctx, database, event).Valid, "the event must be exported and marked for deletion")
}

// TestExportBatch_HoldsNoMySQLResourcesDuringClickHouseInsert is the regression
// for the production incident where the exporter kept a MySQL transaction with
// SELECT ... FOR UPDATE SKIP LOCKED open across the whole ClickHouse insert. An
// underfilled locking scan gap-locks drainer_pending_idx up to supremum under
// REPEATABLE READ, so every API audit INSERT queued behind the ClickHouse call.
func TestExportBatch_HoldsNoMySQLResourcesDuringClickHouseInsert(t *testing.T) {
	ctx := context.Background()
	dsn := containers.MySQL(t).DSN
	exporterDB, exporterPool := newPoolDatabase(t, dsn)
	writerDB := newDatabase(t, dsn)
	drainOutbox(t, ctx, exporterDB)

	selected := newOutboxEvent("test.export.selected")
	require.NoError(t, seedOutboxEvent(ctx, exporterDB, selected))

	entered := make(chan struct{})
	release := make(chan struct{})
	releaseCH := sync.OnceFunc(func() { close(release) })
	defer releaseCH()
	blockingCH := &recordingClickHouse{
		ClickHouse: clickhouse.NewNoop(),
		onInsert: func(ctx context.Context) {
			close(entered)
			select {
			case <-release:
			case <-ctx.Done():
			}
		},
	}
	handler := &Handler{db: exporterDB, clickhouse: blockingCH}

	type outcome struct {
		result batchResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, err := handler.exportBatch(ctx)
		done <- outcome{result: result, err: err}
	}()

	select {
	case <-entered:
	case <-time.After(30 * time.Second):
		t.Fatal("exporter never reached the ClickHouse insert")
	}

	require.Zero(t, exporterPool.Stats().InUse, "exporter must not hold a MySQL connection while ClickHouse is in flight")

	concurrent := newOutboxEvent("test.export.concurrent")
	insertCtx, cancelInsert := context.WithTimeout(ctx, 5*time.Second)
	defer cancelInsert()
	insertStart := time.Now()
	require.NoError(t, seedOutboxEvent(insertCtx, writerDB, concurrent), "concurrent outbox INSERT must not wait on the exporter")
	require.Less(t, time.Since(insertStart), 2*time.Second, "concurrent outbox INSERT must commit promptly")

	require.False(t, outboxDeletedAt(t, ctx, writerDB, selected).Valid, "nothing may be marked before ClickHouse acknowledges")

	releaseCH()
	var got outcome
	select {
	case got = <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("exporter did not finish after ClickHouse was released")
	}
	require.NoError(t, got.err)
	require.Positive(t, got.result.EventsExported)

	exported := blockingCH.eventIDs()
	require.Contains(t, exported, selected.EventID, "the selected row must reach ClickHouse")
	require.NotContains(t, exported, concurrent.EventID, "a row inserted mid-batch is not part of the selected set")
	require.True(t, outboxDeletedAt(t, ctx, writerDB, selected).Valid, "the selected row must be marked after ClickHouse acknowledges")
	require.False(t, outboxDeletedAt(t, ctx, writerDB, concurrent).Valid, "the mark must touch only the selected pks")
	require.Zero(t, exporterPool.Stats().InUse, "exporter must release every connection after the batch")
}

// TestExportBatch_MarkFailureAfterClickHouseAckKeepsRowReplayable models a
// Restate attempt whose context is cancelled (worker lost, invocation aborted)
// after ClickHouse acknowledged the batch but before the soft-delete ran. The
// row must stay pending so the replayed attempt exports and marks it; the
// second ClickHouse write is the documented at-least-once duplicate.
func TestExportBatch_MarkFailureAfterClickHouseAckKeepsRowReplayable(t *testing.T) {
	ctx := context.Background()
	database := newDatabase(t, containers.MySQL(t).DSN)
	drainOutbox(t, ctx, database)

	event := newOutboxEvent("test.export.replay")
	require.NoError(t, seedOutboxEvent(ctx, database, event))

	attemptCtx, cancelAttempt := context.WithCancel(ctx)
	defer cancelAttempt()
	ackThenCancelCH := &recordingClickHouse{
		ClickHouse: clickhouse.NewNoop(),
		onInsert:   func(context.Context) { cancelAttempt() },
	}
	handler := &Handler{db: database, clickhouse: ackThenCancelCH}

	result, err := handler.exportBatch(attemptCtx)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, result.EventsExported)
	require.Contains(t, ackThenCancelCH.eventIDs(), event.EventID, "ClickHouse acknowledged the row before the mark failed")
	require.False(t, outboxDeletedAt(t, ctx, database, event).Valid, "a failed mark must leave the row pending")

	replayCH := &recordingClickHouse{ClickHouse: clickhouse.NewNoop()}
	handler.clickhouse = replayCH
	replayResult, err := handler.exportBatch(ctx)
	require.NoError(t, err)
	require.Positive(t, replayResult.EventsExported)
	require.Contains(t, replayCH.eventIDs(), event.EventID, "the replay must export the row again")
	require.True(t, outboxDeletedAt(t, ctx, database, event).Valid, "the replay must mark the row once ClickHouse acknowledges")
}
