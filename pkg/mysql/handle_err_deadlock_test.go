package mysql

import (
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestIsTransactionTimeoutError(t *testing.T) {
	t.Parallel()

	vitessTimeout := &mysql.MySQLError{
		Number:  1105,
		Message: "target: unkey.-.primary: vttablet: rpc error: code = Aborted desc = transaction 1782950310686082683: ended at 2026-07-02 05:31:19.003 UTC (exceeded timeout: 20s)",
	}

	t.Run("vitess transaction timeout", func(t *testing.T) {
		require.True(t, IsTransactionTimeoutError(vitessTimeout))
		require.True(t, IsTransientError(vitessTimeout))
	})

	t.Run("wrapped vitess transaction timeout", func(t *testing.T) {
		wrapped := fmt.Errorf("database failed to update ratelimit: %w", vitessTimeout)
		require.True(t, IsTransactionTimeoutError(wrapped))
		require.True(t, IsTransientError(wrapped))
	})

	t.Run("other vitess 1105 error", func(t *testing.T) {
		other := &mysql.MySQLError{Number: 1105, Message: "syntax error near 'FROM'"}
		require.False(t, IsTransactionTimeoutError(other))
		require.False(t, IsTransientError(other))
	})

	t.Run("innodb lock wait timeout is not vitess timeout", func(t *testing.T) {
		lockWait := &mysql.MySQLError{Number: 1205, Message: "Lock wait timeout exceeded"}
		require.False(t, IsTransactionTimeoutError(lockWait))
		require.True(t, IsTransientError(lockWait))
	})
}
