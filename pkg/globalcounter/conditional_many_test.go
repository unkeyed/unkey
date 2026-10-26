package globalcounter_test

import (
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/globalcounter"
	"github.com/unkeyed/unkey/pkg/globalcounter/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCounter_ConditionalAddMany(t *testing.T) {
	database, _ := newDatabase(t)
	newCounter := func(t *testing.T) *globalcounter.Counter {
		t.Helper()
		counter, err := globalcounter.New(globalcounter.Config{DB: database, Clock: clock.NewTestClock()})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)
		return counter
	}
	names := func(count int) []string {
		out := make([]string, count)
		for i := range out {
			out[i] = uid.New(uid.RatelimitNamespacePrefix)
		}
		return out
	}
	// within adds cost to a name unless its total would pass limit, and records
	// whether it did, the way a limiter would.
	within := func(name string, cost, limit int64, passed *bool) globalcounter.ConditionalAddCheck {
		return globalcounter.ConditionalAddCheck{
			Name: name,
			Delta: func(current int64) (int64, bool) {
				*passed = current+cost <= limit
				return cost, !*passed
			},
		}
	}
	fixed := func(name string, delta int64) globalcounter.ConditionalAddCheck {
		return globalcounter.ConditionalAddCheck{Name: name, Delta: func(int64) (int64, bool) { return delta, false }}
	}
	total := func(t *testing.T, counter *globalcounter.Counter, name string) int64 {
		t.Helper()
		count, err := counter.Get(t.Context(), globalcounter.GetParams{Name: name})
		require.NoError(t, err)
		return count
	}

	t.Run("an abort in any check leaves every check untouched and every check decided", func(t *testing.T) {
		counter := newCounter(t)
		pair := names(2)
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: pair[1], Delta: 9}))
		entries := memoryEntries(t)

		var firstPassed, secondPassed bool
		totals, err := counter.ConditionalAddMany(t.Context(), globalcounter.ConditionalAddManyParams{
			Checks: []globalcounter.ConditionalAddCheck{within(pair[0], 5, 10, &firstPassed), within(pair[1], 5, 10, &secondPassed)},
		})
		require.NoError(t, err)
		require.True(t, firstPassed, "the first check is within its limit and must still be decided")
		require.False(t, secondPassed)
		require.Equal(t, []int64{0, 9}, totals)
		require.Zero(t, total(t, counter, pair[0]), "the first check must not apply when the second aborts")
		require.Equal(t, int64(9), total(t, counter, pair[1]))
		require.Zero(t, memoryEntries(t)-entries, "an aborted call must not create entries")

		totals, err = counter.ConditionalAddMany(t.Context(), globalcounter.ConditionalAddManyParams{
			Checks: []globalcounter.ConditionalAddCheck{fixed(pair[0], 3), fixed(pair[1], -2)},
		})
		require.NoError(t, err)
		require.Equal(t, []int64{3, 7}, totals)
		require.Equal(t, int64(3), total(t, counter, pair[0]))
		require.Equal(t, int64(7), total(t, counter, pair[1]))
		require.Equal(t, 1.0, memoryEntries(t)-entries, "only the new name adds an entry")
	})

	t.Run("a zero change leaves its check alone while others apply", func(t *testing.T) {
		counter := newCounter(t)
		pair := names(2)
		persisted := persistedRows(t, "persisted")
		_, err := counter.ConditionalAddMany(t.Context(), globalcounter.ConditionalAddManyParams{
			Checks: []globalcounter.ConditionalAddCheck{fixed(pair[0], 0), fixed(pair[1], 4)},
		})
		require.NoError(t, err)
		counter.Close()

		// The applied check persists once when Close drains the queue and once in the
		// final snapshot. A phantom entry for the zero check would add a third row.
		require.Equal(t, 2.0, persistedRows(t, "persisted")-persisted, "the zero check must create no entry")
	})

	t.Run("each check sees this writer and peers together", func(t *testing.T) {
		counter := newCounter(t)
		pair := names(2)
		require.NoError(t, (&db.BulkQueries{}).Persist(t.Context(), database.RW(), []db.PersistParams{
			{Name: pair[0], CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: 70, Revision: 1},
			{Name: pair[1], CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: -20, Revision: 1},
		}))
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: pair[1], Delta: 5}))

		seen := map[string]int64{}
		record := func(name string) globalcounter.ConditionalAddCheck {
			return globalcounter.ConditionalAddCheck{Name: name, Delta: func(current int64) (int64, bool) { seen[name] = current; return 0, false }}
		}
		_, err := counter.ConditionalAddMany(t.Context(), globalcounter.ConditionalAddManyParams{
			Checks: []globalcounter.ConditionalAddCheck{record(pair[0]), record(pair[1])},
		})
		require.NoError(t, err)
		require.Equal(t, map[string]int64{pair[0]: 70, pair[1]: -15}, seen)
	})

	t.Run("an overflow in one check applies none of them", func(t *testing.T) {
		counter := newCounter(t)
		pair := names(2)
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: pair[1], Delta: math.MaxInt64}))

		_, err := counter.ConditionalAddMany(t.Context(), globalcounter.ConditionalAddManyParams{
			Checks: []globalcounter.ConditionalAddCheck{fixed(pair[0], 4), fixed(pair[1], 1)},
		})
		require.Error(t, err)
		require.Zero(t, total(t, counter, pair[0]), "the first check must not apply when the second overflows")
		require.Equal(t, int64(math.MaxInt64), total(t, counter, pair[1]))
	})

	t.Run("duplicate names are rejected before any check runs", func(t *testing.T) {
		counter := newCounter(t)
		name := names(1)[0]
		called := false
		check := globalcounter.ConditionalAddCheck{Name: name, Delta: func(int64) (int64, bool) { called = true; return 1, false }}
		_, err := counter.ConditionalAddMany(t.Context(), globalcounter.ConditionalAddManyParams{
			Checks: []globalcounter.ConditionalAddCheck{check, check},
		})
		require.Error(t, err)
		require.False(t, called)
		require.Zero(t, total(t, counter, name))
	})

	t.Run("calls in opposite name orders complete and keep every count exact", func(t *testing.T) {
		counter := newCounter(t)
		pair := names(2)
		forward := []globalcounter.ConditionalAddCheck{fixed(pair[0], 1), fixed(pair[1], 1)}
		backward := []globalcounter.ConditionalAddCheck{fixed(pair[1], 1), fixed(pair[0], 1)}
		var group sync.WaitGroup
		for _, order := range [][]globalcounter.ConditionalAddCheck{forward, backward, forward, backward} {
			group.Go(func() {
				for range 100 {
					_, err := counter.ConditionalAddMany(t.Context(), globalcounter.ConditionalAddManyParams{Checks: order})
					require.NoError(t, err)
				}
			})
		}
		group.Wait()

		require.Equal(t, int64(400), total(t, counter, pair[0]))
		require.Equal(t, int64(400), total(t, counter, pair[1]))
	})

	t.Run("a rotation after the peer reads makes the decision start over", func(t *testing.T) {
		counter := newCounter(t)
		name := uid.New(uid.RatelimitNamespacePrefix)
		stale := globalcounter.Sequence(counter)
		globalcounter.RotateOnce(counter)

		require.True(t, globalcounter.DecideManyAt(counter, name, stale))
		total, err := counter.Get(t.Context(), globalcounter.GetParams{Name: name})
		require.NoError(t, err)
		require.Zero(t, total, "a refused decision must not change anything")
		require.False(t, globalcounter.DecideManyAt(counter, name, globalcounter.Sequence(counter)))
	})

	t.Run("a change that overflows this writer's own total aborts the whole call", func(t *testing.T) {
		counter := newCounter(t)
		overflowing, other := uid.New(uid.RatelimitNamespacePrefix), uid.New(uid.RatelimitNamespacePrefix)
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: overflowing, Delta: math.MaxInt64}))
		globalcounter.RotateOnce(counter)
		require.NoError(t, (&db.BulkQueries{}).Persist(t.Context(), database.RW(), []db.PersistParams{
			{Name: overflowing, CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: -math.MaxInt64, Revision: 1},
		}))

		one := func(int64) (int64, bool) { return 1, false }
		_, err := counter.ConditionalAddMany(t.Context(), globalcounter.ConditionalAddManyParams{Checks: []globalcounter.ConditionalAddCheck{
			{Name: other, Delta: one}, {Name: overflowing, Delta: one},
		}})
		require.Error(t, err)
		for _, name := range []string{other, overflowing} {
			total, err := counter.Get(t.Context(), globalcounter.GetParams{Name: name})
			require.NoError(t, err)
			require.Zero(t, total, "no check of a rejected call may change: "+name)
		}
	})
}
