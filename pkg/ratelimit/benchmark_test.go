package ratelimit

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand/v2"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/uid"
)

// discardStore is a [Store] that accepts every write and holds nothing, so a
// benchmark measures the limiter alone.
type discardStore struct{}

func (discardStore) NowMs(context.Context) (int64, error)                       { return 0, nil }
func (discardStore) WriteRows(context.Context, []StoredRow) error               { return nil }
func (discardStore) DeleteFadedRows(context.Context, int64, int) (int64, error) { return 0, nil }
func (discardStore) ReadRows(context.Context, [32]byte, string, int) ([]StoredRow, error) {
	return nil, nil
}

// benchRate never denies within a benchmark, so every decision takes the
// path that adds units.
var benchRate = Rate{Limit: 1 << 40, Duration: time.Minute}

// warmLimiter returns a limiter on a clock that stands still, with `keys`
// entries whose peer rows were just read, each holding `peerRows` summed
// peer rows in as many periods. No request reads the store.
func warmLimiter(b *testing.B, store Store, keys, peerRows int) (*limiter, []Request) {
	b.Helper()
	testClock := clock.NewTestClock(time.Unix(1_700_000_000, 0))
	l, err := newLimiter(Config{Clock: testClock, Store: store, Random: rand.New(rand.NewPCG(1, 2))})
	require.NoError(b, err)
	b.Cleanup(func() { require.NoError(b, l.Close()) })
	nowMs := testClock.Now().UnixMilli()
	periodMs := periodLengthMs(benchRate.Duration)
	prefix := uid.New(uid.TestPrefix)
	requests := make([]Request, keys)
	for i := range requests {
		requests[i] = Request{Fingerprint: sha256.Sum256(fmt.Appendf(nil, "%s-%d", prefix, i)), Rate: benchRate, Units: 1}
		e := l.entryOf(requests[i].Fingerprint, benchRate)
		for p := range peerRows {
			e.rows[rowKey{periodStartMs: periodStartMs(nowMs, periodMs) - int64(p)*periodMs, podID: ""}] = row{scaledSum: 1, own: false, committedSum: 0}
		}
		e.levelSum, err = computeLevelSum(e.rows, periodStartMs(nowMs, periodMs), benchRate.Duration)
		require.NoError(b, err)
		e.rowsReadAtMs = nowMs
		e.rowsReadOK = true
	}
	return l, requests
}

// BenchmarkAllow measures one decision on a warm entry, for one key and for
// 1,024 keys decided in parallel.
func BenchmarkAllow(b *testing.B) {
	ctx := context.Background()
	for _, peerRows := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("one key/%d peer rows", peerRows), func(b *testing.B) {
			l, requests := warmLimiter(b, discardStore{}, 1, peerRows)
			b.ReportAllocs()
			for b.Loop() {
				_, err := l.Allow(ctx, requests[0])
				if err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("1024 keys in parallel/%d peer rows", peerRows), func(b *testing.B) {
			l, requests := warmLimiter(b, discardStore{}, 1_024, peerRows)
			var next atomic.Int64
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := int(next.Add(1)) * 97
				for pb.Next() {
					i++
					_, err := l.Allow(ctx, requests[i%len(requests)])
					if err != nil {
						b.Error(err)
						return
					}
				}
			})
		})
	}
}

// BenchmarkFlush measures one flush of `keys` limits that each admitted one
// request since the last flush: collecting the rows, sending them in
// batches of `maxRowsPerWrite`, and marking them committed.
func BenchmarkFlush(b *testing.B) {
	ctx := context.Background()
	stores := []struct {
		name  string
		store func(b *testing.B) Store
	}{
		{"discard store", func(*testing.B) Store { return discardStore{} }},
		{"mysql", func(b *testing.B) Store { return newMySQLTestStore(b) }},
	}
	for _, s := range stores {
		for _, keys := range []int{1_000, 5_000} {
			b.Run(fmt.Sprintf("%s/%d keys", s.name, keys), func(b *testing.B) {
				l, requests := warmLimiter(b, s.store(b), keys, 1)
				b.ReportAllocs()
				for b.Loop() {
					b.StopTimer()
					for _, request := range requests {
						_, err := l.Allow(ctx, request)
						if err != nil {
							b.Fatal(err)
						}
					}
					b.StartTimer()
					err := l.flush(ctx)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// BenchmarkComputeLevelSum measures the merge of a read: the level of N
// rows against the current period.
func BenchmarkComputeLevelSum(b *testing.B) {
	nowMs := time.Unix(1_700_000_000, 0).UnixMilli()
	periodMs := periodLengthMs(benchRate.Duration)
	for _, n := range []int{2, 20, 200} {
		b.Run(fmt.Sprintf("%d rows", n), func(b *testing.B) {
			rows := make(map[rowKey]row, n)
			for i := range n {
				rows[rowKey{periodStartMs: periodStartMs(nowMs, periodMs) - int64(i/2)*periodMs, podID: fmt.Sprint(i % 2)}] = row{scaledSum: float64(i + 1), own: i%2 == 0, committedSum: 0}
			}
			b.ReportAllocs()
			for b.Loop() {
				_, err := computeLevelSum(rows, periodStartMs(nowMs, periodMs), benchRate.Duration)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDeleteFadedRows measures one cleanup batch on MySQL: a bounded
// delete of `rowsPerDelete` faded rows, one per fingerprint.
func BenchmarkDeleteFadedRows(b *testing.B) {
	ctx := context.Background()
	store := newMySQLTestStore(b)
	// Rows of test runs that faded long ago share the table, so each
	// iteration first removes every row before the cut.
	const cutMs = int64(20_000_000_000_000)
	podID := uid.New("", 16)
	b.ReportAllocs()
	for b.Loop() {
		b.StopTimer()
		for {
			deleted, err := store.DeleteFadedRows(ctx, cutMs, rowsPerDelete)
			require.NoError(b, err)
			if deleted < rowsPerDelete {
				break
			}
		}
		prefix := uid.New(uid.TestPrefix)
		rows := make([]StoredRow, rowsPerDelete)
		for i := range rows {
			rows[i] = StoredRow{Fingerprint: sha256.Sum256(fmt.Appendf(nil, "%s-%d", prefix, i)), PeriodStartMs: 1, PodID: podID, ScaledSum: 1, DeleteAfterMs: cutMs - 1}
		}
		for start := 0; start < len(rows); start += maxRowsPerWrite {
			require.NoError(b, store.WriteRows(ctx, rows[start:min(start+maxRowsPerWrite, len(rows))]))
		}
		b.StartTimer()
		deleted, err := store.DeleteFadedRows(ctx, cutMs, rowsPerDelete)
		if err != nil {
			b.Fatal(err)
		}
		if deleted != rowsPerDelete {
			b.Fatalf("deleted %d rows, want %d", deleted, rowsPerDelete)
		}
	}
}
