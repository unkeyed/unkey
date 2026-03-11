package mysql

import (
	"errors"
	"io"
	"net"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// MySQL server error codes
const (
	errDeadlock           = 1213
	errLockWaitTimeout    = 1205
	errTooManyConnections = 1040
)

// MySQL client error codes (CR_* errors)
const (
	errServerGone = 2006
	errServerLost = 2013
)

// IsDeadlockError returns true if the error is a MySQL deadlock error (1213).
func IsDeadlockError(err error) bool {
	return isMySQLError(err, errDeadlock)
}

// IsLockWaitTimeoutError returns true if the error is a MySQL lock wait timeout error (1205).
func IsLockWaitTimeoutError(err error) bool {
	return isMySQLError(err, errLockWaitTimeout)
}

// IsTooManyConnectionsError returns true if the error is a MySQL too many connections error (1040).
func IsTooManyConnectionsError(err error) bool {
	return isMySQLError(err, errTooManyConnections)
}

// IsConnectionError returns true if the error indicates a connection problem.
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	if isMySQLError(err, errServerGone) || isMySQLError(err, errServerLost) {
		return true
	}

	if errors.Is(err, mysql.ErrInvalidConn) {
		return true
	}

	errStr := err.Error()
	if strings.Contains(errStr, "bad connection") ||
		strings.Contains(errStr, "invalid connection") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	return false
}

// IsTransientError returns true if the error is a transient MySQL error that should be retried.
func IsTransientError(err error) bool {
	return IsDeadlockError(err) ||
		IsLockWaitTimeoutError(err) ||
		IsConnectionError(err) ||
		IsTooManyConnectionsError(err)
}

// isMySQLError checks if the error is a MySQL error with the given error number.
func isMySQLError(err error, number uint16) bool {
	if err == nil {
		return false
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == number {
		return true
	}

	return false
}
