package db

import (
	"context"
	"database/sql"
)

// Database defines the interface for partition database operations, providing access
// to read and write replicas and the ability to close connections.
type Database interface {
	// RW returns the write (primary) replica for write operations
	RW() *Replica

	// RO returns the read replica for read operations
	// If no read replica is configured, it returns the write replica
	RO() *Replica

	// Close properly terminates all database connections
	Close() error
}

// DBTX is an interface that abstracts database operations for both
// direct connections and transactions. It allows query methods to work
// with either a database or transaction, making transaction handling more
// flexible.
//
// This interface is implemented by both sql.DB and sql.Tx, as well as
// the custom Replica type in this package.
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}
