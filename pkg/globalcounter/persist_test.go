package globalcounter_test

import (
	"database/sql"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/globalcounter"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCounter_Persist(t *testing.T) {
	database, _ := newDatabase(t)
	t.Run("names at the Unicode character limit can be persisted", func(t *testing.T) {
		prefix := uid.New(uid.RatelimitNamespacePrefix)
		name := prefix + strings.Repeat("界", 512-len(prefix))
		testClock := clock.NewTestClock()
		writer, err := globalcounter.New(globalcounter.Config{DB: database, Clock: testClock})
		require.NoError(t, err)
		globalcounter.Cleanup(t, writer)
		observer, err := globalcounter.New(globalcounter.Config{
			DB: database, Clock: testClock,
		})
		require.NoError(t, err)
		globalcounter.Cleanup(t, observer)
		require.NoError(t, writer.Add(globalcounter.AddParams{Name: name, Delta: -7}))
		require.Eventually(t, func() bool {
			testClock.Tick(5 * time.Second)
			count, err := observer.Get(t.Context(), globalcounter.GetParams{Name: name})
			return err == nil && count == -7
		}, 5*time.Second, 10*time.Millisecond)
	})

	t.Run("concurrent writers persist signed snapshots without losing names or revisions", func(t *testing.T) {
		testClock := clock.NewTestClock()
		writers := make([]*globalcounter.Counter, 2)
		for i := range writers {
			writer, err := globalcounter.New(globalcounter.Config{
				DB: database, Clock: testClock,
			})
			require.NoError(t, err)
			globalcounter.Cleanup(t, writer)
			writers[i] = writer
		}
		observer, err := globalcounter.New(globalcounter.Config{
			DB: database, Clock: testClock,
		})
		require.NoError(t, err)
		globalcounter.Cleanup(t, observer)
		shared := uid.New(uid.RatelimitNamespacePrefix)
		names := make([]string, 16)
		for i := range names {
			names[i] = uid.New(uid.RatelimitNamespacePrefix)
		}
		var workers sync.WaitGroup
		failures := make(chan error, len(names))
		start := make(chan struct{})
		for i, name := range names {
			workers.Go(func() {
				<-start
				delta := int64(3)
				if i%2 == 1 {
					delta = -1
				}
				for range 100 {
					for _, key := range []string{shared, name} {
						if err := writers[i%2].Add(globalcounter.AddParams{Name: key, Delta: delta}); err != nil {
							failures <- err
							return
						}
					}
				}
			})
		}
		close(start)
		workers.Wait()
		close(failures)
		for err := range failures {
			require.NoError(t, err)
		}
		require.Eventually(t, func() bool {
			testClock.Tick(5 * time.Second)
			count, err := observer.Get(t.Context(), globalcounter.GetParams{Name: shared})
			if err != nil || count != 1_600 {
				return false
			}
			for i, name := range names {
				want := int64(300)
				if i%2 == 1 {
					want = -100
				}
				count, err := observer.Get(t.Context(), globalcounter.GetParams{Name: name})
				if err != nil || count != want {
					return false
				}
			}
			return true
		}, 5*time.Second, 10*time.Millisecond)
	})
}

func TestCounter_DropsTimedOutSnapshotsWithoutBlockingLaterWrites(t *testing.T) {
	config := containers.MySQLIsolated(t)
	database, err := mysql.New(mysql.Config{PrimaryDSN: config.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	droppedName := uid.New(uid.RatelimitNamespacePrefix)
	healthyName := uid.New(uid.RatelimitNamespacePrefix)
	droppedBefore, persistedBefore := persistedRows(t, "dropped"), persistedRows(t, "persisted")

	t.Run("writer", func(t *testing.T) {
		writePool, err := sql.Open("mysql", config.DSN)
		require.NoError(t, err)
		writePool.SetMaxOpenConns(1)
		t.Cleanup(func() { require.NoError(t, writePool.Close()) })
		writer, err := globalcounter.New(globalcounter.Config{
			DB:    mysql.NewFromReplicas(database.RO(), mysql.NewReplicaFromDB(writePool, "rw", sqlcomment.Static{})),
			Clock: clock.NewTestClock(),
		})
		require.NoError(t, err)
		globalcounter.Cleanup(t, writer)
		awaitStartupPasses(t, writePool)
		conn, err := writePool.Conn(t.Context())
		require.NoError(t, err)
		release := sync.OnceFunc(func() { require.NoError(t, conn.Close()) })
		t.Cleanup(release)

		// The startup passes may have waited for the connection too, so count from here.
		waited := writePool.Stats().WaitDuration
		require.NoError(t, writer.Add(globalcounter.AddParams{Name: droppedName, Delta: 7}))
		// The occupied pool records completed waits only after the write times out.
		require.Eventually(t, func() bool { return writePool.Stats().WaitDuration > waited }, 10*time.Second, 10*time.Millisecond)
		release()

		count, err := writer.Get(t.Context(), globalcounter.GetParams{Name: droppedName})
		require.NoError(t, err)
		require.Equal(t, int64(7), count)
		require.NoError(t, writer.Add(globalcounter.AddParams{Name: healthyName, Delta: 19}))
	})

	// Subtest cleanup drains both workers before the observer checks persisted counts.
	observer, err := globalcounter.New(globalcounter.Config{DB: database, Clock: clock.NewTestClock()})
	require.NoError(t, err)
	globalcounter.Cleanup(t, observer)
	count, err := observer.Get(t.Context(), globalcounter.GetParams{Name: healthyName})
	require.NoError(t, err)
	require.Equal(t, int64(19), count)
	count, err = observer.Get(t.Context(), globalcounter.GetParams{Name: droppedName})
	require.NoError(t, err)
	require.Zero(t, count, "the timed-out snapshot must not be replayed")
	require.Equal(t, 1.0, persistedRows(t, "dropped")-droppedBefore, "the timed-out row is counted as dropped")
	require.Equal(t, 1.0, persistedRows(t, "persisted")-persistedBefore, "only the later row is counted as persisted")
}

func TestCounter_OpenWriteCircuitDropsSnapshotsWithoutBlockingReads(t *testing.T) {
	config := containers.MySQLIsolated(t)
	database, err := mysql.New(mysql.Config{PrimaryDSN: config.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	droppedName := uid.New(uid.RatelimitNamespacePrefix)
	probeName := uid.New(uid.RatelimitNamespacePrefix)

	t.Run("writer", func(t *testing.T) {
		writePool, err := sql.Open("mysql", config.DSN)
		require.NoError(t, err)
		writePool.SetMaxOpenConns(1)
		t.Cleanup(func() { require.NoError(t, writePool.Close()) })
		writer, err := globalcounter.New(globalcounter.Config{
			DB:    mysql.NewFromReplicas(database.RO(), mysql.NewReplicaFromDB(writePool, "rw", sqlcomment.Static{})),
			Clock: clock.NewTestClock(),
		})
		require.NoError(t, err)
		globalcounter.Cleanup(t, writer)
		awaitStartupPasses(t, writePool)
		conn, err := writePool.Conn(t.Context())
		require.NoError(t, err)
		release := sync.OnceFunc(func() { require.NoError(t, conn.Close()) })
		t.Cleanup(release)

		for range 5 {
			waited := writePool.Stats().WaitDuration
			require.NoError(t, writer.Add(globalcounter.AddParams{Name: droppedName, Delta: 7}))
			require.Eventually(t, func() bool { return writePool.Stats().WaitDuration > waited }, 10*time.Second, 10*time.Millisecond)
		}
		release()

		count, err := writer.Get(t.Context(), globalcounter.GetParams{Name: droppedName})
		require.NoError(t, err)
		require.Equal(t, int64(35), count)
		require.NoError(t, writer.Add(globalcounter.AddParams{Name: probeName, Delta: 17}))
	})

	observer, err := globalcounter.New(globalcounter.Config{DB: database, Clock: clock.NewTestClock()})
	require.NoError(t, err)
	globalcounter.Cleanup(t, observer)
	count, err := observer.Get(t.Context(), globalcounter.GetParams{Name: probeName})
	require.NoError(t, err)
	require.Zero(t, count, "an open write circuit must drop snapshots even after MySQL recovers")
}
