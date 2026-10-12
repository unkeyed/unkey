package lease

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/logdrain/internal/db"
	"google.golang.org/protobuf/proto"
)

// TestService_LeaseOwnership guarantees that lease IDs route polling, database
// time controls expiry, each acquisition has an independent fence, and a user
// pause prevents acquisition without allowing in-flight failures to override it.
func TestService_LeaseOwnership(t *testing.T) {
	ctx := context.Background()
	// Keep node time far from database time to prove that lease validity does
	// not depend on the node clock.
	nodeTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	nodeTimeMillis := nodeTime.UnixMilli()
	testClock := clock.NewTestClock(nodeTime)
	// Acquisition scans all workspaces, so unique drain IDs cannot isolate this test.
	mysqlConfig := containers.MySQLIsolated(t)
	database, err := db.New(mysqlConfig.DSN, sqlcomment.ForService("logdrain-lease-integration-test", "test"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	drainID := uid.New(uid.LogdrainPrefix, 16)
	workspaceID := uid.New(uid.WorkspacePrefix, 16)
	config, err := proto.Marshal(&logdrainv1.Config{
		Destination: &logdrainv1.Config_Http{Http: &logdrainv1.HttpConfig{
			Url:    "https://example.com/logs",
			Format: logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_JSON,
		}},
	})
	require.NoError(t, err)
	err = database.InsertLogdrain(ctx, db.InsertLogdrainParams{
		ID:                        drainID,
		WorkspaceID:               workspaceID,
		Name:                      "lease integration test",
		Stream:                    db.LogdrainsStreamAuditLogs,
		Config:                    config,
		Status:                    db.LogdrainsStatusPausedByUser,
		ConsecutiveFailures:       0,
		CommittedOffsetInsertedAt: 0,
		CommittedOffsetEventID:    "",
		NextAttemptAt:             0,
		LeaseID:                   "",
		FencingToken:              "",
		LeaseExpiresAt:            0,
		CreatedAt:                 nodeTimeMillis,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.DeleteLogdrain(context.Background(), drainID))
	})

	services := make([]*Service, 2)
	for i, leaseID := range []string{uid.New(""), uid.New("")} {
		services[i], err = New(Config{
			DB:      database,
			LeaseID: leaseID,
			Clock:   testClock,
		})
		require.NoError(t, err)
	}
	pausedAcquired, err := services[0].acquire(ctx)
	require.NoError(t, err)
	require.Zero(t, pausedAcquired, "a user-paused drain must not be leased")
	setStatus(t, ctx, database, drainID, db.LogdrainsStatusRunning)

	acquired := make([]int, len(services))
	acquireErrors := make([]error, len(services))
	acquireStartedAt := readDatabaseNowMillis(t, ctx, database)
	var acquisitions sync.WaitGroup
	acquisitions.Add(len(services))
	for i := range services {
		go func(index int) {
			defer acquisitions.Done()
			acquired[index], acquireErrors[index] = services[index].acquire(ctx)
		}(i)
	}
	acquisitions.Wait()
	for _, acquireErr := range acquireErrors {
		require.NoError(t, acquireErr)
	}
	acquireCompletedAt := readDatabaseNowMillis(t, ctx, database)
	require.Equal(t, 1, acquired[0]+acquired[1])

	acquiredDrain, err := database.FindLogdrainByID(ctx, drainID)
	require.NoError(t, err)
	leaseID, fencingToken, leaseExpiresAt := acquiredDrain.LeaseID, acquiredDrain.FencingToken, acquiredDrain.LeaseExpiresAt
	require.NotEmpty(t, fencingToken)
	require.GreaterOrEqual(t, leaseExpiresAt, acquireStartedAt+minimumTTL.Milliseconds())
	require.LessOrEqual(t, leaseExpiresAt, acquireCompletedAt+(minimumTTL+ttlJitter).Milliseconds())

	winner := services[0]
	if winner.leaseID != leaseID {
		winner = services[1]
	}
	loser := services[0]
	if loser == winner {
		loser = services[1]
	}
	dueDrains, err := database.ListDueLogdrains(ctx, leaseID)
	require.NoError(t, err)
	require.Equal(t, []db.ListDueLogdrainsRow{{
		LogdrainID:   drainID,
		FencingToken: fencingToken,
	}}, dueDrains)
	otherLeaseDrains, err := database.ListDueLogdrains(ctx, uid.New(""))
	require.NoError(t, err)
	require.Empty(t, otherLeaseDrains)
	leasedDrain, err := database.GetLeasedAndDueLogdrain(ctx, db.GetLeasedAndDueLogdrainParams{
		LogdrainID:   drainID,
		FencingToken: fencingToken,
	})
	require.NoError(t, err)
	require.Equal(t, drainID, leasedDrain.ID)
	setStatus(t, ctx, database, drainID, db.LogdrainsStatusPausedByUser)
	rowsAffected, err := database.RecordLogdrainFailure(ctx, db.RecordLogdrainFailureParams{
		Status:           db.LogdrainsStatusRunning,
		RetryAfterMillis: time.Minute.Milliseconds(),
		LogdrainID:       drainID,
		FencingToken:     fencingToken,
	})
	require.NoError(t, err)
	require.Zero(t, rowsAffected, "an in-flight failure must not override a user pause")
	pausedDrain, err := database.FindLogdrainByID(ctx, drainID)
	require.NoError(t, err)
	require.Equal(t, db.LogdrainsStatusPausedByUser, pausedDrain.Status)
	setStatus(t, ctx, database, drainID, db.LogdrainsStatusRunning)

	staleToken := "stale-fencing-token"
	rowsAffected, err = database.RecordLogdrainSuccess(ctx, db.RecordLogdrainSuccessParams{
		CommittedOffsetInsertedAt: 1,
		CommittedOffsetEventID:    "event_1",
		NextAttemptDelayMillis:    time.Minute.Milliseconds(),
		LogdrainID:                drainID,
		FencingToken:              staleToken,
	})
	require.NoError(t, err)
	require.Zero(t, rowsAffected, "a stale token must not advance the cursor")
	successDelay := time.Minute
	successStartedAt := readDatabaseNowMillis(t, ctx, database)
	rowsAffected, err = database.RecordLogdrainSuccess(ctx, db.RecordLogdrainSuccessParams{
		CommittedOffsetInsertedAt: 1,
		CommittedOffsetEventID:    "event_1",
		NextAttemptDelayMillis:    successDelay.Milliseconds(),
		LogdrainID:                drainID,
		FencingToken:              fencingToken,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)
	successCompletedAt := readDatabaseNowMillis(t, ctx, database)
	nextAttemptAt := readNextAttemptAt(t, ctx, database, drainID)
	require.GreaterOrEqual(t, nextAttemptAt, successStartedAt+successDelay.Milliseconds())
	require.LessOrEqual(t, nextAttemptAt, successCompletedAt+successDelay.Milliseconds())
	rowsAffected, err = database.RecordLogdrainFailure(ctx, db.RecordLogdrainFailureParams{
		Status:           db.LogdrainsStatusPausedByFailure,
		RetryAfterMillis: time.Minute.Milliseconds(),
		LogdrainID:       drainID,
		FencingToken:     staleToken,
	})
	require.NoError(t, err)
	require.Zero(t, rowsAffected, "a stale token must not record failure state")

	retryAfter := time.Minute
	failureStartedAt := readDatabaseNowMillis(t, ctx, database)
	rowsAffected, err = database.RecordLogdrainFailure(ctx, db.RecordLogdrainFailureParams{
		Status:           db.LogdrainsStatusRunning,
		RetryAfterMillis: retryAfter.Milliseconds(),
		LogdrainID:       drainID,
		FencingToken:     fencingToken,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)
	failureCompletedAt := readDatabaseNowMillis(t, ctx, database)
	nextAttemptAt = readNextAttemptAt(t, ctx, database, drainID)
	require.GreaterOrEqual(t, nextAttemptAt, failureStartedAt+retryAfter.Milliseconds())
	require.LessOrEqual(t, nextAttemptAt, failureCompletedAt+retryAfter.Milliseconds())
	require.NoError(t, database.UpdateLogdrainNextAttemptAt(ctx, db.UpdateLogdrainNextAttemptAtParams{
		NextAttemptAt:       0,
		ConsecutiveFailures: 0,
		ID:                  drainID,
	}))

	require.NoError(t, loser.refresh(ctx))
	require.Equal(t, leaseExpiresAt, readLeaseExpiry(t, ctx, database, drainID))

	existingExpiry := readDatabaseNowMillis(t, ctx, database) + time.Hour.Milliseconds()
	setLeaseExpiry(t, ctx, database, drainID, existingExpiry)
	refreshStartedAt := readDatabaseNowMillis(t, ctx, database)
	require.NoError(t, winner.refresh(ctx))
	refreshCompletedAt := readDatabaseNowMillis(t, ctx, database)
	refreshedExpiry := readLeaseExpiry(t, ctx, database, drainID)
	require.GreaterOrEqual(t, refreshedExpiry, refreshStartedAt+minimumTTL.Milliseconds())
	require.LessOrEqual(t, refreshedExpiry, refreshCompletedAt+(minimumTTL+ttlJitter).Milliseconds())
	require.Less(t, refreshedExpiry, existingExpiry, "refresh must replace rather than extend the expiry")

	expiredAt := readDatabaseNowMillis(t, ctx, database) - 1
	setLeaseExpiry(t, ctx, database, drainID, expiredAt)
	_, err = database.GetLeasedAndDueLogdrain(ctx, db.GetLeasedAndDueLogdrainParams{
		LogdrainID:   drainID,
		FencingToken: fencingToken,
	})
	require.ErrorIs(t, err, sql.ErrNoRows)
	rowsAffected, err = database.RecordLogdrainFailure(ctx, db.RecordLogdrainFailureParams{
		Status:           db.LogdrainsStatusRunning,
		RetryAfterMillis: time.Minute.Milliseconds(),
		LogdrainID:       drainID,
		FencingToken:     fencingToken,
	})
	require.NoError(t, err)
	require.Zero(t, rowsAffected, "an expired lease must not record failure state")
	require.NoError(t, winner.refresh(ctx))
	require.Equal(t, expiredAt, readLeaseExpiry(t, ctx, database, drainID))

	reacquired, err := winner.acquire(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, reacquired)
	sameProcessDrain, err := database.FindLogdrainByID(ctx, drainID)
	require.NoError(t, err)
	sameProcessToken := sameProcessDrain.FencingToken
	require.NotEqual(t, fencingToken, sameProcessToken)
	require.NoError(t, winner.refresh(ctx))

	expiredAt = readDatabaseNowMillis(t, ctx, database) - 1
	setLeaseExpiry(t, ctx, database, drainID, expiredAt)
	reacquirer, err := New(Config{
		DB:      database,
		LeaseID: uid.New(""),
		Clock:   testClock,
	})
	require.NoError(t, err)
	reacquired, err = reacquirer.acquire(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, reacquired)
	reacquiredDrain, err := database.FindLogdrainByID(ctx, drainID)
	require.NoError(t, err)
	require.Equal(t, reacquirer.leaseID, reacquiredDrain.LeaseID)
	require.NotEqual(t, sameProcessToken, reacquiredDrain.FencingToken)
	reacquiredExpiry := readLeaseExpiry(t, ctx, database, drainID)
	require.NoError(t, winner.refresh(ctx))
	require.Equal(t, reacquiredExpiry, readLeaseExpiry(t, ctx, database, drainID), "an old lease ID must not refresh a new owner's lease")
}

// readDatabaseNowMillis returns MySQL time with the same precision and
// representation used by lease queries.
func readDatabaseNowMillis(t *testing.T, ctx context.Context, database db.Database) int64 {
	t.Helper()
	var nowMillis int64
	err := database.Conn().QueryRowContext(ctx, "SELECT CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)").Scan(&nowMillis)
	require.NoError(t, err)
	return nowMillis
}

// readLeaseExpiry returns the stored absolute expiry for one test drain.
func readLeaseExpiry(t *testing.T, ctx context.Context, database db.Database, drainID string) int64 {
	t.Helper()
	drain, err := database.FindLogdrainByID(ctx, drainID)
	require.NoError(t, err)
	return drain.LeaseExpiresAt
}

// readNextAttemptAt returns the stored database-time retry timestamp.
func readNextAttemptAt(t *testing.T, ctx context.Context, database db.Database, drainID string) int64 {
	t.Helper()
	drain, err := database.FindLogdrainByID(ctx, drainID)
	require.NoError(t, err)
	return drain.NextAttemptAt
}

// setStatus changes only the status of one test drain.
func setStatus(t *testing.T, ctx context.Context, database db.Database, drainID string, status db.LogdrainsStatus) {
	t.Helper()
	require.NoError(t, database.UpdateLogdrainStatus(ctx, db.UpdateLogdrainStatusParams{Status: status, ID: drainID}))
}

// setLeaseExpiry stores an absolute lease expiry for one test drain.
func setLeaseExpiry(t *testing.T, ctx context.Context, database db.Database, drainID string, expiresAt int64) {
	t.Helper()
	require.NoError(t, database.UpdateLogdrainLeaseExpiry(ctx, db.UpdateLogdrainLeaseExpiryParams{LeaseExpiresAt: expiresAt, ID: drainID}))
}
