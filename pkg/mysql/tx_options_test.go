package mysql_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/globalcounter/db"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestTxRetryWithOptions_BeginsTheTransactionWithTheGivenIsolation(t *testing.T) {
	config := containers.MySQLIsolated(t)
	database, err := mysql.New(mysql.Config{PrimaryDSN: config.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	// A row that another connection commits while the transaction is open is
	// visible to a second read at READ COMMITTED, and not at REPEATABLE READ.
	secondRead := func(opts *sql.TxOptions) int64 {
		name := uid.New(uid.RatelimitNamespacePrefix)
		var second int64
		require.NoError(t, mysql.TxRetryWithOptions(t.Context(), database.RW(), opts, func(ctx context.Context, tx mysql.DBTX) error {
			params := db.ReadParams{Name: name, CurrentWriterID: "", PreviousWriterID: ""}
			first, err := db.Query.Read(ctx, tx, params)
			require.NoError(t, err)
			require.Zero(t, first)
			require.NoError(t, (&db.BulkQueries{}).Persist(ctx, database.RW(), []db.PersistParams{
				{Name: name, WriterID: uid.New("", 16), Count: 5, Revision: 1, DeleteAfter: sql.NullInt64{}, CompactDelayMs: 0},
			}))
			second, err = db.Query.Read(ctx, tx, params)
			return err
		}))
		return second
	}

	require.Zero(t, secondRead(nil), "the default isolation is REPEATABLE READ, which keeps its snapshot")
	require.Equal(t, int64(5), secondRead(&sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: false}))
}
