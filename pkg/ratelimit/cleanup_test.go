package ratelimit

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// TestPlanCleanupDeletesOnlyRunsThatANewerRowOutgrows guarantees the case
// where the layer 1 idle rule was wrong: the database shows `reset_at ≤ now`,
// but a pod has not written a request yet that raised the old rows'
// candidate. Deleting them would make `reset_at` too small after that write.
func TestPlanCleanupDeletesOnlyRunsThatANewerRowOutgrows(t *testing.T) {
	const nowNs = int64(10 * time.Millisecond)
	dueMs := nowNs/time.Millisecond.Nanoseconds() - 1
	notDueMs := dueMs + 2
	oldRun := []StoredSession{
		{SessionID: "a", AnchorAtNs: 100, ConsumedNs: 10, CompactAfterMs: dueMs},
		{SessionID: "b", AnchorAtNs: 105, ConsumedNs: 5, CompactAfterMs: dueMs},
	}

	cases := []struct {
		name         string
		newerAnchor  int64
		wantInserted []StoredSession
	}{
		{name: "newer row anchored after inner_max", newerAnchor: 115, wantInserted: nil},
		{name: "newer row anchored before inner_max", newerAnchor: 114, wantInserted: []StoredSession{
			{SessionID: "merged", AnchorAtNs: 100, ConsumedNs: 15, CompactAfterMs: nowNs/time.Millisecond.Nanoseconds() + compactAfter.Milliseconds()},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows := append(slices.Clone(oldRun), StoredSession{SessionID: "c", AnchorAtNs: tc.newerAnchor, ConsumedNs: 3, CompactAfterMs: notDueMs})
			plan, err := planCleanup(rows, true, nowNs, func() string { return "merged" })
			require.NoError(t, err)
			require.ElementsMatch(t, oldRun, plan.deleted)
			require.Equal(t, tc.wantInserted, plan.inserted)
		})
	}
}

// FuzzPlanCleanupKeepsResetAt guarantees that cleanup never changes
// `max(reset_at, t)` for any pod time `t ≥ now − maxClockSkew`, which is all
// a decision uses, by the formula over all rows: not right after the plan, and not
// after later writes to rows that are not due or new rows that pods may
// create. It also guarantees that an incomplete read, which misses the oldest
// rows, never deletes a run that those rows count.
func FuzzPlanCleanupKeepsResetAt(f *testing.F) {
	fuzz.Seed(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		nowNs := int64(maxClockSkew+time.Millisecond) + int64(c.Uint16()%1_000)
		nowMs := nowNs / time.Millisecond.Nanoseconds()
		// The pod with the clock furthest behind.
		podNowNs := nowNs - maxClockSkew.Nanoseconds()
		table := map[string]StoredSession{}
		for i := range int(c.Uint8() % 12) {
			compactAfterMs := nowMs + 1
			if c.Bool() {
				compactAfterMs = nowMs - 1
			}
			id := fmt.Sprintf("row_%d", i)
			table[id] = StoredSession{
				SessionID:      id,
				AnchorAtNs:     podNowNs - 500 + int64(c.Uint16()%1_000),
				ConsumedNs:     int64(c.Uint8()),
				CompactAfterMs: compactAfterMs,
			}
		}

		// The order of `ListSessions`, which a read cuts off at its limit.
		newestFirst := slices.SortedFunc(maps.Values(table), func(left, right StoredSession) int {
			return cmp.Or(cmp.Compare(right.AnchorAtNs, left.AnchorAtNs), cmp.Compare(left.SessionID, right.SessionID))
		})
		readCount := len(newestFirst) - int(c.Uint8())%(len(newestFirst)+1)
		before := max(podNowNs, resetAtByFormula(sessionsOf(table)))

		mergedCount := 0
		plan, err := planCleanup(newestFirst[:readCount], readCount == len(newestFirst), nowNs, func() string {
			mergedCount++
			return fmt.Sprintf("merged_%d", mergedCount)
		})
		require.NoError(t, err)
		for _, row := range plan.deleted {
			require.Equal(t, table[row.SessionID], row)
			require.Less(t, row.CompactAfterMs, nowMs, "only due rows are deleted")
			delete(table, row.SessionID)
		}
		for _, row := range plan.inserted {
			require.NotContains(t, table, row.SessionID)
			require.Greater(t, row.CompactAfterMs, nowMs)
			table[row.SessionID] = row
		}
		require.Equal(t, before, max(podNowNs, resetAtByFormula(sessionsOf(table))))

		withoutCleanup := maps.Clone(table)
		for _, row := range plan.inserted {
			delete(withoutCleanup, row.SessionID)
		}
		for _, row := range plan.deleted {
			withoutCleanup[row.SessionID] = row
		}
		for i := range int(c.Uint8() % 4) {
			ids := slices.Sorted(maps.Keys(withoutCleanup))
			if len(ids) > 0 && c.Bool() {
				row := withoutCleanup[ids[int(c.Uint8())%len(ids)]]
				if row.CompactAfterMs > nowMs && !slices.ContainsFunc(plan.inserted, func(s StoredSession) bool { return s.SessionID == row.SessionID }) {
					row.ConsumedNs += int64(c.Uint8())
					withoutCleanup[row.SessionID] = row
					table[row.SessionID] = row
				}
				continue
			}
			newestAnchorNs := podNowNs
			for _, row := range withoutCleanup {
				newestAnchorNs = max(newestAnchorNs, row.AnchorAtNs)
			}
			row := StoredSession{
				SessionID:      fmt.Sprintf("new_%d", i),
				AnchorAtNs:     newestAnchorNs + int64(c.Uint8()),
				ConsumedNs:     int64(c.Uint8()),
				CompactAfterMs: nowMs + 1,
			}
			withoutCleanup[row.SessionID] = row
			table[row.SessionID] = row
		}
		require.Equal(t, max(podNowNs, resetAtByFormula(sessionsOf(withoutCleanup))), max(podNowNs, resetAtByFormula(sessionsOf(table))))
	})
}

func sessionsOf(table map[string]StoredSession) []session {
	sessions := make([]session, 0, len(table))
	for _, row := range table {
		sessions = append(sessions, session{id: row.SessionID, anchorAtNs: row.AnchorAtNs, consumedNs: row.ConsumedNs})
	}
	return sessions
}
