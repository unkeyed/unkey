package globalcounter_test

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/globalcounter"
	"github.com/unkeyed/unkey/pkg/globalcounter/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCounter_Cache(t *testing.T) {
	database, readPool := newDatabase(t)
	bulk := &db.BulkQueries{}

	t.Run("cache holds other writers until its five second deadline", func(t *testing.T) {
		testClock := clock.NewTestClock()
		counter, err := globalcounter.New(globalcounter.Config{DB: database, Clock: testClock})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)
		name := uid.New(uid.RatelimitNamespacePrefix)
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: 100}))
		rows := []db.PersistParams{
			{Name: name, CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: 7, Revision: 1},
			{Name: name, CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: -3, Revision: 1},
		}
		require.NoError(t, bulk.Persist(t.Context(), database.RW(), rows))
		params := globalcounter.GetParams{Name: name}
		count, err := counter.Get(t.Context(), params)
		require.NoError(t, err)
		require.Equal(t, int64(104), count)

		rows[0].Count = 30
		rows[0].Revision++
		require.NoError(t, bulk.Persist(t.Context(), database.RW(), rows))
		testClock.Tick(time.Second - time.Nanosecond)
		count, err = counter.Get(t.Context(), params)
		require.NoError(t, err)
		require.Equal(t, int64(104), count)

		testClock.Tick(4*time.Second + time.Nanosecond)
		count, err = counter.Get(t.Context(), params)
		require.NoError(t, err)
		require.Equal(t, int64(127), count)
	})

	t.Run("expired cache refresh includes deductions", func(t *testing.T) {
		testClock := clock.NewTestClock()
		counter, err := globalcounter.New(globalcounter.Config{
			DB: database, Clock: testClock,
		})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)
		row := db.PersistParams{
			Name: uid.New(uid.RatelimitNamespacePrefix), CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: 70, Revision: 1,
		}
		require.NoError(t, bulk.Persist(t.Context(), database.RW(), []db.PersistParams{row}))
		count, err := counter.Get(t.Context(), globalcounter.GetParams{Name: row.Name})
		require.NoError(t, err)
		require.Equal(t, int64(70), count)

		row.Count = -9
		row.Revision++
		require.NoError(t, bulk.Persist(t.Context(), database.RW(), []db.PersistParams{row}))
		testClock.Tick(5 * time.Second)
		count, err = counter.Get(t.Context(), globalcounter.GetParams{Name: row.Name})
		require.NoError(t, err)
		require.Equal(t, int64(-9), count)
		count, err = counter.Get(t.Context(), globalcounter.GetParams{Name: row.Name})
		require.NoError(t, err)
		require.Equal(t, int64(-9), count)
	})

	t.Run("one second old data returns while the replica refresh is blocked", func(t *testing.T) {
		testClock := clock.NewTestClock()
		counter, err := globalcounter.New(globalcounter.Config{
			DB: database, Clock: testClock,
		})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)
		row := db.PersistParams{
			Name: uid.New(uid.RatelimitNamespacePrefix), CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: 19, Revision: 1,
		}
		require.NoError(t, bulk.Persist(t.Context(), database.RW(), []db.PersistParams{row}))
		params := globalcounter.GetParams{Name: row.Name}
		count, err := counter.Get(t.Context(), params)
		require.NoError(t, err)
		require.Equal(t, int64(19), count)
		row.Count = 31
		row.Revision++
		require.NoError(t, bulk.Persist(t.Context(), database.RW(), []db.PersistParams{row}))

		conn, err := readPool.Conn(t.Context())
		require.NoError(t, err)
		release := sync.OnceFunc(func() { require.NoError(t, conn.Close()) })
		t.Cleanup(release)
		waiting := readPool.Stats().WaitCount
		testClock.Tick(time.Second)
		ctx, cancel := context.WithCancel(t.Context())
		t.Cleanup(cancel)
		count, err = counter.Get(ctx, params)
		require.NoError(t, err)
		require.Equal(t, int64(19), count)
		cancel()
		require.Eventually(t, func() bool { return readPool.Stats().WaitCount > waiting }, time.Second, time.Millisecond)
		release()
		require.Eventually(t, func() bool {
			count, err := counter.Get(t.Context(), params)
			return err == nil && count == 31
		}, time.Second, time.Millisecond)
	})

	t.Run("expired reads return errors without caching failures", func(t *testing.T) {
		testClock := clock.NewTestClock()
		counter, err := globalcounter.New(globalcounter.Config{
			DB: database, Clock: testClock,
		})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)
		name := uid.New(uid.RatelimitNamespacePrefix)
		rows := []db.PersistParams{
			{Name: name, CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: 11, Revision: 1},
			{Name: name, CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: 17, Revision: 1},
		}
		require.NoError(t, bulk.Persist(t.Context(), database.RW(), rows))
		params := globalcounter.GetParams{Name: name}
		count, err := counter.Get(t.Context(), params)
		require.NoError(t, err)
		require.Equal(t, int64(28), count)

		rows[0].Count = math.MaxInt64
		rows[0].Revision++
		require.NoError(t, bulk.Persist(t.Context(), database.RW(), rows))
		count, err = counter.Get(t.Context(), params)
		require.NoError(t, err)
		require.Equal(t, int64(28), count)

		testClock.Tick(5 * time.Second)
		failures := getsTotal(t, "error")
		count, err = counter.Get(t.Context(), params)
		require.Error(t, err)
		require.Zero(t, count)
		require.Equal(t, 1.0, getsTotal(t, "error")-failures)
		rows[0].Count = -3
		rows[0].Revision++
		require.NoError(t, bulk.Persist(t.Context(), database.RW(), rows))
		count, err = counter.Get(t.Context(), params)
		require.NoError(t, err)
		require.Equal(t, int64(14), count)
	})

	t.Run("read failures trip the breaker without blocking cache hits and recover", func(t *testing.T) {
		testClock := clock.NewTestClock()
		counter, err := globalcounter.New(globalcounter.Config{
			DB: database, Clock: testClock,
		})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)
		cachedName := uid.New(uid.RatelimitNamespacePrefix)
		uncachedName := uid.New(uid.RatelimitNamespacePrefix)
		failingName := uid.New(uid.RatelimitNamespacePrefix)
		require.NoError(t, bulk.Persist(t.Context(), database.RW(), []db.PersistParams{
			{Name: cachedName, CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: 37, Revision: 1},
			{Name: uncachedName, CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: 61, Revision: 1},
			{Name: failingName, CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: math.MaxInt64, Revision: 1},
			{Name: failingName, CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: 1, Revision: 1},
		}))
		count, err := counter.Get(t.Context(), globalcounter.GetParams{Name: cachedName})
		require.NoError(t, err)
		require.Equal(t, int64(37), count)

		for range 5 {
			_, err := counter.Get(t.Context(), globalcounter.GetParams{Name: failingName})
			require.Error(t, err)
			require.NotErrorIs(t, err, circuitbreaker.ErrTripped)
		}
		count, err = counter.Get(t.Context(), globalcounter.GetParams{Name: uncachedName})
		require.ErrorIs(t, err, circuitbreaker.ErrTripped)
		require.Zero(t, count)
		count, err = counter.Get(t.Context(), globalcounter.GetParams{Name: cachedName})
		require.NoError(t, err)
		require.Equal(t, int64(37), count)

		testClock.Tick(time.Minute + time.Nanosecond)
		count, err = counter.Get(t.Context(), globalcounter.GetParams{Name: uncachedName})
		require.NoError(t, err)
		require.Equal(t, int64(61), count)
	})
}
