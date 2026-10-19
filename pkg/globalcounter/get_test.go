package globalcounter_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/globalcounter"
	"github.com/unkeyed/unkey/pkg/globalcounter/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCounter_Get(t *testing.T) {
	database, _ := newDatabase(t)
	bulk := &db.BulkQueries{}

	t.Run("local changes merge with peers and become visible to another writer", func(t *testing.T) {
		testClock := clock.NewTestClock()
		counter, err := globalcounter.New(globalcounter.Config{
			DB: database, Clock: testClock,
		})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)
		observer, err := globalcounter.New(globalcounter.Config{
			DB: database, Clock: testClock,
		})
		require.NoError(t, err)
		globalcounter.Cleanup(t, observer)
		name := uid.New(uid.RatelimitNamespacePrefix)
		peer := db.PersistParams{Name: name, CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: 100, Revision: 1}
		require.NoError(t, bulk.Persist(t.Context(), database.RW(), []db.PersistParams{peer}))
		params := globalcounter.GetParams{Name: name}
		count, err := counter.Get(t.Context(), params)
		require.NoError(t, err)
		require.Equal(t, int64(100), count)

		mutation := globalcounter.AddParams{Name: name, Delta: 30, DeleteAfter: testClock.Now().Add(time.Hour)}
		require.NoError(t, counter.Add(mutation))
		count, err = counter.Get(t.Context(), params)
		require.NoError(t, err)
		require.Equal(t, int64(130), count)

		peer.Count = 120
		peer.Revision++
		require.NoError(t, bulk.Persist(t.Context(), database.RW(), []db.PersistParams{peer}))
		testClock.Tick(5 * time.Second)
		count, err = counter.Get(t.Context(), params)
		require.NoError(t, err)
		require.Equal(t, int64(150), count)
		mutation.Delta = -45
		require.NoError(t, counter.Add(mutation))
		count, err = counter.Get(t.Context(), params)
		require.NoError(t, err)
		require.Equal(t, int64(105), count)

		require.Eventually(t, func() bool {
			testClock.Tick(5 * time.Second)
			count, err := observer.Get(t.Context(), params)
			return err == nil && count == 105
		}, 5*time.Second, 10*time.Millisecond)
		count, err = counter.Get(t.Context(), params)
		require.NoError(t, err)
		require.Equal(t, int64(105), count)
	})

	t.Run("combined local and peer counts reject overflow", func(t *testing.T) {
		for _, tt := range []struct {
			name  string
			peer  int64
			delta int64
		}{
			{name: "upper", peer: math.MaxInt64, delta: 1},
			{name: "lower", peer: math.MinInt64, delta: -1},
		} {
			t.Run(tt.name, func(t *testing.T) {
				counter, err := globalcounter.New(globalcounter.Config{
					DB: database, Clock: clock.NewTestClock(),
				})
				require.NoError(t, err)
				globalcounter.Cleanup(t, counter)
				name := uid.New(uid.RatelimitNamespacePrefix)
				require.NoError(t, bulk.Persist(t.Context(), database.RW(), []db.PersistParams{
					{Name: name, CompactDelayMs: testCompactDelayMs, WriterID: uid.New("", 16), Count: tt.peer, Revision: 1},
				}))
				require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: tt.delta}))
				overflows := getsTotal(t, "overflow")
				count, err := counter.Get(t.Context(), globalcounter.GetParams{Name: name})
				require.Error(t, err)
				require.Zero(t, count)
				require.Equal(t, 1.0, getsTotal(t, "overflow")-overflows)
				require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: -tt.delta}))
				count, err = counter.Get(t.Context(), globalcounter.GetParams{Name: name})
				require.NoError(t, err)
				require.Equal(t, tt.peer, count)
			})
		}
	})
}
