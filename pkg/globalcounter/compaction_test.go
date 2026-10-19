package globalcounter_test

import (
	"database/sql"
	"math"
	"slices"
	"strings"
	"sync"
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

func TestCounter_Compaction(t *testing.T) {
	config := containers.MySQLIsolated(t)
	database, err := mysql.New(mysql.Config{PrimaryDSN: config.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	const (
		compactedPrefix = "compacted_"
		// ready makes a seeded row compactable at once, and later makes it wait.
		ready = time.Duration(0)
		later = time.Hour
	)
	newCounter := func(t *testing.T) *globalcounter.Counter {
		t.Helper()
		counter, err := globalcounter.New(globalcounter.Config{DB: database, Clock: clock.NewTestClock()})
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)
		return counter
	}
	// deleteAfter is a deletion time after every clock these tests use, so
	// eviction leaves the rows alone.
	deleteAfter := func(hours int64) sql.NullInt64 {
		return sql.NullInt64{Int64: time.Now().Add(time.Duration(hours+24) * time.Hour).UnixMilli(), Valid: true}
	}
	seedMany := func(t *testing.T, rows []db.PersistParams) {
		t.Helper()
		for chunk := range slices.Chunk(rows, 1_000) {
			require.NoError(t, (&db.BulkQueries{}).Persist(t.Context(), database.RW(), chunk))
		}
	}
	row := func(name string, count int64, delete sql.NullInt64, wait time.Duration) db.PersistParams {
		return db.PersistParams{Name: name, WriterID: uid.New("", 16), Count: count, Revision: 1, DeleteAfter: delete, CompactDelayMs: wait.Milliseconds()}
	}
	seed := func(t *testing.T, name string, count int64, delete sql.NullInt64, wait time.Duration) {
		t.Helper()
		seedMany(t, []db.PersistParams{row(name, count, delete, wait)})
	}
	// seedAtLevel seeds a row that is ready and sits at a compaction level, as a
	// row does that earlier passes found nothing to merge with.
	seedAtLevel := func(t *testing.T, name string, count int64, level uint8) {
		t.Helper()
		seeded := row(name, count, deleteAfter(1), ready)
		seedMany(t, []db.PersistParams{seeded})
		require.NoError(t, db.Query.DeferCompaction(t.Context(), database.RW(), db.DeferCompactionParams{
			Name: name, CompactionLevel: level, DelayMs: 0, WriterIds: []string{seeded.WriterID},
		}))
	}
	rows := func(t *testing.T, name string) []db.ListRowsRow {
		t.Helper()
		rows, err := db.Query.ListRows(t.Context(), database.RW(), db.ListRowsParams{Name: name})
		require.NoError(t, err)
		return rows
	}
	compactedRows := func(t *testing.T, name string) []db.ListRowsRow {
		t.Helper()
		var compacted []db.ListRowsRow
		for _, row := range rows(t, name) {
			if strings.HasPrefix(row.WriterID, compactedPrefix) {
				compacted = append(compacted, row)
			}
		}
		return compacted
	}
	total := func(t *testing.T, name string) int64 {
		t.Helper()
		sum, err := db.Query.Read(t.Context(), database.RW(), db.ReadParams{Name: name, CurrentWriterID: "", PreviousWriterID: ""})
		require.NoError(t, err)
		return sum
	}
	// waitsAbout checks that compactionAfter is the database time plus wait, within a
	// minute.
	waitsAbout := func(t *testing.T, compactionAfter int64, wait time.Duration) {
		t.Helper()
		require.InDelta(t, time.Now().Add(wait).UnixMilli(), compactionAfter, float64(time.Minute.Milliseconds()))
	}
	newName := func() string { return uid.New(uid.RatelimitNamespacePrefix) }

	t.Run("rows that are ready merge into one new row, keeping the sum and the latest deletion time", func(t *testing.T) {
		counter := newCounter(t)
		name := newName()
		seed(t, name, 10, deleteAfter(1), ready)
		seed(t, name, -3, deleteAfter(9), ready)
		seed(t, name, 5, deleteAfter(4), ready)
		seed(t, name, 7, deleteAfter(2), later)

		globalcounter.CompactOnce(t.Context(), counter)

		require.Len(t, rows(t, name), 2, "the compacted row and the one row that must still wait")
		compacted := compactedRows(t, name)
		require.Len(t, compacted, 1)
		require.Equal(t, int64(12), compacted[0].Count)
		require.True(t, compacted[0].DeleteAfter.Valid)
		require.InDelta(t, deleteAfter(9).Int64, compacted[0].DeleteAfter.Int64, float64(time.Minute.Milliseconds()), "the latest deletion time wins")
		require.Equal(t, uint8(1), compacted[0].CompactionLevel, "level 0 is for rows that writers wrote")
		waitsAbout(t, compacted[0].CompactionAfter, 20*time.Minute)
		require.Equal(t, int64(19), total(t, name), "compaction must not change the total")
	})

	t.Run("a merged row takes the lowest level of its sources", func(t *testing.T) {
		name := newName()
		seedAtLevel(t, name, 1, 6)
		seedAtLevel(t, name, 2, 3)
		seedAtLevel(t, name, 4, 5)
		// The Counter compacts once when it starts, so it must not exist while the
		// rows are only partly set up.
		counter := newCounter(t)

		globalcounter.CompactOnce(t.Context(), counter)

		compacted := compactedRows(t, name)
		require.Len(t, compacted, 1)
		require.Len(t, rows(t, name), 1)
		require.Equal(t, int64(7), compacted[0].Count)
		require.Equal(t, uint8(3), compacted[0].CompactionLevel, "a counter that still gets rows stays at a low level")
		waitsAbout(t, compacted[0].CompactionAfter, 80*time.Minute)
	})

	t.Run("a row without a deletion time keeps the merged row without one", func(t *testing.T) {
		counter := newCounter(t)
		name := newName()
		seed(t, name, 4, sql.NullInt64{}, ready)
		seed(t, name, 6, deleteAfter(3), ready)

		globalcounter.CompactOnce(t.Context(), counter)

		compacted := compactedRows(t, name)
		require.Len(t, compacted, 1)
		require.Equal(t, int64(10), compacted[0].Count)
		require.False(t, compacted[0].DeleteAfter.Valid, "NULL must win, or the merged counter would be deleted early")
	})

	t.Run("a single ready row is left alone, moves up a level and waits twice as long", func(t *testing.T) {
		counter := newCounter(t)
		name := newName()
		seed(t, name, 4, deleteAfter(1), ready)
		before := rows(t, name)[0]

		globalcounter.CompactOnce(t.Context(), counter)

		after := rows(t, name)
		require.Len(t, after, 1)
		require.Equal(t, before.WriterID, after[0].WriterID, "there is nothing to merge it with, so it is not rewritten")
		require.Equal(t, before.Revision, after[0].Revision)
		require.Equal(t, int64(4), after[0].Count)
		require.Equal(t, uint8(1), after[0].CompactionLevel)
		waitsAbout(t, after[0].CompactionAfter, 20*time.Minute)

		globalcounter.CompactOnce(t.Context(), counter)
		require.Equal(t, after, rows(t, name), "a second pass finds nothing to do")
	})

	t.Run("the wait doubles with each level up to a year, and the level stops at 255", func(t *testing.T) {
		for _, tt := range []struct {
			level     uint8
			wantLevel uint8
			wait      time.Duration
		}{
			{level: 4, wantLevel: 5, wait: 320 * time.Minute},
			{level: 14, wantLevel: 15, wait: 327_680 * time.Minute},
			{level: 15, wantLevel: 16, wait: 365 * 24 * time.Hour},
			{level: 100, wantLevel: 101, wait: 365 * 24 * time.Hour},
			{level: math.MaxUint8, wantLevel: math.MaxUint8, wait: 365 * 24 * time.Hour},
		} {
			name := newName()
			seedAtLevel(t, name, 1, tt.level)
			counter := newCounter(t)

			globalcounter.CompactOnce(t.Context(), counter)

			got := rows(t, name)
			require.Len(t, got, 1)
			require.Equal(t, tt.wantLevel, got[0].CompactionLevel, "level %d", tt.level)
			waitsAbout(t, got[0].CompactionAfter, tt.wait)
		}
	})

	t.Run("a new row merges with the rows that earlier passes moved away", func(t *testing.T) {
		name := newName()
		// A compacted row that waits for a year, and a lone row at a high level.
		seedAtLevel(t, name, 5, 9)
		parked := rows(t, name)[0]
		require.NoError(t, db.Query.DeferCompaction(t.Context(), database.RW(), db.DeferCompactionParams{
			Name: name, CompactionLevel: 9, DelayMs: (365 * 24 * time.Hour).Milliseconds(), WriterIds: []string{parked.WriterID},
		}))
		seed(t, name, 3, deleteAfter(1), ready)
		counter := newCounter(t)

		globalcounter.CompactOnce(t.Context(), counter)

		got := rows(t, name)
		require.Len(t, got, 1, "the new row did not wait for the parked row's year")
		require.Equal(t, int64(8), got[0].Count)
		require.Equal(t, uint8(1), got[0].CompactionLevel, "the counter is written again, so it is back at the lowest level compaction writes")
		require.Equal(t, int64(8), total(t, name))
	})

	t.Run("rows that become ready later merge with an earlier compacted row", func(t *testing.T) {
		counter := newCounter(t)
		name := newName()
		seed(t, name, 4, deleteAfter(1), ready)
		seed(t, name, 6, deleteAfter(1), ready)
		globalcounter.CompactOnce(t.Context(), counter)
		require.Len(t, compactedRows(t, name), 1)

		seed(t, name, 1, deleteAfter(1), ready)
		globalcounter.CompactOnce(t.Context(), counter)

		require.Len(t, rows(t, name), 1)
		require.Equal(t, int64(11), total(t, name))
	})

	t.Run("a row written through a Counter waits for the compaction delay", func(t *testing.T) {
		counter := newCounter(t)
		name := newName()
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: 5, DeleteAfter: time.Time{}}))
		require.Eventually(t, func() bool { return total(t, name) == 5 }, 10*time.Second, 20*time.Millisecond)

		globalcounter.CompactOnce(t.Context(), counter)

		got := rows(t, name)
		require.Len(t, got, 1)
		require.False(t, strings.HasPrefix(got[0].WriterID, compactedPrefix), "a row written a moment ago is not final")
		require.Zero(t, got[0].CompactionLevel)
		require.Greater(t, got[0].CompactionAfter, time.Now().Add(59*time.Minute).UnixMilli())
	})

	t.Run("an older snapshot does not move the compaction time", func(t *testing.T) {
		name := newName()
		fresh := row(name, 5, deleteAfter(1), later)
		fresh.Revision = 4
		seedMany(t, []db.PersistParams{fresh})
		stale := fresh
		stale.Revision = 3
		stale.CompactDelayMs = (10 * later).Milliseconds()
		before := rows(t, name)[0].CompactionAfter
		seedMany(t, []db.PersistParams{stale})
		require.Equal(t, before, rows(t, name)[0].CompactionAfter, "a snapshot MySQL ignores must not postpone compaction")
		newer := fresh
		newer.Revision = 5
		newer.CompactDelayMs = (10 * later).Milliseconds()
		seedMany(t, []db.PersistParams{newer})
		require.Greater(t, rows(t, name)[0].CompactionAfter, before, "an accepted snapshot restarts the wait")
	})

	t.Run("many writers converge into one row without changing the total", func(t *testing.T) {
		counter := newCounter(t)
		name := newName()
		var want int64
		var seeded []db.PersistParams
		for writer := range 900 {
			count := int64(writer%7) - 3
			want += count
			seeded = append(seeded, row(name, count, deleteAfter(1), ready))
		}
		seedMany(t, seeded)
		require.Len(t, rows(t, name), 900)

		globalcounter.CompactOnce(t.Context(), counter)

		require.Len(t, rows(t, name), 1, "more rows than one transaction locks: the next transaction merges the rest into the first result")
		require.Equal(t, want, total(t, name))
	})

	t.Run("a pass handles a bounded number of rows even for a single name", func(t *testing.T) {
		counter := newCounter(t)
		name := newName()
		seeded := make([]db.PersistParams, 5_200)
		for i := range seeded {
			seeded[i] = row(name, 1, deleteAfter(1), ready)
		}
		seedMany(t, seeded)

		globalcounter.CompactOnce(t.Context(), counter)
		require.Greater(t, len(rows(t, name)), 200, "one pass handles at most 5000 rows, so at least 200 of them are left, next to the compacted row")
		require.Less(t, len(rows(t, name)), 400, "and it handled most of them")
		require.Equal(t, int64(5_200), total(t, name))

		globalcounter.CompactOnce(t.Context(), counter)
		require.Len(t, rows(t, name), 1)
		require.Equal(t, int64(5_200), total(t, name))
	})

	t.Run("a pass handles at most 5000 rows even when its names also have parked rows", func(t *testing.T) {
		// Six names with 499 ready and 499 parked rows each, 5,988 rows in all. If
		// the ready and the parked rows each had their own limit, one pass would
		// lock almost all of them.
		var names []string
		original := map[string]struct{}{}
		for range 6 {
			name := newName()
			names = append(names, name)
			var seeded []db.PersistParams
			var parked []string
			for i := range 998 {
				seeded = append(seeded, row(name, 1, deleteAfter(1), ready))
				original[seeded[i].WriterID] = struct{}{}
				if i < 499 {
					parked = append(parked, seeded[i].WriterID)
				}
			}
			seedMany(t, seeded)
			require.NoError(t, db.Query.DeferCompaction(t.Context(), database.RW(), db.DeferCompactionParams{
				Name: name, CompactionLevel: 1, DelayMs: time.Hour.Milliseconds(), WriterIds: parked,
			}))
		}
		counter := newCounter(t)

		globalcounter.CompactOnce(t.Context(), counter)

		untouched := 0
		for _, name := range names {
			for _, row := range rows(t, name) {
				if _, ok := original[row.WriterID]; ok {
					untouched++
				}
			}
		}
		require.GreaterOrEqual(t, untouched, 5_988-5_000, "a pass handles at most 5000 of the rows, so the rest is untouched")
	})

	t.Run("compacting again changes nothing", func(t *testing.T) {
		counter := newCounter(t)
		name := newName()
		seed(t, name, 4, deleteAfter(1), ready)
		seed(t, name, 6, deleteAfter(1), ready)

		globalcounter.CompactOnce(t.Context(), counter)
		first := rows(t, name)
		globalcounter.CompactOnce(t.Context(), counter)
		globalcounter.CompactOnce(t.Context(), counter)

		require.Equal(t, first, rows(t, name), "a transaction repeated after an unknown outcome must find nothing left to merge")
		require.Equal(t, int64(10), total(t, name))
	})

	t.Run("a sum that overflows is left alone and does not hold up other names", func(t *testing.T) {
		counter := newCounter(t)
		poisoned, healthy := newName(), newName()
		// More overflowing rows than one transaction locks, and older than the
		// healthy rows, so a pass that did not move them away would never get past
		// them.
		var seeded []db.PersistParams
		for range 600 {
			seeded = append(seeded, row(poisoned, math.MaxInt64, deleteAfter(1), -time.Minute))
		}
		seedMany(t, seeded)
		seed(t, healthy, 3, deleteAfter(1), ready)
		seed(t, healthy, 4, deleteAfter(1), ready)

		globalcounter.CompactOnce(t.Context(), counter)

		require.Len(t, rows(t, poisoned), 600, "rows whose merge would overflow stay as they are")
		for _, row := range rows(t, poisoned) {
			require.Equal(t, uint8(1), row.CompactionLevel, "they move up a level, so no later pass selects them at once")
			waitsAbout(t, row.CompactionAfter, 20*time.Minute)
		}
		compacted := compactedRows(t, healthy)
		require.Len(t, compacted, 1)
		require.Equal(t, int64(7), compacted[0].Count)
		require.Len(t, rows(t, healthy), 1)
	})

	t.Run("compactors running at the same time count every row once", func(t *testing.T) {
		compactors := make([]*globalcounter.Counter, 3)
		for i := range compactors {
			compactors[i] = newCounter(t)
		}
		sums := map[string]int64{}
		var seeded []db.PersistParams
		for range 30 {
			name := newName()
			for i := range 6 {
				seeded = append(seeded, row(name, int64(i+1), deleteAfter(1), ready))
				sums[name] += int64(i + 1)
			}
		}
		seedMany(t, seeded)

		var group sync.WaitGroup
		for _, compactor := range compactors {
			group.Go(func() {
				for range 3 {
					globalcounter.CompactOnce(t.Context(), compactor)
				}
			})
		}
		group.Wait()

		for name, want := range sums {
			require.Equal(t, want, total(t, name), name)
			require.Less(t, len(rows(t, name)), 6, "every name was compacted at least in part: "+name)
		}
	})

	t.Run("a writer's own total survives its old writer id being compacted", func(t *testing.T) {
		counter, err := globalcounter.NewWithCompactDelay(globalcounter.Config{DB: database, Clock: clock.NewTestClock()}, ready)
		require.NoError(t, err)
		globalcounter.Cleanup(t, counter)
		name := newName()
		require.NoError(t, counter.Add(globalcounter.AddParams{Name: name, Delta: 10, DeleteAfter: time.Time{}}))
		require.Eventually(t, func() bool { return total(t, name) == 10 }, 10*time.Second, 20*time.Millisecond)

		get := func() int64 {
			count, err := counter.Get(t.Context(), globalcounter.GetParams{Name: name})
			require.NoError(t, err)
			return count
		}
		globalcounter.RotateOnce(counter)
		globalcounter.RotateOnce(counter)
		globalcounter.EvictOnce(t.Context(), counter)
		require.Equal(t, int64(10), get(), "its old writer id is a stored row by now")
		globalcounter.CompactOnce(t.Context(), counter)
		require.Equal(t, int64(10), get(), "and compaction leaves the total as it is")
		require.Len(t, rows(t, name), 1)
	})
}
