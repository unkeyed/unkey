package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/unkeyed/unkey/go/pkg/fault"
)

// Replica represents a database connection (either primary or read-only)
// and provides methods for executing queries and transactions.
type Replica struct {
	db   *sql.DB
	mode string // "rw" for read-write, "ro" for read-only
}

// DB returns the underlying sql.DB connection.
// This is useful when you need to access the raw database connection.
func (r *Replica) DB() *sql.DB {
	return r.db
}

// Mode returns the mode of this replica ("rw" or "ro").
func (r *Replica) Mode() string {
	return r.mode
}

// ExecContext executes a query without returning any rows.
// This is typically used for INSERT, UPDATE, DELETE, and DDL statements.
func (r *Replica) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal(fmt.Sprintf("failed to execute query on %s replica", r.mode)))
	}
	return result, nil
}

// PrepareContext creates a prepared statement for later queries or executions.
func (r *Replica) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal(fmt.Sprintf("failed to prepare statement on %s replica", r.mode)))
	}
	return stmt, nil
}

// QueryContext executes a query that returns rows.
func (r *Replica) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal(fmt.Sprintf("failed to execute query on %s replica", r.mode)))
	}
	return rows, nil
}

// QueryRowContext executes a query that is expected to return at most one row.
func (r *Replica) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return r.db.QueryRowContext(ctx, query, args...)
}

// BeginTx starts a transaction with the given options.
// Note: Transactions should only be used on write replicas.
func (r *Replica) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if r.mode == "ro" {
		return nil, fault.New("cannot start transaction on read-only replica")
	}

	tx, err := r.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to begin transaction"))
	}
	return tx, nil
}

// Ping verifies a connection to the database is still alive.
func (r *Replica) Ping() error {
	err := r.db.Ping()
	if err != nil {
		return fault.Wrap(err, fault.Internal(fmt.Sprintf("failed to ping %s replica", r.mode)))
	}
	return nil
}
