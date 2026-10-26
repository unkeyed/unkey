package globalcounter_test

import (
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

func TestCounter_Rotation(t *testing.T) {
	config := containers.MySQLIsolated(t)
	database, err := mysql.New(mysql.Config{PrimaryDSN: config.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	newCounter := func(t *testing.T, testClock clock.Clock) *globalcounter.Counter {
		t.Helper()
		counter, err := globalcounter.New(globalcounter.Config{DB: database, Clock: testClock})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)
		return counter
	}
	rows := func(t *testing.T, name string) map[string]int64 {
		t.Helper()
		listed, err := db.Query.ListRows(t.Context(), database.RW(), db.ListRowsParams{Name: name})
		require.NoError(t, err)
		counts := make(map[string]int64, len(listed))
		for _, row := range listed {
			counts[row.WriterID] = row.Count
		}
		return counts
	}
	stored := func(t *testing.T, name string) int64 {
		t.Helper()
		sum, err := db.Query.Read(t.Context(), database.RW(), db.ReadParams{Name: name, CurrentWriterID: "", PreviousWriterID: ""})
		require.NoError(t, err)
		return sum
	}
	awaitStored := func(t *testing.T, name string, want int64) {
		t.Helper()
		require.Eventually(t, func() bool { return stored(t, name) == want }, 10*time.Second, 20*time.Millisecond)
	}
	get := func(t *testing.T, counter *globalcounter.Counter, name string) int64 {
		t.Helper()
		total, err := counter.Get(t.Context(), globalcounter.GetParams{Name: name})
		require.NoError(t, err)
		return total
	}

	t.Run("each writer id starts at zero and persists as its own row, so the total is their sum", func(t *testing.T) {
		counter := newCounter(t, clock.NewTestClock())
		name := uid.New(uid.RatelimitNamespacePrefix)
		first := globalcounter.WriterID(counter)

		require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: 10}))
		globalcounter.RotateOnce(counter)
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: -3}))

		require.NotEqual(t, first, globalcounter.WriterID(counter))
		require.Equal(t, first, globalcounter.PreviousWriterID(counter))
		require.Equal(t, int64(7), get(t, counter, name), "a rotation must not reset the total, and the new id must not inherit the old count")
		awaitStored(t, name, 7)
		require.Equal(t, map[string]int64{first: 10, globalcounter.WriterID(counter): -3}, rows(t, name))
	})

	t.Run("the writer id rotates on a schedule, and not at startup", func(t *testing.T) {
		testClock := clock.NewTestClock()
		counter := newCounter(t, testClock)
		first := globalcounter.WriterID(counter)
		require.Equal(t, first, globalcounter.WriterID(counter))

		testClock.Tick(10 * time.Minute)
		require.Equal(t, first, globalcounter.WriterID(counter), "ten minutes is less than the shortest interval")
		require.Eventually(t, func() bool {
			testClock.Tick(time.Minute)
			return globalcounter.WriterID(counter) != first
		}, 5*time.Second, 10*time.Millisecond, "the interval with its jitter is at most 18 minutes")
		require.Equal(t, first, globalcounter.PreviousWriterID(counter))
	})

	t.Run("reads stay exact while writer ids rotate and then leave memory", func(t *testing.T) {
		counter := newCounter(t, clock.NewTestClock())
		observerClock := clock.NewTestClock()
		observer := newCounter(t, observerClock)
		name := uid.New(uid.RatelimitNamespacePrefix)
		observed := func() int64 {
			observerClock.Tick(10 * time.Second) // expire the observer's cached peer sum
			return get(t, observer, name)
		}

		require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: 10}))
		awaitStored(t, name, 10)
		require.Equal(t, int64(10), get(t, counter, name))
		require.Equal(t, int64(10), observed())

		// The first id is now the previous one. The writer still holds it in memory,
		// so its stored row must not be counted again.
		globalcounter.RotateOnce(counter)
		require.Equal(t, int64(10), get(t, counter, name))
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: 5}))
		awaitStored(t, name, 15)
		require.Equal(t, int64(15), get(t, counter, name))

		// The first id is two rotations old and no longer counts from memory. Its
		// stored row counts instead, so nothing is lost or counted twice. The peer
		// cache key changed with the rotation, so this read does not serve the old
		// sum.
		entries := memoryEntries(t)
		globalcounter.RotateOnce(counter)
		require.Equal(t, int64(15), get(t, counter, name))
		globalcounter.EvictOnce(t.Context(), counter)
		require.Equal(t, -1.0, memoryEntries(t)-entries, "eviction forgets the entry of the writer id that left memory")
		require.Equal(t, int64(15), get(t, counter, name))

		require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: -2}))
		require.Equal(t, int64(13), get(t, counter, name))
		awaitStored(t, name, 13)
		require.Equal(t, int64(13), observed(), "another writer reads the same total")
	})

	// A writer whose database writes time out drops its snapshots, as a database
	// outage does. The helper returns it with the snapshot of name dropped, and a
	// function that lets writes through again.
	droppedWriter := func(t *testing.T, name string, deleteAfter time.Duration) (*globalcounter.Counter, *clock.TestClock, func()) {
		t.Helper()
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
		conn, err := writePool.Conn(t.Context())
		require.NoError(t, err)
		release := sync.OnceFunc(func() { require.NoError(t, conn.Close()) })
		t.Cleanup(release)

		waited := writePool.Stats().WaitDuration
		params := globalcounter.AddParams{Name: name, Delta: 7}
		if deleteAfter != 0 {
			params.DeleteAfter = testClock.Now().Add(deleteAfter)
		}
		require.NoError(t, writer.Add(params))
		require.Eventually(t, func() bool { return writePool.Stats().WaitDuration > waited }, 10*time.Second, 10*time.Millisecond)
		return writer, testClock, release
	}

	t.Run("a dropped snapshot of the previous writer id is queued again and persisted", func(t *testing.T) {
		name := uid.New(uid.RatelimitNamespacePrefix)
		writer, _, release := droppedWriter(t, name, 0)
		require.Zero(t, stored(t, name), "the write timed out and was dropped")

		globalcounter.RotateOnce(writer)
		release()
		globalcounter.EvictOnce(t.Context(), writer)

		awaitStored(t, name, 7)
		require.Equal(t, int64(7), get(t, writer, name))
	})

	t.Run("an entry forgotten before MySQL acknowledged it is counted as abandoned", func(t *testing.T) {
		name := uid.New(uid.RatelimitNamespacePrefix)
		writer, _, release := droppedWriter(t, name, 0)
		abandoned := persistedRows(t, "abandoned")

		globalcounter.RotateOnce(writer)
		globalcounter.RotateOnce(writer)
		release()
		globalcounter.EvictOnce(t.Context(), writer)

		require.Equal(t, 1.0, persistedRows(t, "abandoned")-abandoned)
		require.Zero(t, get(t, writer, name), "the lost snapshot is gone from the total")
		require.Zero(t, stored(t, name))
	})

	t.Run("an entry that the same pass forgets is not queued again", func(t *testing.T) {
		name := uid.New(uid.RatelimitNamespacePrefix)
		writer, testClock, release := droppedWriter(t, name, -2*time.Hour)
		globalcounter.RotateOnce(writer)
		release()
		testClock.Tick(2 * time.Minute)

		globalcounter.EvictOnce(t.Context(), writer)

		// A snapshot that the pass queued would be written within the batch interval.
		time.Sleep(2 * time.Second)
		require.Zero(t, stored(t, name), "its row is deleted by this pass and its entry forgotten, so nothing may publish it")
	})
}
