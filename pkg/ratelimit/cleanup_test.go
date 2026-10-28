package ratelimit

import (
	"context"
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fault"
)

// TestDeleteAfterIsWhenTheRowHasFaded guarantees that a row is deleted once
// its level is below `negligibleUnits` for a pod clock `maxClockSkew` behind,
// and never while its period is open.
func TestDeleteAfterIsWhenTheRowHasFaded(t *testing.T) {
	for _, rate := range []Rate{{Limit: 1, Duration: time.Second}, {Limit: 1_000_000, Duration: time.Minute}, {Limit: math.MaxInt64, Duration: maxDuration}} {
		periodNs := periodLengthNs(rate.Duration)
		startNs := periodStartNs(time.Unix(1_700_000_000, 0).UnixNano(), periodNs)
		for _, sum := range []float64{0.001, 1, float64(rate.Limit) * math.Exp(periodDurations)} {
			key := rowKey{periodStartNs: startNs, podID: "pod"}
			deleteNs := deleteAfterNs(startNs, periodNs, rate.Duration, sum)
			require.GreaterOrEqual(t, deleteNs, startNs+periodNs+maxClockSkew.Nanoseconds())
			podNowNs := deleteNs - maxClockSkew.Nanoseconds()
			require.LessOrEqual(t, row{scaledSum: sum}.levelAt(key, podNowNs, rate.Duration), negligibleUnits*(1+1e-9))
		}
	}
}

func seedRows(store *memoryStore, count int, deleteAfterNs int64) {
	rows := make([]StoredRow, count)
	for i := range rows {
		var name [32]byte
		binary.BigEndian.PutUint64(name[:], uint64(i))
		rows[i] = StoredRow{Fingerprint: name, PeriodStartNs: 0, PodID: "pod", ScaledSum: 1, DeleteAfterNs: deleteAfterNs}
	}
	store.seed(rows)
}

func TestCleanUpDeletesOnlyFadedRows(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	nowNs := testClock.Now().UnixNano()
	seedRows(store, 3, nowNs-1)
	kept := StoredRow{Fingerprint: fingerprint("kept"), PeriodStartNs: 0, PodID: "pod", ScaledSum: 1, DeleteAfterNs: nowNs}
	store.seed([]StoredRow{kept})

	require.NoError(t, l.cleanUp(ctx))
	require.Len(t, store.rows, 1)
	require.Equal(t, []StoredRow{kept}, store.storedRows(kept.Fingerprint))
}

func TestCleanUpReportsABacklogAndTheNextPassContinues(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	seedRows(store, rowsPerDelete*maxDeletesPerCleanupPass+1, testClock.Now().UnixNano()-1)

	require.ErrorIs(t, l.cleanUp(ctx), errCleanupBacklog)
	require.Len(t, store.rows, 1)
	require.NoError(t, l.cleanUp(ctx))
	require.Empty(t, store.rows)
}

// TestALateWriteAfterCleanupRestoresTheRow guarantees why D needs no write
// deadline: a write that lands after cleanup deleted its row inserts the
// pod's whole sum, so the table holds what it would have held without the
// delete, and that sum has faded as much as the deleted one.
func TestALateWriteAfterCleanupRestoresTheRow(t *testing.T) {
	ctx := context.Background()
	l, testClock := newTestLimiter(t)
	store := memoryStoreOf(t, l)
	rate := Rate{Limit: 10, Duration: time.Second}
	request := Request{Fingerprint: fingerprint("a"), Rate: rate, Units: 1}
	_, err := l.Allow(ctx, request)
	require.NoError(t, err)
	require.NoError(t, l.flush(ctx))
	written := store.storedRows(request.Fingerprint)

	store.setFailWrites(true)
	_, err = l.Allow(ctx, request)
	require.NoError(t, err)
	require.Error(t, l.flush(ctx))

	testClock.Set(time.Unix(0, written[0].DeleteAfterNs+1))
	require.NoError(t, l.cleanUp(ctx))
	require.Empty(t, store.storedRows(request.Fingerprint))

	store.setFailWrites(false)
	require.NoError(t, l.flush(ctx))
	rows := store.storedRows(request.Fingerprint)
	require.Len(t, rows, 1)
	require.InEpsilon(t, 2*written[0].ScaledSum, rows[0].ScaledSum, 1e-15)
	restored := row{scaledSum: rows[0].ScaledSum}.levelAt(rowKey{periodStartNs: rows[0].PeriodStartNs, podID: l.podID}, testClock.Now().UnixNano(), rate.Duration)
	require.Less(t, restored, negligibleUnits, "the next cleanup may delete it again, it has faded")
}

func TestCleanUpReturnsAFailedStoreClock(t *testing.T) {
	ctx := context.Background()
	l, _ := newTestLimiter(t)
	l.store = failingClockStore{Store: l.store}
	require.Error(t, l.cleanUp(ctx))
}

type failingClockStore struct {
	Store
}

func (failingClockStore) NowNs(context.Context) (int64, error) {
	return 0, fault.New("the clock read failed")
}
