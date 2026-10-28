package ratelimit

import (
	"crypto/sha256"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	shareddb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestMySQLStoreRoundTrip guarantees the write rules of `UpsertRow`: a stale
// write never moves a row back, and a float64 sum survives MySQL bit for
// bit. It also guarantees that `ListPeriods` returns the own row and one
// summed peer row per period, newest first, within its period limit.
func TestMySQLStoreRoundTrip(t *testing.T) {
	ctx := t.Context()
	store := newMySQLTestStore(t)
	nowNs, err := store.NowNs(ctx)
	require.NoError(t, err)
	require.InDelta(t, time.Now().UnixNano(), nowNs, float64(time.Minute))

	fingerprint := sha256.Sum256([]byte(uid.New(uid.TestPrefix)))
	podID := uid.New("", 16)
	deleteAfterNs := nowNs + time.Hour.Nanoseconds()
	newer := StoredRow{Fingerprint: fingerprint, PeriodStartNs: 200, PodID: podID, ScaledSum: math.Pi * math.Exp(9.75), DeleteAfterNs: deleteAfterNs}
	older := StoredRow{Fingerprint: fingerprint, PeriodStartNs: 100, PodID: podID, ScaledSum: 1.0 / 3, DeleteAfterNs: deleteAfterNs}
	require.NoError(t, store.WriteRows(ctx, []StoredRow{newer, older}))
	stale := newer
	stale.ScaledSum, stale.DeleteAfterNs = 1, nowNs
	require.NoError(t, store.WriteRows(ctx, []StoredRow{stale}))

	rows, err := store.ReadRows(ctx, fingerprint, podID, 10)
	require.NoError(t, err)
	require.Equal(t, []StoredRow{newer, older}, rows)

	peerA := StoredRow{Fingerprint: fingerprint, PeriodStartNs: 200, PodID: uid.New("", 16), ScaledSum: 2.5, DeleteAfterNs: deleteAfterNs + 1}
	peerB := StoredRow{Fingerprint: fingerprint, PeriodStartNs: 200, PodID: uid.New("", 16), ScaledSum: 0.25, DeleteAfterNs: deleteAfterNs}
	require.NoError(t, store.WriteRows(ctx, []StoredRow{peerA, peerB}))
	peers := StoredRow{Fingerprint: fingerprint, PeriodStartNs: 200, PodID: "", ScaledSum: 2.75, DeleteAfterNs: deleteAfterNs + 1}
	newer.DeleteAfterNs = deleteAfterNs + 1
	rows, err = store.ReadRows(ctx, fingerprint, podID, 1)
	require.NoError(t, err)
	require.Equal(t, []StoredRow{newer, peers}, rows, "the limit counts periods, not rows")
}

// TestMySQLStoreWritesAFullChunk guarantees that MySQL accepts a write of
// `maxRowsPerWrite` rows, the most that flush sends in one call.
func TestMySQLStoreWritesAFullChunk(t *testing.T) {
	ctx := t.Context()
	store := newMySQLTestStore(t)
	nowNs, err := store.NowNs(ctx)
	require.NoError(t, err)

	fingerprint := sha256.Sum256([]byte(uid.New(uid.TestPrefix)))
	podID := uid.New("", 16)
	rows := make([]StoredRow, maxRowsPerWrite)
	for i := range rows {
		rows[i] = StoredRow{Fingerprint: fingerprint, PeriodStartNs: int64(i+1) * 100, PodID: podID, ScaledSum: 1, DeleteAfterNs: nowNs + time.Hour.Nanoseconds()}
	}
	require.NoError(t, store.WriteRows(ctx, rows))

	read, err := store.ReadRows(ctx, fingerprint, podID, maxRowsPerWrite)
	require.NoError(t, err)
	require.Len(t, read, maxRowsPerWrite)
}

// TestMySQLStoreDeletesFadedRows guarantees that `DeleteFadedRows` deletes
// only rows past their delete time, at most its limit per call.
func TestMySQLStoreDeletesFadedRows(t *testing.T) {
	ctx := t.Context()
	store := newMySQLTestStore(t)
	// The database outlives test runs, so delete times are far in the past,
	// and each run deletes everything before its own cut.
	cutNs := int64(1_000_000) + time.Now().UnixNano()%1_000_000*10
	_, err := store.DeleteFadedRows(ctx, cutNs+10, 1_000_000)
	require.NoError(t, err)

	fingerprint := sha256.Sum256([]byte(uid.New(uid.TestPrefix)))
	podID := uid.New("", 16)
	var rows []StoredRow
	for i := range int64(5) {
		rows = append(rows, StoredRow{Fingerprint: fingerprint, PeriodStartNs: i + 1, PodID: podID, ScaledSum: 1, DeleteAfterNs: cutNs - 1})
	}
	kept := StoredRow{Fingerprint: fingerprint, PeriodStartNs: 100, PodID: podID, ScaledSum: 1, DeleteAfterNs: cutNs}
	require.NoError(t, store.WriteRows(ctx, append(rows, kept)))

	deleted, err := store.DeleteFadedRows(ctx, cutNs, 3)
	require.NoError(t, err)
	require.Equal(t, int64(3), deleted)
	deleted, err = store.DeleteFadedRows(ctx, cutNs, 3)
	require.NoError(t, err)
	require.Equal(t, int64(2), deleted)
	read, err := store.ReadRows(ctx, fingerprint, podID, 10)
	require.NoError(t, err)
	require.Equal(t, []StoredRow{kept}, read)
}

// TestMySQLStoreReturnsDatabaseErrors guarantees that a failed query reaches
// the caller, so flush keeps its rows and cleanup stops its pass.
func TestMySQLStoreReturnsDatabaseErrors(t *testing.T) {
	ctx := t.Context()
	database, err := shareddb.New(shareddb.Config{PrimaryDSN: containers.MySQL(t).DSN, ReadOnlyDSN: "", Tags: sqlcomment.Static{}})
	require.NoError(t, err)
	store := NewMySQLStore(database.RW(), database.RO())
	require.NoError(t, database.Close())

	r := StoredRow{Fingerprint: sha256.Sum256([]byte(uid.New(uid.TestPrefix))), PeriodStartNs: 100, PodID: uid.New("", 16), ScaledSum: 1, DeleteAfterNs: 1}
	_, err = store.NowNs(ctx)
	require.Error(t, err)
	require.Error(t, store.WriteRows(ctx, []StoredRow{r}))
	_, err = store.ReadRows(ctx, r.Fingerprint, r.PodID, 10)
	require.Error(t, err)
	_, err = store.DeleteFadedRows(ctx, 2, 10)
	require.Error(t, err)
}

// newMySQLTestStore returns a store on the test container through `pkg/db`,
// as the API builds it.
func newMySQLTestStore(t testing.TB) Store {
	t.Helper()
	database, err := shareddb.New(shareddb.Config{PrimaryDSN: containers.MySQL(t).DSN, ReadOnlyDSN: "", Tags: sqlcomment.Static{}})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	return NewMySQLStore(database.RW(), database.RO())
}
