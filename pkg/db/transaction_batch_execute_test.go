package db

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

func TestExecuteTransactionBatch(t *testing.T) {
	mysqlConfig := containers.MySQL(t)
	database, err := New(Config{
		PrimaryDSN:  mysqlConfig.DSN,
		ReadOnlyDSN: "",
		Tags:        sqlcomment.Disabled(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	err = database.BatchRW().executeTransactionBatch(t.Context(), "TestExecuteTransactionBatch", []transactionBatchStatement{{
		query: "SET @unkey_transaction_batch_test = ?",
		args:  []any{42},
	}})
	require.NoError(t, err)
}
