package globalcounter_test

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/globalcounter"
	"github.com/unkeyed/unkey/pkg/globalcounter/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCounter_ConditionalAdd(t *testing.T) {
	database, _ := newDatabase(t)
	newCounter := func(t *testing.T) *globalcounter.Counter {
		t.Helper()
		counter, err := globalcounter.New(globalcounter.Config{DB: database, Clock: clock.NewTestClock()})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)
		return counter
	}

	t.Run("callers racing for the last room cannot overshoot the limit", func(t *testing.T) {
		counter := newCounter(t)
		name := uid.New(uid.RatelimitNamespacePrefix)
		const limit, callers = 10, 64
		var applied atomic.Int64
		var group sync.WaitGroup
		start := make(chan struct{})
		for range callers {
			group.Go(func() {
				<-start
				_, err := counter.ConditionalAdd(t.Context(), globalcounter.ConditionalAddParams{
					Name: name,
					Delta: func(current int64) int64 {
						if current+1 > limit {
							return 0
						}
						applied.Add(1)
						return 1
					},
				})
				require.NoError(t, err)
			})
		}
		close(start)
		group.Wait()

		require.Equal(t, int64(limit), applied.Load())
		total, err := counter.Get(t.Context(), globalcounter.GetParams{Name: name})
		require.NoError(t, err)
		require.Equal(t, int64(limit), total)
	})

	t.Run("Delta sees this writer and peers together, and the result includes the delta", func(t *testing.T) {
		counter := newCounter(t)
		name := uid.New(uid.RatelimitNamespacePrefix)
		require.NoError(t, (&db.BulkQueries{}).Persist(t.Context(), database.RW(), []db.PersistParams{
			{Name: name, CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: 70, Revision: 1},
		}))
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: 5}))

		var seen int64
		total, err := counter.ConditionalAdd(t.Context(), globalcounter.ConditionalAddParams{Name: name, Delta: func(current int64) int64 { seen = current; return -3 }})
		require.NoError(t, err)
		require.Equal(t, int64(75), seen)
		require.Equal(t, int64(72), total)
	})

	t.Run("a zero delta writes nothing", func(t *testing.T) {
		counter := newCounter(t)
		name := uid.New(uid.RatelimitNamespacePrefix)
		entries, skipped := memoryEntries(t), addsTotal(t, "skipped")

		total, err := counter.ConditionalAdd(t.Context(), globalcounter.ConditionalAddParams{Name: name, Delta: func(int64) int64 { return 0 }})
		require.NoError(t, err)
		require.Zero(t, total)
		require.Zero(t, memoryEntries(t)-entries, "a skipped add must not create a memory entry")
		require.Equal(t, 1.0, addsTotal(t, "skipped")-skipped)

		counter.Close()
		observer := newCounter(t)
		count, err := observer.Get(t.Context(), globalcounter.GetParams{Name: name})
		require.NoError(t, err)
		require.Zero(t, count)
		persisted := persistedRows(t, "persisted")
		observer.Close()
		require.Zero(t, persistedRows(t, "persisted")-persisted)
	})

	t.Run("overflowing deltas and totals are rejected without a change", func(t *testing.T) {
		counter := newCounter(t)
		name := uid.New(uid.RatelimitNamespacePrefix)
		require.NoError(t, (&db.BulkQueries{}).Persist(t.Context(), database.RW(), []db.PersistParams{
			{Name: name, CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: math.MaxInt64, Revision: 1},
		}))
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: -1}))

		called := false
		_, err := counter.ConditionalAdd(t.Context(), globalcounter.ConditionalAddParams{Name: name, Delta: func(int64) int64 { called = true; return 1 }})
		require.NoError(t, err, "the total is MaxInt64-1, so a delta of 1 fits")
		require.True(t, called)

		_, err = counter.ConditionalAdd(t.Context(), globalcounter.ConditionalAddParams{Name: name, Delta: func(int64) int64 { return 1 }})
		require.Error(t, err, "the total is MaxInt64 and a delta of 1 overflows")
		count, err := counter.Get(t.Context(), globalcounter.GetParams{Name: name})
		require.NoError(t, err)
		require.Equal(t, int64(math.MaxInt64), count)
	})

	t.Run("a rotation after the peer read makes the decision start over", func(t *testing.T) {
		counter := newCounter(t)
		name := uid.New(uid.RatelimitNamespacePrefix)
		stale := globalcounter.Sequence(counter)
		globalcounter.RotateOnce(counter)

		require.True(t, globalcounter.DecideAt(counter, name, stale), "the peer total belongs to the old pair of writer ids")
		total, err := counter.Get(t.Context(), globalcounter.GetParams{Name: name})
		require.NoError(t, err)
		require.Zero(t, total, "a refused decision must not change anything")
		require.False(t, globalcounter.DecideAt(counter, name, globalcounter.Sequence(counter)))
	})

	t.Run("a change that overflows this writer's own total is rejected", func(t *testing.T) {
		counter := newCounter(t)
		name := uid.New(uid.RatelimitNamespacePrefix)
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: math.MaxInt64}))
		globalcounter.RotateOnce(counter)
		// The peers offset the previous writer id, so the total fits while the sum
		// of the current and previous writer id does not.
		require.NoError(t, (&db.BulkQueries{}).Persist(t.Context(), database.RW(), []db.PersistParams{
			{Name: name, CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: -math.MaxInt64, Revision: 1},
		}))

		_, err := counter.ConditionalAdd(t.Context(), globalcounter.ConditionalAddParams{Name: name, Delta: func(int64) int64 { return 1 }})
		require.Error(t, err)
		total, err := counter.Get(t.Context(), globalcounter.GetParams{Name: name})
		require.NoError(t, err, "a rejected change must leave the counter readable")
		require.Zero(t, total)
	})
}
