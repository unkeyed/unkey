package globalcounter_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/globalcounter"
	"github.com/unkeyed/unkey/pkg/globalcounter/db"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCounter_Eviction(t *testing.T) {
	config := containers.MySQLIsolated(t)
	database, err := mysql.New(mysql.Config{PrimaryDSN: config.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	t.Run("removes expired rows and memory but keeps live and unexpiring counters", func(t *testing.T) {
		testClock := clock.NewTestClock()
		newCounter := func() *globalcounter.Counter {
			counter, err := globalcounter.New(globalcounter.Config{DB: database, Clock: testClock})
			require.NoError(t, err)
			globalcounter.Cleanup(t, counter)
			return counter
		}
		writer, observer := newCounter(), newCounter()
		deleteAfter := testClock.Now().Add(time.Minute)
		entriesBefore := memoryEntries(t)
		expiredName, liveName, foreverName := uid.New(uid.RatelimitNamespacePrefix), uid.New(uid.RatelimitNamespacePrefix), uid.New(uid.RatelimitNamespacePrefix)
		require.NoError(t, writer.Add(globalcounter.AddParams{Name: expiredName, Delta: 5, DeleteAfter: deleteAfter}))
		require.NoError(t, writer.Add(globalcounter.AddParams{Name: liveName, Delta: 6, DeleteAfter: deleteAfter.Add(time.Hour)}))
		require.NoError(t, writer.Add(globalcounter.AddParams{Name: foreverName, Delta: 7}))
		require.NoError(t, writer.Add(globalcounter.AddParams{Name: foreverName, Delta: 1}))
		require.Equal(t, 3.0, memoryEntries(t)-entriesBefore, "rewriting a name must not count it again")
		names := map[string]int64{expiredName: 5, liveName: 6, foreverName: 8}
		// Close persists every local counter synchronously. Polling with a ticking
		// test clock would eventually fire an eviction pass and delete the row.
		writer.Close()
		for name, want := range names {
			got, err := observer.Get(t.Context(), globalcounter.GetParams{Name: name})
			require.NoError(t, err)
			require.Equal(t, want, got, name)
		}

		testClock.Tick(2 * time.Minute)
		globalcounter.EvictOnce(t.Context(), observer)
		globalcounter.EvictOnce(t.Context(), writer)
		testClock.Tick(5 * time.Second)

		want := map[string]int64{expiredName: 0, liveName: 6, foreverName: 8}
		require.Equal(t, 2.0, memoryEntries(t)-entriesBefore, "eviction removes only the expired name")
		for name, count := range want {
			for _, counter := range []*globalcounter.Counter{writer, observer} {
				got, err := counter.Get(t.Context(), globalcounter.GetParams{Name: name})
				require.NoError(t, err)
				require.Equal(t, count, got, name)
			}
		}
	})

	t.Run("keeps memory when the row delete fails", func(t *testing.T) {
		brokenPool, err := sql.Open("mysql", config.DSN)
		require.NoError(t, err)
		require.NoError(t, brokenPool.Close())
		testClock := clock.NewTestClock()
		counter, err := globalcounter.New(globalcounter.Config{
			DB:    mysql.NewFromReplicas(database.RO(), mysql.NewReplicaFromDB(brokenPool, "rw", sqlcomment.Static{})),
			Clock: testClock,
		})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)
		name := uid.New(uid.RatelimitNamespacePrefix)
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: 5, DeleteAfter: testClock.Now().Add(time.Minute)}))

		testClock.Tick(10 * time.Minute)
		globalcounter.EvictOnce(t.Context(), counter)

		count, err := counter.Get(t.Context(), globalcounter.GetParams{Name: name})
		require.NoError(t, err)
		require.Equal(t, int64(5), count, "dropping memory while a row may survive would restart revisions below it")
	})

	t.Run("keeps an entry for a grace period after its latest change, even if its deletion time passed long ago", func(t *testing.T) {
		testClock := clock.NewTestClock()
		counter, err := globalcounter.New(globalcounter.Config{DB: database, Clock: testClock})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)
		entriesBefore := memoryEntries(t)
		name := uid.New(uid.RatelimitNamespacePrefix)
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: 5, DeleteAfter: testClock.Now().Add(-2 * time.Hour)}))

		globalcounter.EvictOnce(t.Context(), counter)
		require.Equal(t, 1.0, memoryEntries(t)-entriesBefore, "a snapshot of the entry may still be in flight")

		testClock.Tick(2 * time.Minute)
		globalcounter.EvictOnce(t.Context(), counter)
		require.Zero(t, memoryEntries(t)-entriesBefore)
	})

	t.Run("forgets retired entries even when the deletes fail", func(t *testing.T) {
		brokenPool, err := sql.Open("mysql", config.DSN)
		require.NoError(t, err)
		require.NoError(t, brokenPool.Close())
		counter, err := globalcounter.New(globalcounter.Config{
			DB:    mysql.NewFromReplicas(database.RO(), mysql.NewReplicaFromDB(brokenPool, "rw", sqlcomment.Static{})),
			Clock: clock.NewTestClock(),
		})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)
		entriesBefore, abandonedBefore := memoryEntries(t), persistedRows(t, "abandoned")
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: uid.New(uid.RatelimitNamespacePrefix), Delta: 5}))
		globalcounter.RotateOnce(counter)
		globalcounter.RotateOnce(counter)

		globalcounter.EvictOnce(t.Context(), counter)

		require.Zero(t, memoryEntries(t)-entriesBefore, "a retired entry needs no database access to be forgotten")
		require.Equal(t, 1.0, persistedRows(t, "abandoned")-abandonedBefore, "its snapshot never reached MySQL")
	})

	t.Run("deletes every expired row across batches", func(t *testing.T) {
		testClock := clock.NewTestClock()
		counter, err := globalcounter.New(globalcounter.Config{DB: database, Clock: testClock})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)

		// One more expired row than a single delete batch holds.
		expired := sql.NullInt64{Int64: testClock.Now().Add(-time.Hour).UnixMilli(), Valid: true}
		live := sql.NullInt64{Int64: testClock.Now().Add(time.Hour).UnixMilli(), Valid: true}
		writerID := uid.New("", 16)
		liveName := uid.New(uid.RatelimitNamespacePrefix)
		rows := make([]db.PersistParams, 0, 10_002)
		for range 10_001 {
			rows = append(rows, db.PersistParams{Name: uid.New(uid.RatelimitNamespacePrefix), CompactDelayMs: testCompactDelayMs, WriterID: writerID, Count: 1, Revision: 1, DeleteAfter: expired})
		}
		rows = append(rows, db.PersistParams{Name: liveName, CompactDelayMs: testCompactDelayMs, WriterID: writerID, Count: 1, Revision: 1, DeleteAfter: live})
		require.NoError(t, (&db.BulkQueries{}).Persist(t.Context(), database.RW(), rows))

		globalcounter.EvictOnce(t.Context(), counter)

		stale := db.DeleteExpiredParams{CutoffMs: sql.NullInt64{Int64: expired.Int64 + 1, Valid: true}, Limit: 100_000}
		remaining, err := db.Query.DeleteExpired(t.Context(), database.RW(), stale)
		require.NoError(t, err)
		require.Zero(t, remaining)
		count, err := counter.Get(t.Context(), globalcounter.GetParams{Name: liveName})
		require.NoError(t, err)
		require.Equal(t, int64(1), count)
	})
	t.Run("abandons the pass when cancelled", func(t *testing.T) {
		writePool, err := sql.Open("mysql", config.DSN)
		require.NoError(t, err)
		writePool.SetMaxOpenConns(1)
		t.Cleanup(func() { require.NoError(t, writePool.Close()) })
		testClock := clock.NewTestClock()
		counter, err := globalcounter.New(globalcounter.Config{
			DB:    mysql.NewFromReplicas(database.RO(), mysql.NewReplicaFromDB(writePool, "rw", sqlcomment.Static{})),
			Clock: testClock,
		})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)
		awaitStartupPasses(t, writePool)
		name := uid.New(uid.RatelimitNamespacePrefix)
		expired := sql.NullInt64{Int64: testClock.Now().Add(-2 * time.Hour).UnixMilli(), Valid: true}
		require.NoError(t, (&db.BulkQueries{}).Persist(t.Context(), database.RW(), []db.PersistParams{
			{Name: name, CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: 9, Revision: 1, DeleteAfter: expired},
		}))
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: 1, DeleteAfter: testClock.Now().Add(-2 * time.Hour)}))

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		globalcounter.EvictOnce(ctx, counter)

		count, err := counter.Get(t.Context(), globalcounter.GetParams{Name: name})
		require.NoError(t, err)
		require.Equal(t, int64(10), count, "a cancelled pass must not delete the row or the memory entry")
		stale := db.DeleteExpiredParams{CutoffMs: sql.NullInt64{Int64: expired.Int64 + 1, Valid: true}, Limit: 100}
		remaining, err := db.Query.DeleteExpired(t.Context(), database.RW(), stale)
		require.NoError(t, err)
		require.Positive(t, remaining)
	})
	t.Run("forgets its own row too, so a restarted revision is not ignored", func(t *testing.T) {
		writePool, err := sql.Open("mysql", config.DSN)
		require.NoError(t, err)
		writePool.SetMaxOpenConns(1)
		t.Cleanup(func() { require.NoError(t, writePool.Close()) })
		testClock := clock.NewTestClock()
		writer, err := globalcounter.New(globalcounter.Config{
			DB:    mysql.NewFromReplicas(database.RO(), mysql.NewReplicaFromDB(writePool, "rw", sqlcomment.Static{})),
			Clock: testClock,
		})
		require.NoError(t, err)
		globalcounter.Cleanup(t, writer)
		awaitStartupPasses(t, writePool)
		observerClock := clock.NewTestClock()
		observer, err := globalcounter.New(globalcounter.Config{DB: database, Clock: observerClock})
		require.NoError(t, err)
		globalcounter.Cleanup(t, observer)
		name := uid.New(uid.RatelimitNamespacePrefix)
		observed := func(want int64) bool {
			observerClock.Tick(5 * time.Second)
			count, err := observer.Get(t.Context(), globalcounter.GetParams{Name: name})
			return err == nil && count == want
		}

		// Revision 1 persists with an expiry of one hour.
		require.NoError(t, writer.Add(globalcounter.AddParams{Name: name, Delta: 1, DeleteAfter: testClock.Now().Add(time.Hour)}))
		require.Eventually(t, func() bool { return observed(1) }, 10*time.Second, 50*time.Millisecond)

		// Revision 2 shortens the expiry to one minute, but its snapshot is dropped,
		// so the row keeps revision 1 and the later expiry.
		conn, err := writePool.Conn(t.Context())
		require.NoError(t, err)
		release := sync.OnceFunc(func() { require.NoError(t, conn.Close()) })
		t.Cleanup(release)
		// The startup passes may have waited for the connection too, so count from here.
		waited := writePool.Stats().WaitDuration
		require.NoError(t, writer.Add(globalcounter.AddParams{Name: name, Delta: 1, DeleteAfter: testClock.Now().Add(time.Minute)}))
		require.Eventually(t, func() bool { return writePool.Stats().WaitDuration > waited }, 10*time.Second, 10*time.Millisecond)
		release()

		// The entry expires in memory while the row, expiring in an hour, survives.
		testClock.Tick(10 * time.Minute)
		globalcounter.EvictOnce(t.Context(), writer)
		require.Eventually(t, func() bool { return observed(0) }, 10*time.Second, 50*time.Millisecond, "the writer's own row must go with its entry")

		// The name starts again at revision 1. A surviving row at revision 1 or
		// higher would ignore it.
		require.NoError(t, writer.Add(globalcounter.AddParams{Name: name, Delta: 5, DeleteAfter: testClock.Now().Add(time.Hour)}))
		require.Eventually(t, func() bool { return observed(5) }, 10*time.Second, 50*time.Millisecond)
	})
}
