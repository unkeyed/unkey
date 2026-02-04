package db

import (
	"context"

	"github.com/unkeyed/unkey/pkg/wide"
)

// Database error type constants for classification.
const (
	DBErrorTypeNotFound           = "not_found"
	DBErrorTypeDuplicateKey       = "duplicate_key"
	DBErrorTypeDeadlock           = "deadlock"
	DBErrorTypeLockTimeout        = "lock_timeout"
	DBErrorTypeTooManyConnections = "too_many_connections"
	DBErrorTypeConnection         = "connection_error"
	DBErrorTypeUnknown            = "unknown"
)

// ClassifyDBError returns a string classification of the database error.
// This is useful for logging and metrics to understand what type of error occurred.
func ClassifyDBError(err error) string {
	if err == nil {
		return ""
	}

	switch {
	case IsNotFound(err):
		return DBErrorTypeNotFound
	case IsDuplicateKeyError(err):
		return DBErrorTypeDuplicateKey
	case IsDeadlockError(err):
		return DBErrorTypeDeadlock
	case IsLockWaitTimeoutError(err):
		return DBErrorTypeLockTimeout
	case IsTooManyConnectionsError(err):
		return DBErrorTypeTooManyConnections
	case IsConnectionError(err):
		return DBErrorTypeConnection
	default:
		return DBErrorTypeUnknown
	}
}

// SetDBErrorContext sets the db_error_type wide field when a database error occurs.
func SetDBErrorContext(ctx context.Context, err error) {
	if err != nil {
		wide.Set(ctx, wide.FieldDBErrorType, ClassifyDBError(err))
	}
}
