package globalcounter_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/globalcounter"
	"github.com/unkeyed/unkey/pkg/globalcounter/db"
	"github.com/unkeyed/unkey/pkg/mysql"
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
		globalcounter.RotateOnce(counter)
		require.Equal(t, int64(15), get(t, counter, name))
		require.Equal(t, int64(15), get(t, counter, name))

		require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: -2}))
		require.Equal(t, int64(13), get(t, counter, name))
		awaitStored(t, name, 13)
		require.Equal(t, int64(13), observed(), "another writer reads the same total")
	})
}
