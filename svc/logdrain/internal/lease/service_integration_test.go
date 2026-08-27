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
// time controls expiry, and each acquisition has an independent fence.
func TestService_LeaseOwnership(t *testing.T) {
	ctx := context.Background()
	// Keep node time far from database time to prove that lease validity does
	// not depend on the node clock.
	nodeTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	nodeTimeMillis := nodeTime.UnixMilli()
	testClock := clock.NewTestClock(nodeTime)
	mysqlConfig := containers.MySQL(t)
	database, err := db.New(mysqlConfig.DSN, sqlcomment.ForService("logdrain-lease-integration-test", "test"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	drainID := uid.New("ld", 16)
	workspaceID := uid.New("ws", 16)
	config, err := proto.Marshal(&logdrainv1.Config{
		Destination: &logdrainv1.Config_Http{Http: &logdrainv1.HttpConfig{
			Url:    "https://example.com/logs",
			Format: logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_JSON,
		}},
	})
	require.NoError(t, err)
	_, err = database.Conn().ExecContext(ctx, `
		INSERT INTO logdrains (id, workspace_id, name, stream, config, enabled, created_at)
		VALUES (?, ?, 'lease integration test', 'audit_logs', ?, true, ?)
	`, drainID, workspaceID, config, nodeTimeMillis)
	require.NoError(t, err)
	_, err = database.Conn().ExecContext(ctx, `
		INSERT INTO logdrain_state (logdrain_id, lease_id, fencing_token, lease_expires_at)
		VALUES (?, '', '', 0)
	`, drainID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := database.Conn().ExecContext(context.Background(), "DELETE FROM logdrain_state WHERE logdrain_id = ?", drainID)
		require.NoError(t, cleanupErr)
		_, cleanupErr = database.Conn().ExecContext(context.Background(), "DELETE FROM logdrains WHERE id = ?", drainID)
		require.NoError(t, cleanupErr)
	})

	services := make([]*Service, 2)
	for i, leaseID := range []string{uid.New(""), uid.New("")} {
		services[i], err = New(Config{DB: database, LeaseID: leaseID, Clock: testClock})
		require.NoError(t, err)
	}

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

	var leaseID, fencingToken string
	var leaseExpiresAt int64
	err = database.Conn().QueryRowContext(ctx, `
		SELECT lease_id, fencing_token, lease_expires_at
		FROM logdrain_state WHERE logdrain_id = ?
	`, drainID).Scan(&leaseID, &fencingToken, &leaseExpiresAt)
	require.NoError(t, err)
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
	require.Equal(t, []db.ListDueLogdrainsRow{{LogdrainID: drainID, FencingToken: fencingToken}}, dueDrains)
	otherLeaseDrains, err := database.ListDueLogdrains(ctx, uid.New(""))
	require.NoError(t, err)
	require.Empty(t, otherLeaseDrains)
	leasedDrain, err := database.GetLeasedLogdrain(ctx, db.GetLeasedLogdrainParams{
		LogdrainID:   drainID,
		FencingToken: fencingToken,
	})
	require.NoError(t, err)
	require.Equal(t, drainID, leasedDrain.ID)

	staleToken := "stale-fencing-token"
	rowsAffected, err := database.RecordLogdrainSuccess(ctx, db.RecordLogdrainSuccessParams{
		CommittedOffsetInsertedAt: 1,
		CommittedOffsetEventID:    "event_1",
		LogdrainID:                drainID,
		FencingToken:              staleToken,
	})
	require.NoError(t, err)
	require.Zero(t, rowsAffected, "a stale token must not advance the cursor")
	rowsAffected, err = database.RecordLogdrainFailure(ctx, db.RecordLogdrainFailureParams{
		Status:           db.LogdrainStateStatusPausedByFailure,
		RetryAfterMillis: time.Minute.Milliseconds(),
		LogdrainID:       drainID,
		FencingToken:     staleToken,
	})
	require.NoError(t, err)
	require.Zero(t, rowsAffected, "a stale token must not record failure state")

	retryAfter := time.Minute
	failureStartedAt := readDatabaseNowMillis(t, ctx, database)
	rowsAffected, err = database.RecordLogdrainFailure(ctx, db.RecordLogdrainFailureParams{
		Status:           db.LogdrainStateStatusActive,
		RetryAfterMillis: retryAfter.Milliseconds(),
		LogdrainID:       drainID,
		FencingToken:     fencingToken,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)
	failureCompletedAt := readDatabaseNowMillis(t, ctx, database)
	nextAttemptAt := readNextAttemptAt(t, ctx, database, drainID)
	require.GreaterOrEqual(t, nextAttemptAt, failureStartedAt+retryAfter.Milliseconds())
	require.LessOrEqual(t, nextAttemptAt, failureCompletedAt+retryAfter.Milliseconds())
	_, err = database.Conn().ExecContext(ctx, "UPDATE logdrain_state SET consecutive_failures = 0, next_attempt_at = 0 WHERE logdrain_id = ?", drainID)
	require.NoError(t, err)

	require.NoError(t, loser.refresh(ctx))
	require.Equal(t, leaseExpiresAt, readLeaseExpiry(t, ctx, database, drainID))

	existingExpiry := readDatabaseNowMillis(t, ctx, database) + time.Hour.Milliseconds()
	_, err = database.Conn().ExecContext(ctx, "UPDATE logdrain_state SET lease_expires_at = ? WHERE logdrain_id = ?", existingExpiry, drainID)
	require.NoError(t, err)
	refreshStartedAt := readDatabaseNowMillis(t, ctx, database)
	require.NoError(t, winner.refresh(ctx))
	refreshCompletedAt := readDatabaseNowMillis(t, ctx, database)
	refreshedExpiry := readLeaseExpiry(t, ctx, database, drainID)
	require.GreaterOrEqual(t, refreshedExpiry, refreshStartedAt+minimumTTL.Milliseconds())
	require.LessOrEqual(t, refreshedExpiry, refreshCompletedAt+(minimumTTL+ttlJitter).Milliseconds())
	require.Less(t, refreshedExpiry, existingExpiry, "refresh must replace rather than extend the expiry")

	expiredAt := readDatabaseNowMillis(t, ctx, database) - 1
	_, err = database.Conn().ExecContext(ctx, "UPDATE logdrain_state SET lease_expires_at = ? WHERE logdrain_id = ?", expiredAt, drainID)
	require.NoError(t, err)
	_, err = database.GetLeasedLogdrain(ctx, db.GetLeasedLogdrainParams{
		LogdrainID:   drainID,
		FencingToken: fencingToken,
	})
	require.ErrorIs(t, err, sql.ErrNoRows)
	rowsAffected, err = database.RecordLogdrainFailure(ctx, db.RecordLogdrainFailureParams{
		Status:           db.LogdrainStateStatusActive,
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
	var sameProcessToken string
	err = database.Conn().QueryRowContext(ctx, "SELECT fencing_token FROM logdrain_state WHERE logdrain_id = ?", drainID).Scan(&sameProcessToken)
	require.NoError(t, err)
	require.NotEqual(t, fencingToken, sameProcessToken)
	require.NoError(t, winner.refresh(ctx))

	expiredAt = readDatabaseNowMillis(t, ctx, database) - 1
	_, err = database.Conn().ExecContext(ctx, "UPDATE logdrain_state SET lease_expires_at = ? WHERE logdrain_id = ?", expiredAt, drainID)
	require.NoError(t, err)
	reacquirer, err := New(Config{DB: database, LeaseID: uid.New(""), Clock: testClock})
	require.NoError(t, err)
	reacquired, err = reacquirer.acquire(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, reacquired)
	var reacquiredLeaseID, reacquiredToken string
	err = database.Conn().QueryRowContext(ctx, "SELECT lease_id, fencing_token FROM logdrain_state WHERE logdrain_id = ?", drainID).Scan(&reacquiredLeaseID, &reacquiredToken)
	require.NoError(t, err)
	require.Equal(t, reacquirer.leaseID, reacquiredLeaseID)
	require.NotEqual(t, sameProcessToken, reacquiredToken)
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
	var expiryMillis int64
	err := database.Conn().QueryRowContext(ctx, "SELECT lease_expires_at FROM logdrain_state WHERE logdrain_id = ?", drainID).Scan(&expiryMillis)
	require.NoError(t, err)
	return expiryMillis
}

// readNextAttemptAt returns the stored database-time retry timestamp.
func readNextAttemptAt(t *testing.T, ctx context.Context, database db.Database, drainID string) int64 {
	t.Helper()
	var nextAttemptAtMillis int64
	err := database.Conn().QueryRowContext(ctx, "SELECT next_attempt_at FROM logdrain_state WHERE logdrain_id = ?", drainID).Scan(&nextAttemptAtMillis)
	require.NoError(t, err)
	return nextAttemptAtMillis
}
