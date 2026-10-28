package ratelimit

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	shareddb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestMySQLStoreRoundTrip guarantees the write rules of `UpsertRow`: a stale
// write never moves a row back, and MySQL refuses writes to a closed bucket
// by its own clock. It also guarantees that `ListBuckets` returns the own row
// and one summed peer row per bucket, newest first, within its bucket limit.
func TestMySQLStoreRoundTrip(t *testing.T) {
	ctx := t.Context()
	store := newMySQLTestStore(t)
	nowNs, err := store.NowNs(ctx)
	require.NoError(t, err)
	require.InDelta(t, time.Now().UnixNano(), nowNs, float64(time.Minute))

	fingerprint := sha256.Sum256([]byte(uid.New(uid.TestPrefix)))
	podID := uid.New("", 16)
	open := StoredRow{Fingerprint: fingerprint, BucketStartNs: 200, PodID: podID, ConsumedNs: 7, ResetAtNs: 210, CompactAfterNs: nowNs + time.Hour.Nanoseconds()}
	closed := StoredRow{Fingerprint: fingerprint, BucketStartNs: 100, PodID: podID, ConsumedNs: 5, ResetAtNs: 105, CompactAfterNs: nowNs - time.Hour.Nanoseconds()}
	require.NoError(t, store.WriteRows(ctx, []StoredRow{open, closed}))
	stale := open
	stale.ConsumedNs, stale.ResetAtNs = 3, 203
	grown := closed
	grown.ConsumedNs, grown.ResetAtNs = 9, 109
	require.NoError(t, store.WriteRows(ctx, []StoredRow{stale, grown}))

	rows, err := store.ReadRows(ctx, fingerprint, podID, 10)
	require.NoError(t, err)
	require.Equal(t, []StoredRow{open}, rows, "a refused insert stores zero consumed time, which reads leave out")

	peerA := StoredRow{Fingerprint: fingerprint, BucketStartNs: 200, PodID: uid.New("", 16), ConsumedNs: 2, ResetAtNs: 230, CompactAfterNs: open.CompactAfterNs}
	peerB := StoredRow{Fingerprint: fingerprint, BucketStartNs: 200, PodID: uid.New("", 16), ConsumedNs: 3, ResetAtNs: 220, CompactAfterNs: open.CompactAfterNs}
	older := StoredRow{Fingerprint: fingerprint, BucketStartNs: 150, PodID: peerA.PodID, ConsumedNs: 4, ResetAtNs: 154, CompactAfterNs: open.CompactAfterNs}
	require.NoError(t, store.WriteRows(ctx, []StoredRow{peerA, peerB, older}))
	peers := StoredRow{Fingerprint: fingerprint, BucketStartNs: 200, PodID: "", ConsumedNs: 5, ResetAtNs: 230, CompactAfterNs: open.CompactAfterNs}
	olderPeers := older
	olderPeers.PodID = ""
	rows, err = store.ReadRows(ctx, fingerprint, podID, 10)
	require.NoError(t, err)
	require.Equal(t, []StoredRow{open, peers, olderPeers}, rows)

	refused := StoredRow{Fingerprint: fingerprint, BucketStartNs: 300, PodID: podID, ConsumedNs: 1, ResetAtNs: 301, CompactAfterNs: closed.CompactAfterNs}
	require.NoError(t, store.WriteRows(ctx, []StoredRow{refused}))
	rows, err = store.ReadRows(ctx, fingerprint, podID, 1)
	require.NoError(t, err)
	require.Equal(t, []StoredRow{open, peers}, rows, "the limit counts buckets with consumed time, not rows")
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
		startNs := int64(i+1) * 100
		rows[i] = StoredRow{Fingerprint: fingerprint, BucketStartNs: startNs, PodID: podID, ConsumedNs: 1, ResetAtNs: startNs + 1, CompactAfterNs: nowNs + time.Hour.Nanoseconds()}
	}
	require.NoError(t, store.WriteRows(ctx, rows))

	read, err := store.ReadRows(ctx, fingerprint, podID, maxRowsPerWrite)
	require.NoError(t, err)
	require.Len(t, read, maxRowsPerWrite)
}

// TestMySQLStoreCleanup guarantees the promises that cleanup relies on: due
// rows page in `ListRowsToCompact` order, closed rows come oldest first with
// the base row, and a cleanup whose rows changed after it read them changes
// nothing.
func TestMySQLStoreCleanup(t *testing.T) {
	ctx := t.Context()
	store := newMySQLTestStore(t)

	// The database outlives test runs, so each run lists its own range.
	dueNs := time.Now().Add(time.Hour).UnixNano()
	beforeDue := DueRow{Fingerprint: lastFingerprint, CompactAfterNs: dueNs - 1}
	// `first` sorts before `second`, and `first[0] > first[1]`, so a cursor
	// that lost its first byte would sort before `first` and list it again.
	first := sha256.Sum256([]byte(uid.New(uid.TestPrefix)))
	first[0], first[1] = 1, 0
	second := sha256.Sum256([]byte(uid.New(uid.TestPrefix)))
	second[0] = 2
	// CompactAfterNs is an hour ahead, so MySQL accepts the writes.
	a := StoredRow{Fingerprint: second, BucketStartNs: 100, PodID: "a", ConsumedNs: 10, ResetAtNs: 110, CompactAfterNs: dueNs}
	b := StoredRow{Fingerprint: second, BucketStartNs: 100, PodID: "b", ConsumedNs: 5, ResetAtNs: 104, CompactAfterNs: dueNs}
	c := StoredRow{Fingerprint: first, BucketStartNs: 100, PodID: "a", ConsumedNs: 1, ResetAtNs: 101, CompactAfterNs: dueNs}
	require.NoError(t, store.WriteRows(ctx, []StoredRow{a, b, c}))

	page, err := store.ListDueRows(ctx, dueNs+1, beforeDue, 1)
	require.NoError(t, err)
	require.Equal(t, []DueRow{{Fingerprint: first, CompactAfterNs: dueNs}}, page)
	page, err = store.ListDueRows(ctx, dueNs+1, page[0], 3)
	require.NoError(t, err)
	require.Equal(t, []DueRow{{Fingerprint: second, CompactAfterNs: dueNs}, {Fingerprint: second, CompactAfterNs: dueNs}}, page)
	page, err = store.ListDueRows(ctx, dueNs+1, page[1], 3)
	require.NoError(t, err)
	require.Empty(t, page, "the next page starts after the last fingerprint")
	page, err = store.ListDueRows(ctx, dueNs, beforeDue, 2)
	require.NoError(t, err)
	require.Empty(t, page)

	closed, err := store.ReadClosedRows(ctx, second, dueNs, 10)
	require.NoError(t, err)
	require.Empty(t, closed)
	closed, err = store.ReadClosedRows(ctx, second, dueNs+1, 10)
	require.NoError(t, err)
	require.Equal(t, []StoredRow{a, b}, closed)

	base := StoredRow{Fingerprint: second, BucketStartNs: 0, PodID: "", ConsumedNs: 15, ResetAtNs: 115, CompactAfterNs: dueNs + 1}
	stale := b
	stale.ConsumedNs = 4
	applied, err := store.ApplyCleanup(ctx, []StoredRow{a, stale}, []StoredRow{base})
	require.NoError(t, err)
	require.False(t, applied)
	closed, err = store.ReadClosedRows(ctx, second, dueNs+1, 10)
	require.NoError(t, err)
	require.Equal(t, []StoredRow{a, b}, closed, "a failed delete must roll back the earlier delete")

	applied, err = store.ApplyCleanup(ctx, []StoredRow{a, b}, []StoredRow{base})
	require.NoError(t, err)
	require.True(t, applied)
	closed, err = store.ReadClosedRows(ctx, second, 0, 10)
	require.NoError(t, err)
	require.Equal(t, []StoredRow{base}, closed, "the base row is always read")

	next := base
	next.ConsumedNs, next.ResetAtNs = 20, 120
	applied, err = store.ApplyCleanup(ctx, []StoredRow{base}, []StoredRow{next})
	require.NoError(t, err)
	require.True(t, applied)
	closed, err = store.ReadClosedRows(ctx, second, dueNs+1, 10)
	require.NoError(t, err)
	require.Equal(t, []StoredRow{next}, closed)
}

// TestMySQLStoreReturnsDatabaseErrors guarantees that a failed query reaches
// the caller, so flush keeps its rows and cleanup stops its pass.
func TestMySQLStoreReturnsDatabaseErrors(t *testing.T) {
	ctx := t.Context()
	database, err := shareddb.New(shareddb.Config{PrimaryDSN: containers.MySQL(t).DSN, ReadOnlyDSN: "", Tags: sqlcomment.Static{}})
	require.NoError(t, err)
	store := NewMySQLStore(database.RW(), database.RO())
	require.NoError(t, database.Close())

	r := StoredRow{Fingerprint: sha256.Sum256([]byte(uid.New(uid.TestPrefix))), BucketStartNs: 100, PodID: uid.New("", 16), ConsumedNs: 1, ResetAtNs: 101, CompactAfterNs: 1}
	_, err = store.NowNs(ctx)
	require.Error(t, err)
	require.Error(t, store.WriteRows(ctx, []StoredRow{r}))
	_, err = store.ReadRows(ctx, r.Fingerprint, r.PodID, 10)
	require.Error(t, err)
	_, err = store.ReadClosedRows(ctx, r.Fingerprint, 2, 10)
	require.Error(t, err)
	_, err = store.ListDueRows(ctx, 2, firstDueRow, 10)
	require.Error(t, err)
	_, err = store.ApplyCleanup(ctx, []StoredRow{r}, nil)
	require.Error(t, err)
}

// newMySQLTestStore returns a store on the test container through `pkg/db`,
// as the API builds it.
func newMySQLTestStore(t *testing.T) Store {
	t.Helper()
	database, err := shareddb.New(shareddb.Config{PrimaryDSN: containers.MySQL(t).DSN, ReadOnlyDSN: "", Tags: sqlcomment.Static{}})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	return NewMySQLStore(database.RW(), database.RO())
}
