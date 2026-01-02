package db

import (
	"database/sql"
	"errors"
)

// IsNotFound returns true if the error is sql.ErrNoRows.
// Use this for consistent not-found handling across the codebase.
func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
