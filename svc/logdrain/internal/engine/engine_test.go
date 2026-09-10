package engine

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/logdrain/internal/db"
	"github.com/unkeyed/unkey/svc/logdrain/internal/source"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
	"google.golang.org/protobuf/proto"
)

// TestProcess_CatchesUpEmptyWindows requires one process call to drain the
// backlog in bounded reads, delaying the next attempt only at the watermark.
func TestProcess_CatchesUpEmptyWindows(t *testing.T) {
	ctx := context.Background()
	start := time.Now().Add(-time.Hour).UnixMilli()
	watermark := start + (130 * time.Minute).Milliseconds()
	database := &windowDatabase{drain: db.GetLeasedAndDueLogdrainRow{
		ID: "drain", WorkspaceID: "workspace", Stream: db.LogdrainsStreamAuditLogs,
		CommittedOffsetInsertedAt: start,
	}}
	var ends []time.Duration
	reader := windowSource{read: func(_ context.Context, _ string, from source.Cursor, to int64, _ int, _ *logdrainv1.Config) ([]sink.Event, source.Cursor, error) {
		require.Positive(t, to-from.Time)
		require.LessOrEqual(t, to-from.Time, time.Hour.Milliseconds())
		ends = append(ends, time.Duration(to-start)*time.Millisecond)
		return nil, from, nil
	}}
	eng, err := New(Config{DB: database, LeaseID: "lease", AuditLogs: reader, PollInterval: time.Hour, WatermarkLag: 5 * time.Minute, BatchSize: 100})
	require.NoError(t, err)
	eng.process(ctx, workItem{id: "drain", now: time.UnixMilli(watermark).Add(5 * time.Minute)})
	require.Equal(t, []time.Duration{
		time.Minute, 3 * time.Minute, 7 * time.Minute, 15 * time.Minute,
		31 * time.Minute, 63 * time.Minute, 123 * time.Minute, 130 * time.Minute,
	}, ends)
	require.Len(t, database.commits, 8)
	for _, commit := range database.commits[:7] {
		require.Zero(t, commit.NextAttemptDelayMillis)
	}
	require.Equal(t, time.Hour.Milliseconds(), database.commits[7].NextAttemptDelayMillis)
	require.Equal(t, watermark, database.drain.CommittedOffsetInsertedAt)
	require.Empty(t, database.drain.CommittedOffsetEventID)
}

// TestProcess_DeliveryFailure stops catch-up without committing a partial
// window's proposed cursor when its events were not acknowledged.
func TestProcess_DeliveryFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	encoded, err := proto.Marshal(&logdrainv1.Config{Destination: &logdrainv1.Config_Http{Http: &logdrainv1.HttpConfig{
		Url: server.URL, Format: logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_JSON,
	}}})
	require.NoError(t, err)
	start := time.Now().Add(-time.Hour).UnixMilli()
	minute := time.Minute.Milliseconds()
	database := &windowDatabase{drain: db.GetLeasedAndDueLogdrainRow{
		ID: "drain", WorkspaceID: "workspace", Stream: db.LogdrainsStreamAuditLogs,
		CommittedOffsetInsertedAt: start, Config: encoded,
	}}
	reads := 0
	reader := windowSource{read: func(_ context.Context, _ string, from source.Cursor, to int64, limit int, _ *logdrainv1.Config) ([]sink.Event, source.Cursor, error) {
		reads++
		require.Equal(t, 1, reads)
		require.Equal(t, source.Cursor{Time: start}, from)
		require.Equal(t, start+minute, to)
		require.Equal(t, 2, limit)
		return []sink.Event{{EventID: "event", Stream: "audit_logs", Time: start, Payload: sink.AuditLogPayload{ID: "event"}}}, source.Cursor{Time: start, EventID: "event"}, nil
	}}
	eng, err := New(Config{DB: database, LeaseID: "lease", AuditLogs: reader, PollInterval: time.Hour, BatchSize: 2, PauseThreshold: 5, UnsafeAllowPrivateEndpoints: true})
	require.NoError(t, err)
	eng.process(context.Background(), workItem{id: "drain", now: time.UnixMilli(start + 6*minute)})
	require.Equal(t, 1, reads)
	require.Len(t, database.failures, 1)
	require.Empty(t, database.commits)
	require.Equal(t, start, database.drain.CommittedOffsetInsertedAt)
}

// TestProcess_LogsCommittedBatch captures process output without changing the global logger.
func TestProcess_LogsCommittedBatch(t *testing.T) {
	if os.Getenv("TEST_LOGDRAIN_BATCH_LOG") != "1" {
		binary, err := os.Executable()
		require.NoError(t, err)
		cmd := exec.Command(binary, "-test.run=^TestProcess_LogsCommittedBatch$")
		cmd.Env = append(os.Environ(), "TEST_LOGDRAIN_BATCH_LOG=1", "NO_COLOR=1", "UNKEY_LOG_LEVEL=info")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, string(output))
		require.Contains(t, string(output), `msg="logdrain batch delivered"`)
		require.Contains(t, string(output), "drain_id=drain stream=audit_logs events=1")
		require.Contains(t, string(output), "cursor_time=2026-09-09T12:01:00.000Z")
		require.Contains(t, string(output), "lag_ms=7140000")
		return
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	encoded, err := proto.Marshal(&logdrainv1.Config{Destination: &logdrainv1.Config_Http{Http: &logdrainv1.HttpConfig{
		Url: server.URL, Format: logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_JSON,
	}}})
	require.NoError(t, err)
	start := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	database := &windowDatabase{drain: db.GetLeasedAndDueLogdrainRow{
		ID: "drain", WorkspaceID: "workspace", Stream: db.LogdrainsStreamAuditLogs,
		CommittedOffsetInsertedAt: start.UnixMilli(), Config: encoded,
	}}
	reader := windowSource{read: func(_ context.Context, _ string, from source.Cursor, _ int64, _ int, _ *logdrainv1.Config) ([]sink.Event, source.Cursor, error) {
		return []sink.Event{{EventID: "event", Stream: "audit_logs", Time: start.UnixMilli(), Payload: sink.AuditLogPayload{ID: "event"}}}, from, nil
	}}
	eng, err := New(Config{
		DB: database, LeaseID: "lease", AuditLogs: reader, PollInterval: time.Minute, BatchSize: 2,
		Clock: clock.NewTestClock(start.Add(2 * time.Hour)), UnsafeAllowPrivateEndpoints: true,
	})
	require.NoError(t, err)
	eng.process(context.Background(), workItem{id: "drain", now: start.Add(time.Minute)})
	require.Len(t, database.commits, 1)
}

// TestProcess_ZeroCursor advances through empty history using ordinary windows.
func TestProcess_ZeroCursor(t *testing.T) {
	const watermark int64 = 180000
	database := &windowDatabase{drain: db.GetLeasedAndDueLogdrainRow{
		ID: "drain", WorkspaceID: "workspace", Stream: db.LogdrainsStreamAuditLogs,
	}}
	reader := windowSource{
		read: func(_ context.Context, _ string, from source.Cursor, _ int64, _ int, _ *logdrainv1.Config) ([]sink.Event, source.Cursor, error) {
			return nil, from, nil
		},
	}
	eng, err := New(Config{DB: database, LeaseID: "lease", AuditLogs: reader, PollInterval: time.Hour, BatchSize: 100})
	require.NoError(t, err)
	eng.process(context.Background(), workItem{id: "drain", now: time.UnixMilli(watermark)})
	require.Empty(t, database.failures)
	require.Len(t, database.commits, 2)
	require.Equal(t, int64(60000), database.commits[0].CommittedOffsetInsertedAt)
	require.Zero(t, database.commits[0].NextAttemptDelayMillis)
	require.Equal(t, watermark, database.drain.CommittedOffsetInsertedAt)
}

// TestProcess_EventTypes passes persisted filters through every window and advances empty ones.
func TestProcess_EventTypes(t *testing.T) {
	for _, eventTypes := range [][]string{nil, {"key.create", "key.delete"}} {
		t.Run(fmt.Sprint(eventTypes), func(t *testing.T) {
			encoded, err := proto.Marshal(&logdrainv1.Config{Stream: &logdrainv1.Config_AuditLogs{
				AuditLogs: &logdrainv1.AuditLogStreamConfig{EventTypes: eventTypes},
			}})
			require.NoError(t, err)
			database := &windowDatabase{drain: db.GetLeasedAndDueLogdrainRow{
				ID: "drain", WorkspaceID: "workspace", Stream: db.LogdrainsStreamAuditLogs,
				CommittedOffsetInsertedAt: 1000000, Config: encoded,
			}}
			reader := windowSource{read: func(_ context.Context, _ string, from source.Cursor, _ int64, _ int, filter *logdrainv1.Config) ([]sink.Event, source.Cursor, error) {
				require.Equal(t, eventTypes, filter.GetAuditLogs().GetEventTypes())
				return nil, from, nil
			}}
			eng, err := New(Config{DB: database, LeaseID: "lease", AuditLogs: reader, PollInterval: time.Minute, BatchSize: 2})
			require.NoError(t, err)
			eng.process(context.Background(), workItem{id: "drain", now: time.UnixMilli(1180000)})
			require.Len(t, database.commits, 2)
			require.Equal(t, int64(1180000), database.drain.CommittedOffsetInsertedAt)
		})
	}
}

func TestProcess_KeyVerificationOutcomes(t *testing.T) {
	encoded, err := proto.Marshal(&logdrainv1.Config{Stream: &logdrainv1.Config_KeyVerifications{
		KeyVerifications: &logdrainv1.KeyVerificationStreamConfig{Outcomes: []string{"RATE_LIMITED"}, KeySpaceIds: []string{"ks_1", "ks_2"}},
	}})
	require.NoError(t, err)
	database := &windowDatabase{drain: db.GetLeasedAndDueLogdrainRow{
		ID: "drain", WorkspaceID: "workspace", Stream: db.LogdrainsStreamAuditLogs,
		CommittedOffsetInsertedAt: 1000000, Config: encoded,
	}}
	reader := windowSource{read: func(_ context.Context, _ string, from source.Cursor, _ int64, _ int, filter *logdrainv1.Config) ([]sink.Event, source.Cursor, error) {
		require.Equal(t, []string{"RATE_LIMITED"}, filter.GetKeyVerifications().GetOutcomes())
		require.Equal(t, []string{"ks_1", "ks_2"}, filter.GetKeyVerifications().GetKeySpaceIds())
		return nil, from, nil
	}}
	eng, err := New(Config{DB: database, LeaseID: "lease", KeyVerifications: reader, PollInterval: time.Minute, BatchSize: 2})
	require.NoError(t, err)
	eng.process(t.Context(), workItem{id: "drain", now: time.UnixMilli(1180000)})
	require.Empty(t, database.failures)
	require.Len(t, database.commits, 2)
	require.Equal(t, int64(1180000), database.drain.CommittedOffsetInsertedAt)
}

// windowDatabase models cursor persistence and due-time gating between reads.
type windowDatabase struct {
	db.Database
	drain    db.GetLeasedAndDueLogdrainRow
	commits  []db.RecordLogdrainSuccessParams
	failures []db.RecordLogdrainFailureParams
}

// GetLeasedAndDueLogdrain respects the delay recorded by the previous batch.
func (d *windowDatabase) GetLeasedAndDueLogdrain(context.Context, db.GetLeasedAndDueLogdrainParams) (db.GetLeasedAndDueLogdrainRow, error) {
	if len(d.commits) > 0 && d.commits[len(d.commits)-1].NextAttemptDelayMillis > 0 {
		return db.GetLeasedAndDueLogdrainRow{}, sql.ErrNoRows
	}
	return d.drain, nil
}

// RecordLogdrainSuccess makes committed progress visible to the next read.
func (d *windowDatabase) RecordLogdrainSuccess(_ context.Context, params db.RecordLogdrainSuccessParams) (int64, error) {
	d.commits = append(d.commits, params)
	d.drain.CommittedOffsetInsertedAt = params.CommittedOffsetInsertedAt
	d.drain.CommittedOffsetEventID = params.CommittedOffsetEventID
	return 1, nil
}

// RecordLogdrainFailure records retries without changing the cursor.
func (d *windowDatabase) RecordLogdrainFailure(_ context.Context, params db.RecordLogdrainFailureParams) (int64, error) {
	d.failures = append(d.failures, params)
	return 1, nil
}

// TestBackoff guarantees the retry schedule and the total wait before the
// default failure threshold.
func TestBackoff(t *testing.T) {
	want := []time.Duration{
		1 * time.Minute,
		2 * time.Minute,
		4 * time.Minute,
		8 * time.Minute,
		16 * time.Minute,
		32 * time.Minute,
		64 * time.Minute,
		128 * time.Minute,
		4 * time.Hour,
		4 * time.Hour,
	}
	for failures, delay := range want {
		require.Equal(t, delay, backoff(failures))
	}

	var total time.Duration
	for failures := range 49 {
		total += backoff(failures)
	}
	require.Equal(t, 7*24*time.Hour+15*time.Minute, total)
}

// TestRetryDelay guarantees destination hints can extend local backoff but
// cannot exceed the one-day cap.
func TestRetryDelay(t *testing.T) {
	require.Equal(t, time.Minute, retryDelay(0, 30*time.Second))
	require.Equal(t, 2*time.Hour, retryDelay(0, 2*time.Hour))
	require.Equal(t, 24*time.Hour, retryDelay(0, 7*24*time.Hour))
	require.Equal(t, 4*time.Hour, retryDelay(8, 0))
}
