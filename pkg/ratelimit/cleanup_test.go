package ratelimit

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// closedRow returns a row of `podID` in the bucket at `startNs`, with buckets
// of `widthNs`.
func closedRow(fingerprint [32]byte, startNs, widthNs int64, podID string, consumedNs, resetAtNs int64) StoredRow {
	return StoredRow{
		Fingerprint:    fingerprint,
		BucketStartNs:  startNs,
		PodID:          podID,
		ConsumedNs:     consumedNs,
		ResetAtNs:      resetAtNs,
		CompactAfterNs: compactAfterNs(startNs, widthNs),
	}
}

// TestPlanCleanupMergesClosedBucketsIntoTheBaseRow guarantees the merge of
// `planCleanup` on an example: two closed buckets and an old base row become
// one base row with the same candidate, and the open bucket keeps counting
// it.
func TestPlanCleanupMergesClosedBucketsIntoTheBaseRow(t *testing.T) {
	const widthNs = 100
	lagNs := compactAfterNs(0, widthNs)
	nowNs := 1_000 + lagNs + 1
	base := StoredRow{BucketStartNs: 0, PodID: "", ConsumedNs: 50, ResetAtNs: 980, CompactAfterNs: 980 + lagNs}
	rows := []StoredRow{
		base,
		closedRow([32]byte{}, 900, widthNs, "a", 30, 960),
		closedRow([32]byte{}, 900, widthNs, "b", 20, 950),
		closedRow([32]byte{}, 1_000, widthNs, "a", 40, 1_060),
	}
	plan, err := planCleanup(rows, true, nowNs)
	require.NoError(t, err)
	require.ElementsMatch(t, rows, plan.deleted)
	// Candidates: base 980 + 90 = 1,070; bucket 900 max(900 + 50, 960) + 40
	// = 1,000; bucket 1,000 max(1,000 + 40, 1,060) = 1,060.
	require.Equal(t, []StoredRow{{BucketStartNs: 0, PodID: "", ConsumedNs: 140, ResetAtNs: 1_070, CompactAfterNs: 1_070 + lagNs}}, plan.inserted)
	require.False(t, plan.rowLimit)
}

// TestPlanCleanupDeletesAnIdleLimitOnlyAfterACompleteRead guarantees that the
// base row is dropped only when every bucket that can still be written
// starts after its `reset_at`, and only when the read saw every closed row.
func TestPlanCleanupDeletesAnIdleLimitOnlyAfterACompleteRead(t *testing.T) {
	const widthNs = 100
	lagNs := compactAfterNs(0, widthNs)
	rows := []StoredRow{closedRow([32]byte{}, 900, widthNs, "a", 30, 960), closedRow([32]byte{}, 1_000, widthNs, "a", 30, 1_060)}

	plan, err := planCleanup(rows, true, 1_060+lagNs)
	require.NoError(t, err)
	require.ElementsMatch(t, rows, plan.deleted)
	require.Len(t, plan.inserted, 1, "the base row is not idle yet")

	plan, err = planCleanup(rows, true, 1_060+lagNs+1)
	require.NoError(t, err)
	require.ElementsMatch(t, rows, plan.deleted)
	require.Empty(t, plan.inserted)

	plan, err = planCleanup(rows, false, 1_060+lagNs+1)
	require.NoError(t, err)
	require.Equal(t, rows[:1], plan.deleted, "the newest bucket may be incomplete")
	require.Equal(t, []StoredRow{{BucketStartNs: 0, PodID: "", ConsumedNs: 30, ResetAtNs: 960, CompactAfterNs: 960 + lagNs}}, plan.inserted)

	plan, err = planCleanup(rows[1:], false, 1_060+lagNs+1)
	require.NoError(t, err)
	require.Equal(t, cleanupPlan{deleted: nil, inserted: nil, rowLimit: true}, plan)
}

func TestPlanCleanupRejectsInvalidRows(t *testing.T) {
	const widthNs = 100
	const nowNs = int64(time.Hour)
	valid := closedRow([32]byte{1}, 1_000, widthNs, "a", 10, 1_010)
	with := func(change func(*StoredRow)) StoredRow {
		r := valid
		r.PodID = "b"
		change(&r)
		return r
	}
	cases := []struct {
		name  string
		nowNs int64
		rows  []StoredRow
	}{
		{name: "time at the Unix epoch", nowNs: 0, rows: []StoredRow{valid}},
		{name: "two fingerprints", nowNs: nowNs, rows: []StoredRow{valid, with(func(r *StoredRow) { r.Fingerprint = [32]byte{2} })}},
		{name: "repeated row", nowNs: nowNs, rows: []StoredRow{valid, valid}},
		{name: "bucket row without a pod ID", nowNs: nowNs, rows: []StoredRow{valid, with(func(r *StoredRow) { r.PodID = "" })}},
		{name: "base row with a pod ID", nowNs: nowNs, rows: []StoredRow{valid, with(func(r *StoredRow) { r.BucketStartNs = 0; r.CompactAfterNs = r.ResetAtNs + compactAfterNs(0, widthNs) })}},
		{name: "negative consumed time", nowNs: nowNs, rows: []StoredRow{valid, with(func(r *StoredRow) { r.ConsumedNs = -1 })}},
		{name: "two bucket widths", nowNs: nowNs, rows: []StoredRow{valid, with(func(r *StoredRow) { r.CompactAfterNs++ })}},
		{name: "open row", nowNs: valid.CompactAfterNs, rows: []StoredRow{valid}},
		{name: "consumed sum overflows", nowNs: nowNs, rows: []StoredRow{
			with(func(r *StoredRow) { r.PodID = "c"; r.ConsumedNs = math.MaxInt64/2 + 1 }),
			with(func(r *StoredRow) { r.PodID = "d"; r.ConsumedNs = math.MaxInt64/2 + 1 }),
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := planCleanup(tc.rows, true, tc.nowNs)
			require.Error(t, err)
		})
	}
}

// FuzzPlanCleanupKeepsResetAt guarantees that cleanup never changes
// `max(reset_at, t)` for any pod time `t > now − maxClockSkew`, which is all a
// decision uses: not right after the plan, and not after later writes to open
// buckets or to new buckets. It also guarantees that an incomplete read,
// which misses the newest closed rows, never merges part of a bucket.
func FuzzPlanCleanupKeepsResetAt(f *testing.F) {
	fuzz.Seed(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		widthNs := int64(c.Uint8()%20) + 1
		lagNs := compactAfterNs(0, widthNs)
		nowNs := 10*lagNs + int64(c.Uint16()%1_000)
		podNowNs := nowNs - maxClockSkew.Nanoseconds() + 1
		pods := []string{"a", "b", "c"}

		table := map[rowKey]StoredRow{}
		add := func(startNs int64, pod string, consumedNs int64) {
			key := rowKey{bucketStartNs: startNs, podID: pod}
			r := table[key]
			r.BucketStartNs, r.PodID, r.CompactAfterNs = startNs, pod, compactAfterNs(startNs, widthNs)
			r.ResetAtNs = max(r.ResetAtNs, startNs+int64(c.Uint8())%widthNs) + consumedNs
			r.ConsumedNs += consumedNs
			table[key] = r
		}
		if c.Bool() {
			resetAtNs := nowNs - 2*lagNs + int64(c.Uint16())*4*lagNs/math.MaxUint16
			table[baseRowKey] = StoredRow{ConsumedNs: int64(c.Uint8()) + 1, ResetAtNs: resetAtNs, CompactAfterNs: resetAtNs + lagNs}
		}
		for range int(c.Uint8() % 16) {
			startNs := bucketStartNs(nowNs-lagNs-5*widthNs+int64(c.Uint8()%10)*widthNs, widthNs)
			add(startNs, pods[c.Uint8()%3], int64(c.Uint8()%3)*int64(c.Uint8()))
		}
		before := max(podNowNs, resetAtByFormula(rowsOfTable(table)))

		var closed []StoredRow
		for _, r := range table {
			if r.BucketStartNs == 0 || r.CompactAfterNs < nowNs {
				closed = append(closed, r)
			}
		}
		slices.SortFunc(closed, func(left, right StoredRow) int {
			return cmp.Or(cmp.Compare(left.BucketStartNs, right.BucketStartNs), cmp.Compare(left.PodID, right.PodID))
		})
		readCount := len(closed) - int(c.Uint8())%(len(closed)+1)
		plan, err := planCleanup(closed[:readCount], readCount == len(closed), nowNs)
		require.NoError(t, err)

		after := map[rowKey]StoredRow{}
		for key, r := range table {
			after[key] = r
		}
		for _, r := range plan.deleted {
			key := rowKey{bucketStartNs: r.BucketStartNs, podID: r.PodID}
			require.Equal(t, table[key], r)
			delete(after, key)
		}
		deletedBuckets := map[int64]bool{}
		for _, r := range plan.deleted {
			deletedBuckets[r.BucketStartNs] = true
		}
		for key := range after {
			require.False(t, key.bucketStartNs > 0 && deletedBuckets[key.bucketStartNs], "a bucket is merged whole or not at all")
		}
		for _, r := range plan.inserted {
			require.NotContains(t, after, baseRowKey)
			after[baseRowKey] = r
		}
		require.Equal(t, before, max(podNowNs, resetAtByFormula(rowsOfTable(after))))

		// Later writes go to open buckets: in both tables they raise the
		// same rows by the same amounts.
		for range int(c.Uint8() % 6) {
			startNs := bucketStartNs(nowNs-lagNs+widthNs+int64(c.Uint8()%20)*widthNs, widthNs)
			pod := pods[c.Uint8()%3]
			consumedNs := int64(c.Uint8())
			resetAtNs := startNs + int64(c.Uint8())%widthNs + consumedNs
			for _, rows := range []map[rowKey]StoredRow{table, after} {
				key := rowKey{bucketStartNs: startNs, podID: pod}
				r := rows[key]
				r.BucketStartNs, r.PodID, r.CompactAfterNs = startNs, pod, compactAfterNs(startNs, widthNs)
				r.ConsumedNs += consumedNs
				r.ResetAtNs = max(r.ResetAtNs, resetAtNs)
				rows[key] = r
			}
		}
		require.Equal(t, max(podNowNs, resetAtByFormula(rowsOfTable(table))), max(podNowNs, resetAtByFormula(rowsOfTable(after))))
	})
}

func rowsOfTable(table map[rowKey]StoredRow) map[rowKey]row {
	rows := make(map[rowKey]row, len(table))
	for key, r := range table {
		rows[key] = row{consumedNs: r.ConsumedNs, resetAtNs: r.ResetAtNs, own: false, committedNs: 0, committedResetAtNs: 0}
	}
	return rows
}

func newCleanupTestLimiter(t *testing.T, seed uint64) (*limiter, *memoryStore, *clock.TestClock) {
	t.Helper()
	testClock := clock.NewTestClock(time.Unix(1_700_000_000, 0))
	store := newMemoryStore(testClock)
	l, err := newLimiter(Config{Clock: testClock, Store: store, Random: rand.New(rand.NewPCG(seed, 1))})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, l.Close()) })
	return l, store, testClock
}

// TestCleanUpDeletesEveryIdleFingerprintInOnePass guarantees that one pass
// reaches every due row: past its random start, before it after the wrap,
// and across several pages. 900 fingerprints with 4 closed rows each give
// 3,600 due rows, more than `dueRowsPerPage`, but fewer fingerprints than
// `maxFingerprintsPerCleanupPass`.
func TestCleanUpDeletesEveryIdleFingerprintInOnePass(t *testing.T) {
	const fingerprints = 900
	const rowsPerFingerprint = 4
	widthNs := time.Second.Nanoseconds()
	for seed := range uint64(5) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			l, store, testClock := newCleanupTestLimiter(t, seed)
			// Buckets spread over 20 minutes and end 50 minutes ago.
			oldestNs := bucketStartNs(testClock.Now().Add(-70*time.Minute).UnixNano(), widthNs)
			var rows []StoredRow
			for f := range fingerprints {
				for r := range rowsPerFingerprint {
					startNs := oldestNs + int64(f*rowsPerFingerprint+r)*widthNs
					rows = append(rows, closedRow(fingerprint(fmt.Sprintf("idle %d", f)), startNs, widthNs, "pod", time.Millisecond.Nanoseconds(), startNs+time.Millisecond.Nanoseconds()))
				}
			}
			store.seed(rows)

			require.NoError(t, l.cleanUp(context.Background()))
			for f := range fingerprints {
				require.Empty(t, store.storedRows(fingerprint(fmt.Sprintf("idle %d", f))), "fingerprint %d", f)
			}
		})
	}
}

// TestCleanUpReportsABacklogAndTheNextPassContinues guarantees that a pass
// that reaches `maxFingerprintsPerCleanupPass` reports the backlog, so the
// next pass starts soon, and that the next pass cleans the rest.
func TestCleanUpReportsABacklogAndTheNextPassContinues(t *testing.T) {
	l, store, testClock := newCleanupTestLimiter(t, 1)
	widthNs := time.Second.Nanoseconds()
	startNs := bucketStartNs(testClock.Now().Add(-time.Hour).UnixNano(), widthNs)
	idle := func(i int) [32]byte { return fingerprint(fmt.Sprintf("idle %d", i)) }
	var rows []StoredRow
	for i := range maxFingerprintsPerCleanupPass + 1 {
		rows = append(rows, closedRow(idle(i), startNs, widthNs, "pod", 1, startNs+1))
	}
	store.seed(rows)
	remaining := func() int {
		count := 0
		for i := range maxFingerprintsPerCleanupPass + 1 {
			if len(store.storedRows(idle(i))) > 0 {
				count++
			}
		}
		return count
	}

	require.ErrorIs(t, l.cleanUp(context.Background()), errCleanupBacklog)
	require.Equal(t, 1, remaining())
	require.NoError(t, l.cleanUp(context.Background()))
	require.Zero(t, remaining())
}

func TestCleanUpReturnsAFailedPrimaryRead(t *testing.T) {
	l, store, testClock := newCleanupTestLimiter(t, 1)
	widthNs := time.Second.Nanoseconds()
	startNs := bucketStartNs(testClock.Now().Add(-time.Hour).UnixNano(), widthNs)
	store.seed([]StoredRow{closedRow(fingerprint("idle"), startNs, widthNs, "pod", 1, startNs+1)})
	store.setFailReads(true)

	require.Error(t, l.cleanUp(context.Background()))
}

// TestCleanUpPagesPastAFingerprintWithAFullPage guarantees that a pass moves
// past a full page of one fingerprint. The busy fingerprint has
// `dueRowsPerPage + 1` closed rows in one bucket, and the idle fingerprint
// sorts after it with the same `CompactAfterNs`, so only the second page
// reaches it.
func TestCleanUpPagesPastAFingerprintWithAFullPage(t *testing.T) {
	l, store, testClock := newCleanupTestLimiter(t, 1)
	busy := [32]byte{1}
	idle := [32]byte{2}
	widthNs := (time.Hour / 1000).Nanoseconds()
	startNs := bucketStartNs(testClock.Now().Add(-10*time.Minute).UnixNano(), widthNs)
	var rows []StoredRow
	for i := range dueRowsPerPage + 1 {
		rows = append(rows, closedRow(busy, startNs, widthNs, fmt.Sprintf("pod %04d", i), time.Second.Nanoseconds(), startNs+time.Second.Nanoseconds()))
	}
	rows = append(rows, closedRow(idle, startNs, widthNs, "pod", 1, startNs+1))
	store.seed(rows)

	require.NoError(t, l.cleanUp(context.Background()))
	wantResetAtNs := startNs + int64(dueRowsPerPage+1)*time.Second.Nanoseconds()
	require.Equal(t, []StoredRow{{
		Fingerprint:    busy,
		BucketStartNs:  0,
		PodID:          "",
		ConsumedNs:     int64(dueRowsPerPage+1) * time.Second.Nanoseconds(),
		ResetAtNs:      wantResetAtNs,
		CompactAfterNs: wantResetAtNs + compactAfterNs(0, widthNs),
	}}, store.storedRows(busy))
	require.Empty(t, store.storedRows(idle))
}

// TestCleanUpReportsABucketBeyondTheReadLimit guarantees that a bucket with
// more rows than a cleanup read returns is left alone and reported, and that
// the pass still cleans the other fingerprints.
func TestCleanUpReportsABucketBeyondTheReadLimit(t *testing.T) {
	l, store, testClock := newCleanupTestLimiter(t, 1)
	crowded := [32]byte{1}
	idle := [32]byte{2}
	widthNs := time.Second.Nanoseconds()
	startNs := bucketStartNs(testClock.Now().Add(-10*time.Minute).UnixNano(), widthNs)
	var rows []StoredRow
	for i := range maxRowsPerCleanupRead {
		rows = append(rows, closedRow(crowded, startNs, widthNs, fmt.Sprintf("pod %04d", i), 1, startNs+1))
	}
	rows = append(rows, closedRow(idle, startNs+widthNs, widthNs, "pod", 1, startNs+widthNs+1))
	store.seed(rows)

	require.Error(t, l.cleanUp(context.Background()))
	require.Len(t, store.storedRows(crowded), maxRowsPerCleanupRead)
	require.Empty(t, store.storedRows(idle))
}
