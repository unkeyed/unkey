package db

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestRollbackError(t *testing.T) {
	cause := &mysql.MySQLError{Number: 3024, Message: "statement timed out"}
	for _, rollbackErr := range []error{nil, sql.ErrTxDone, fmt.Errorf("rollback: %w", sql.ErrTxDone)} {
		require.Same(t, cause, rollbackError(cause, rollbackErr))
	}

	rollbackErr := errors.New("connection lost during rollback")
	err := rollbackError(cause, rollbackErr)
	require.ErrorIs(t, err, cause)
	require.ErrorIs(t, err, rollbackErr)
	var mysqlErr *mysql.MySQLError
	require.ErrorAs(t, err, &mysqlErr)
	require.Equal(t, uint16(3024), mysqlErr.Number)
	require.Contains(t, err.Error(), cause.Error())
	require.Contains(t, err.Error(), rollbackErr.Error())
	require.Equal(t, "Unable to rollback database transaction.", fault.UserFacingMessage(err))
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.App.Internal.ServiceUnavailable.URN(), code)
}
