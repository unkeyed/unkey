package auditlogexport

import (
	"context"
	"encoding/json"
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

type incidentClickHouse struct {
	clickhouse.ClickHouse
	entered chan struct{}
}

func (c *incidentClickHouse) InsertAuditLogs(ctx context.Context, rows []schema.AuditLogV1) error {
	close(c.entered)
	return c.ClickHouse.InsertAuditLogs(ctx, rows)
}

func TestCancelledClickHouseAcquiresDoNotBlockOutboxWrites(t *testing.T) {
	ctx := context.Background()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	client, err := clickhouse.New(clickhouse.Config{URL: containers.ClickHouse(t).DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	for {
		drainer := &Handler{db: database, clickhouse: clickhouse.NewNoop()}
		result, err := drainer.exportBatch(ctx)
		require.NoError(t, err)
		if result.EventsExported < batchLimit {
			break
		}
	}
	seed := func(ctx context.Context) error {
		event := auditlog.Event{
			EventID: uid.New("evt"), WorkspaceID: uid.New("ws"), Time: time.Now().UnixMilli(),
			Bucket: "audit", Event: "test.incident", Source: auditlog.EventSourcePlatform,
			Actor: auditlog.EventActor{Type: "system", ID: "test"},
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}
		return database.InsertClickhouseOutbox(ctx, db.InsertClickhouseOutboxParams{
			Version: auditlog.OutboxVersionV1, WorkspaceID: event.WorkspaceID,
			EventID: event.EventID, Payload: payload, CreatedAt: event.Time,
		})
	}
	require.NoError(t, seed(ctx))
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	for range 300 {
		require.Error(t, client.Conn().Ping(cancelled))
	}
	stats := client.Conn().Stats()
	t.Logf("after 300 cancelled acquires: occupied=%d idle=%d limit=%d", stats.Open, stats.Idle, stats.MaxOpenConns)
	observed := &incidentClickHouse{ClickHouse: client, entered: make(chan struct{})}
	handler := &Handler{db: database, clickhouse: observed}
	exportCtx, cancelExport := context.WithTimeout(ctx, 8*time.Second)
	defer cancelExport()
	done := make(chan error, 1)
	exited := make(chan struct{})
	t.Cleanup(func() { cancelExport(); <-exited })
	start := time.Now()
	go func() {
		defer close(exited)
		_, err := handler.exportBatch(exportCtx)
		done <- err
	}()
	select {
	case <-observed.entered:
	case err := <-done:
		t.Fatalf("exporter failed before CH: %v", err)
	case <-exportCtx.Done():
		t.Fatal("exporter did not reach CH")
	}
	writerCtx, cancelWriter := context.WithTimeout(ctx, time.Second)
	writerStart := time.Now()
	writerErr := seed(writerCtx)
	writerElapsed := time.Since(writerStart)
	cancelWriter()
	exportErr := <-done
	t.Logf("concurrent outbox insert=%s error=%v; export=%s error=%v", writerElapsed, writerErr, time.Since(start), exportErr)
	require.Zero(t, stats.Open, "cancelled requests must not leak pool slots")
	require.NoError(t, writerErr, "ClickHouse must not block unrelated outbox inserts")
	require.NoError(t, exportErr, "healthy export must still succeed after cancelled requests")
}
