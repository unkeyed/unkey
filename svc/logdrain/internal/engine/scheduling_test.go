package engine_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/logdrain/internal/db"
	"github.com/unkeyed/unkey/svc/logdrain/internal/engine"
	"github.com/unkeyed/unkey/svc/logdrain/internal/source"
	drainsink "github.com/unkeyed/unkey/svc/logdrain/sink"
	"google.golang.org/protobuf/proto"
)

func TestEngine_NonAuditSourceFailureKeepsRetryAndCursor(t *testing.T) {
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	serviceClock := clock.NewTestClock(now)
	encoded, err := proto.Marshal(&logdrainv1.Config{Stream: &logdrainv1.Config_RuntimeLogs{RuntimeLogs: &logdrainv1.RuntimeLogStreamConfig{}}})
	require.NoError(t, err)
	database := &scheduleDatabase{
		drain: db.GetLeasedAndDueLogdrainRow{ID: "drain", Config: encoded, Stream: db.LogdrainsStreamAuditLogs, CommittedOffsetInsertedAt: now.Add(-time.Minute).UnixMilli()},
		clock: serviceClock, committed: make(chan db.RecordLogdrainSuccessParams, 1), failures: make(chan db.RecordLogdrainFailureParams, 1),
	}
	eng, err := engine.New(engine.Config{DB: database, LeaseID: "lease", Clock: serviceClock, RuntimeLogs: &scheduleSource{clock: serviceClock, failure: errors.New("source unavailable")}, PollInterval: time.Minute, WatermarkLag: 5 * time.Minute, BatchSize: 100, PauseThreshold: 50})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- eng.Run(ctx) }()
	t.Cleanup(func() { cancel(); require.NoError(t, <-done) })
	select {
	case failure := <-database.failures:
		require.Equal(t, time.Minute.Milliseconds(), failure.RetryAfterMillis)
		require.Equal(t, db.LogdrainsStatusRunning, failure.Status)
		require.Empty(t, database.committed)
	case <-time.After(5 * time.Second):
		t.Fatal("source failure was not recorded")
	}
}

func TestEngine_StreamWatermarkAndCatchup(t *testing.T) {
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name   string
		config *logdrainv1.Config
		lag    time.Duration
		delay  time.Duration
	}{
		{"legacy audit", &logdrainv1.Config{}, 5 * time.Minute, time.Minute},
		{"audit", &logdrainv1.Config{Stream: &logdrainv1.Config_AuditLogs{AuditLogs: &logdrainv1.AuditLogStreamConfig{}}}, 5 * time.Minute, time.Minute},
		{"verification", &logdrainv1.Config{Stream: &logdrainv1.Config_KeyVerifications{KeyVerifications: &logdrainv1.KeyVerificationStreamConfig{}}}, 15 * time.Second, 5 * time.Second},
		{"gateway", &logdrainv1.Config{Stream: &logdrainv1.Config_GatewayRequests{GatewayRequests: &logdrainv1.GatewayRequestStreamConfig{}}}, 15 * time.Second, 5 * time.Second},
		{"runtime", &logdrainv1.Config{Stream: &logdrainv1.Config_RuntimeLogs{RuntimeLogs: &logdrainv1.RuntimeLogStreamConfig{}}}, 15 * time.Second, 5 * time.Second},
		{"ratelimit", &logdrainv1.Config{Stream: &logdrainv1.Config_Ratelimits{Ratelimits: &logdrainv1.RatelimitStreamConfig{}}}, 15 * time.Second, 5 * time.Second},
	} {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := proto.Marshal(test.config)
			require.NoError(t, err)
			serviceClock := clock.NewTestClock(now)
			tickerClock := &scheduleClock{Clock: serviceClock, intervals: make(chan time.Duration, 1)}
			database := &scheduleDatabase{
				drain:     db.GetLeasedAndDueLogdrainRow{ID: "drain", Config: encoded, Stream: db.LogdrainsStreamRuntimeLogs, CommittedOffsetInsertedAt: now.Add(-10 * time.Minute).UnixMilli()},
				committed: make(chan db.RecordLogdrainSuccessParams, 32),
				failures:  make(chan db.RecordLogdrainFailureParams, 1),
				clock:     serviceClock,
			}
			reader := &scheduleSource{clock: serviceClock}
			eng, err := engine.New(engine.Config{DB: database, LeaseID: "lease", Clock: tickerClock, AuditLogs: reader, KeyVerifications: reader, GatewayRequests: reader, RuntimeLogs: reader, Ratelimits: reader, PollInterval: time.Minute, WatermarkLag: 5 * time.Minute, BatchSize: 100})
			require.NoError(t, err)
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			go func() { done <- eng.Run(ctx) }()
			t.Cleanup(func() { cancel(); require.NoError(t, <-done) })
			select {
			case interval := <-tickerClock.intervals:
				require.Equal(t, 5*time.Second, interval)
			case <-time.After(5 * time.Second):
				t.Fatal("discovery ticker did not start")
			}
			commits := 0
			for {
				select {
				case commit := <-database.committed:
					commits++
					if commit.NextAttemptDelayMillis == 0 {
						continue
					}
					require.Greater(t, commits, 1)
					require.Equal(t, test.delay.Milliseconds(), commit.NextAttemptDelayMillis)
					require.Equal(t, now.Add(-test.lag).UnixMilli(), commit.CommittedOffsetInsertedAt)
					assertDueSchedule(t, commit, now.Add(37*time.Second), test.delay)
					return
				case <-time.After(5 * time.Second):
					t.Fatal("catch-up did not complete without another poll")
				}
			}
		})
	}
}

func assertDueSchedule(t *testing.T, commit db.RecordLogdrainSuccessParams, committedAt time.Time, delay time.Duration) {
	t.Helper()
	mysqlConfig := containers.MySQL(t)
	pool, err := sql.Open("mysql", mysqlConfig.DSN)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	conn, err := pool.Conn(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	workspaceID, drainID := uniqueIDs()
	seedDrain(t, pool, workspaceID, drainID, "https://example.com", commit.CommittedOffsetInsertedAt-1)
	cleanupDrain(t, pool, drainID)
	_, err = conn.ExecContext(t.Context(), "SET timestamp = ?", committedAt.Unix())
	require.NoError(t, err)
	queries := db.NewQueries(conn)
	rows, err := queries.AcquireLogdrainLease(t.Context(), db.AcquireLogdrainLeaseParams{LogdrainID: drainID, LeaseID: drainID, FencingToken: "fence", TtlMillis: time.Hour.Milliseconds()})
	require.NoError(t, err)
	require.EqualValues(t, 1, rows)
	commit.LogdrainID = drainID
	commit.FencingToken = "fence"
	rows, err = queries.RecordLogdrainSuccess(t.Context(), commit)
	require.NoError(t, err)
	require.EqualValues(t, 1, rows)
	var nextAttempt int64
	require.NoError(t, conn.QueryRowContext(t.Context(), "SELECT next_attempt_at FROM logdrains WHERE id = ?", drainID).Scan(&nextAttempt))
	require.Equal(t, committedAt.Add(delay).UnixMilli(), nextAttempt)
	for elapsed := time.Duration(0); elapsed < delay; elapsed += 5 * time.Second {
		_, err = conn.ExecContext(t.Context(), "SET timestamp = ?", committedAt.Add(elapsed).Unix())
		require.NoError(t, err)
		due, err := queries.ListDueLogdrains(t.Context(), drainID)
		require.NoError(t, err)
		require.Empty(t, due, "caught-up drain must not be queued at %s", elapsed)
		_, err = queries.GetLeasedAndDueLogdrain(t.Context(), db.GetLeasedAndDueLogdrainParams{LogdrainID: drainID, FencingToken: "fence"})
		require.ErrorIs(t, err, sql.ErrNoRows)
	}
	_, err = conn.ExecContext(t.Context(), "SET timestamp = ?", committedAt.Add(delay).Unix())
	require.NoError(t, err)
	due, err := queries.ListDueLogdrains(t.Context(), drainID)
	require.NoError(t, err)
	require.Len(t, due, 1)
}

type scheduleSource struct {
	clock   *clock.TestClock
	failure error
}

type scheduleClock struct {
	clock.Clock
	intervals chan time.Duration
}

func (c *scheduleClock) NewTicker(interval time.Duration) clock.Ticker {
	ticker := c.Clock.NewTicker(interval)
	c.intervals <- interval
	return ticker
}

func (s *scheduleSource) Read(_ context.Context, _ string, from source.Cursor, _ int64, _ int, _ *logdrainv1.Config) ([]drainsink.Event, source.Cursor, error) {
	s.clock.Tick(time.Millisecond)
	return nil, from, s.failure
}

type scheduleDatabase struct {
	db.Database
	mu        sync.Mutex
	drain     db.GetLeasedAndDueLogdrainRow
	committed chan db.RecordLogdrainSuccessParams
	failures  chan db.RecordLogdrainFailureParams
	caughtUp  bool
	clock     *clock.TestClock
	started   bool
}

func (d *scheduleDatabase) ListDueLogdrains(context.Context, string) ([]db.ListDueLogdrainsRow, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.caughtUp {
		return nil, nil
	}
	return []db.ListDueLogdrainsRow{{LogdrainID: d.drain.ID}}, nil
}

func (d *scheduleDatabase) CountLogdrainsByStatus(context.Context) ([]db.CountLogdrainsByStatusRow, error) {
	return nil, nil
}

func (d *scheduleDatabase) GetLeasedAndDueLogdrain(context.Context, db.GetLeasedAndDueLogdrainParams) (db.GetLeasedAndDueLogdrainRow, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.caughtUp {
		return db.GetLeasedAndDueLogdrainRow{}, sql.ErrNoRows
	}
	if !d.started {
		d.clock.Tick(2 * time.Second)
		d.started = true
	}
	return d.drain, nil
}

func (d *scheduleDatabase) RecordLogdrainSuccess(_ context.Context, p db.RecordLogdrainSuccessParams) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.drain.CommittedOffsetInsertedAt = p.CommittedOffsetInsertedAt
	d.drain.CommittedOffsetEventID = p.CommittedOffsetEventID
	d.caughtUp = p.NextAttemptDelayMillis > 0
	d.committed <- p
	return 1, nil
}

func (d *scheduleDatabase) RecordLogdrainFailure(_ context.Context, p db.RecordLogdrainFailureParams) (int64, error) {
	d.failures <- p
	return 1, nil
}
