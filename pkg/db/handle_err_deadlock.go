package db

import (
	"errors"
	"io"
	"net"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// MySQL server error codes
// See: https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html
const (
	// errDeadlock is returned when a deadlock is detected.
	// The entire transaction should be retried.
	errDeadlock = 1213

	// errLockWaitTimeout is returned when a lock wait timeout is exceeded.
	// By default, only the statement is rolled back (not the entire transaction).
	// The statement should be retried.
	errLockWaitTimeout = 1205

	// errTooManyConnections is returned when the server has too many connections.
	// This is a transient error that may resolve when connections are freed.
	errTooManyConnections = 1040
)

// MySQL client error codes (CR_* errors)
// See: https://dev.mysql.com/doc/mysql-errors/8.0/en/client-error-reference.html
const (
	// errServerGone is returned when the MySQL server has gone away.
	// This can happen due to connection timeout or server restart.
	errServerGone = 2006

	// errServerLost is returned when the connection to the server was lost during a query.
	// This can happen due to network issues or server crash.
	errServerLost = 2013
)

// IsDeadlockError returns true if the error is a MySQL deadlock error (1213).
// Deadlocks cause InnoDB to roll back the entire transaction.
// The entire transaction should be retried.
func IsDeadlockError(err error) bool {
	return isMySQLError(err, errDeadlock)
}

// IsLockWaitTimeoutError returns true if the error is a MySQL lock wait timeout error (1205).
// By default, only the current statement is rolled back (not the entire transaction).
// See: https://dev.mysql.com/doc/refman/8.0/en/innodb-error-handling.html
func IsLockWaitTimeoutError(err error) bool {
	return isMySQLError(err, errLockWaitTimeout)
}

// IsTooManyConnectionsError returns true if the error is a MySQL too many connections error (1040).
// This is a transient error that may resolve when other connections are closed.
func IsTooManyConnectionsError(err error) bool {
	return isMySQLError(err, errTooManyConnections)
}

// IsConnectionError returns true if the error indicates a connection problem.
// This includes server gone (2006), server lost (2013), and network errors.
// These are transient errors that may resolve on retry with a new connection.
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for MySQL client errors
	if isMySQLError(err, errServerGone) || isMySQLError(err, errServerLost) {
		return true
	}

	// Check for go-sql-driver specific errors
	if errors.Is(err, mysql.ErrInvalidConn) {
		return true
	}

	// Check for common driver error messages
	errStr := err.Error()
	if strings.Contains(errStr, "bad connection") ||
		strings.Contains(errStr, "invalid connection") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") {
		return true
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Check for EOF (connection closed)
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	return false
}

// IsTransientError returns true if the error is a transient MySQL error that should be retried.
// This includes deadlocks, lock wait timeouts, connection errors, and too many connections.
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
