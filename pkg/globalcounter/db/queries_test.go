package db_test

import (
	"database/sql"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/globalcounter/db"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestQueries(t *testing.T) {
	config := containers.MySQLIsolated(t)
	database, err := mysql.New(mysql.Config{PrimaryDSN: config.DSN, ReadOnlyDSN: ""})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	ctx := t.Context()
	queries := &db.Queries{}
	bulk := &db.BulkQueries{}

	t.Run("persist keeps the newest signed snapshot across replay and reordering", func(t *testing.T) {
		name := uid.New(uid.RatelimitNamespacePrefix)
		writerID := uid.New("", 16)
		read := db.ReadParams{Name: name, CurrentWriterID: uid.New("", 16), PreviousWriterID: uid.New("", 16)}
		row := db.PersistParams{
			Name:        name,
			WriterID:    writerID,
			Count:       100,
			Revision:    7,
			DeleteAfter: sql.NullInt64{},
		}
		require.NoError(t, bulk.Persist(ctx, database.RW(), []db.PersistParams{row}))
		newer := row
		newer.Count = -20
		newer.Revision = 8
		require.NoError(t, bulk.Persist(ctx, database.RW(), []db.PersistParams{newer, row}))
		require.NoError(t, bulk.Persist(ctx, database.RW(), []db.PersistParams{newer}))
		equalRevision := newer
		equalRevision.Count = 400
		require.NoError(t, bulk.Persist(ctx, database.RW(), []db.PersistParams{equalRevision}))
		count, err := queries.Read(ctx, database.RO(), read)
		require.NoError(t, err)
		require.Equal(t, int64(-20), count)
	})

	t.Run("persist writes compaction level 0 and a compaction_after that is later than now", func(t *testing.T) {
		name := uid.New(uid.RatelimitNamespacePrefix)
		row := db.PersistParams{Name: name, WriterID: uid.New("", 16), Count: 1, Revision: 1, DeleteAfter: sql.NullInt64{}, CompactDelayMs: time.Hour.Milliseconds()}
		require.NoError(t, bulk.Persist(ctx, database.RW(), []db.PersistParams{row}))
		listed, err := queries.ListRows(ctx, database.RO(), db.ListRowsParams{Name: name})
		require.NoError(t, err)
		require.Len(t, listed, 1)
		require.Zero(t, listed[0].CompactionLevel)
		require.Greater(t, listed[0].CompactionAfter, time.Now().Add(59*time.Minute).UnixMilli())
	})

	t.Run("read sums signed contributions excluding only the current and previous writer ids", func(t *testing.T) {
		name := uid.New(uid.RatelimitNamespacePrefix)
		otherName := uid.New(uid.RatelimitNamespacePrefix)
		ownID := uid.New("", 16)
		previousID := uid.New("", 16)
		otherID := uid.New("", 16)
		read := db.ReadParams{Name: name, CurrentWriterID: ownID, PreviousWriterID: previousID}
		count, err := queries.Read(ctx, database.RO(), read)
		require.NoError(t, err)
		require.Zero(t, count)
		rows := []db.PersistParams{
			{Name: name, WriterID: ownID, Count: 70, Revision: 1, DeleteAfter: sql.NullInt64{}},
			{Name: name, WriterID: previousID, Count: 40, Revision: 1, DeleteAfter: sql.NullInt64{}},
			{Name: name, WriterID: otherID, Count: 20, Revision: 2, DeleteAfter: sql.NullInt64{}},
			{Name: name, WriterID: uid.New("", 16), Count: -3, Revision: 1, DeleteAfter: sql.NullInt64{}},
			{Name: otherName, WriterID: otherID, Count: 500, Revision: 1, DeleteAfter: sql.NullInt64{}},
		}
		require.NoError(t, bulk.Persist(ctx, database.RW(), rows))
		count, err = queries.Read(ctx, database.RO(), read)
		require.NoError(t, err)
		require.Equal(t, int64(17), count)
		read.CurrentWriterID = otherID
		count, err = queries.Read(ctx, database.RO(), read)
		require.NoError(t, err)
		require.Equal(t, int64(67), count)
		read.PreviousWriterID = ""
		count, err = queries.Read(ctx, database.RO(), read)
		require.NoError(t, err)
		require.Equal(t, int64(107), count)
		count, err = queries.Read(ctx, database.RO(), db.ReadParams{Name: otherName, CurrentWriterID: ownID, PreviousWriterID: previousID})
		require.NoError(t, err)
		require.Equal(t, int64(500), count)
	})

	t.Run("read rejects overflow instead of clamping the sum", func(t *testing.T) {
		for _, count := range []int64{math.MaxInt64, math.MinInt64} {
			name := uid.New(uid.RatelimitNamespacePrefix)
			rows := []db.PersistParams{
				{Name: name, WriterID: uid.New("", 16), Count: count, Revision: 1, DeleteAfter: sql.NullInt64{}},
				{Name: name, WriterID: uid.New("", 16), Count: count, Revision: 1, DeleteAfter: sql.NullInt64{}},
			}
			require.NoError(t, bulk.Persist(ctx, database.RW(), rows))
			_, err := queries.Read(ctx, database.RO(), db.ReadParams{Name: name, CurrentWriterID: uid.New("", 16), PreviousWriterID: uid.New("", 16)})
			require.Error(t, err)
		}
	})

	t.Run("cleanup is bounded and leaves future and permanent rows", func(t *testing.T) {
		name := uid.New(uid.RatelimitNamespacePrefix)
		rows := []db.PersistParams{
			{Name: name, WriterID: uid.New("", 16), Count: 7, Revision: 1, DeleteAfter: sql.NullInt64{Int64: 999, Valid: true}},
			{Name: name, WriterID: uid.New("", 16), Count: 11, Revision: 1, DeleteAfter: sql.NullInt64{Int64: 1000, Valid: true}},
			{Name: name, WriterID: uid.New("", 16), Count: 13, Revision: 1, DeleteAfter: sql.NullInt64{Int64: 1001, Valid: true}},
			{Name: name, WriterID: uid.New("", 16), Count: 17, Revision: 1, DeleteAfter: sql.NullInt64{}},
		}
		require.NoError(t, bulk.Persist(ctx, database.RW(), rows))
		read := db.ReadParams{Name: name, CurrentWriterID: uid.New("", 16), PreviousWriterID: uid.New("", 16)}
		count, err := queries.Read(ctx, database.RO(), read)
		require.NoError(t, err)
		require.Equal(t, int64(48), count)
		cleanup := db.DeleteExpiredParams{CutoffMs: sql.NullInt64{Int64: 1000, Valid: true}, Limit: 1}
		for _, remaining := range []int64{41, 30} {
			deleted, err := queries.DeleteExpired(ctx, database.RW(), cleanup)
			require.NoError(t, err)
			require.Equal(t, int64(1), deleted)
			count, err = queries.Read(ctx, database.RO(), read)
			require.NoError(t, err)
			require.Equal(t, remaining, count)
		}
		deleted, err := queries.DeleteExpired(ctx, database.RW(), cleanup)
		require.NoError(t, err)
		require.Zero(t, deleted)
	})

	t.Run("a newer revision overwrites expiry, including with NULL", func(t *testing.T) {
		const base = int64(1_000_000_000)
		expiry := func(offset int64) sql.NullInt64 { return sql.NullInt64{Int64: base + offset, Valid: true} }
		writerID := uid.New("", 16)
		sweep := db.DeleteExpiredParams{CutoffMs: expiry(2000), Limit: 100}
		_, err := queries.DeleteExpired(ctx, database.RW(), db.DeleteExpiredParams{CutoffMs: expiry(2000), Limit: 100_000})
		require.NoError(t, err)

		for _, tt := range []struct {
			name     string
			rows     []db.PersistParams
			survives bool
		}{
			{name: "a later expiry replaces an earlier one", rows: []db.PersistParams{
				{Revision: 1, DeleteAfter: expiry(5000)}, {Revision: 2, DeleteAfter: expiry(2000)},
			}},
			{name: "a delayed older revision cannot extend expiry", rows: []db.PersistParams{
				{Revision: 2, DeleteAfter: expiry(2000)}, {Revision: 1, DeleteAfter: expiry(9000)},
			}},
			{name: "a NULL expiry clears the current expiry", rows: []db.PersistParams{
				{Revision: 1, DeleteAfter: expiry(2000)}, {Revision: 2, DeleteAfter: sql.NullInt64{}},
			}, survives: true},
			{name: "a delayed older NULL cannot clear expiry", rows: []db.PersistParams{
				{Revision: 2, DeleteAfter: expiry(2000)}, {Revision: 1, DeleteAfter: sql.NullInt64{}},
			}},
		} {
			name := uid.New(uid.RatelimitNamespacePrefix)
			for _, row := range tt.rows {
				row.Name, row.WriterID, row.Count = name, writerID, 1
				require.NoError(t, bulk.Persist(ctx, database.RW(), []db.PersistParams{row}), tt.name)
			}
			deleted, err := queries.DeleteExpired(ctx, database.RW(), sweep)
			require.NoError(t, err)
			want := int64(1)
			if tt.survives {
				want = 0
			}
			require.Equal(t, want, deleted, tt.name)
		}
	})
}
