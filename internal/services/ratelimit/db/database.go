package db

import (
	"context"
	"database/sql"
)

// DBTX is the database interface required by sqlc-generated query methods. It
// mirrors the subset of [database/sql.DB] and [database/sql.Tx] that the
// generated code calls, so either a plain connection or a transaction
// satisfies it. [pkg/mysql.Replica] implements this interface, so callers can
// hand in primary and replica connections directly.
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

// Queries implements [Querier] by executing sqlc-generated SQL against a
// [DBTX]. The sqlc generator attaches one method per query to this type.
type Queries struct {
	db DBTX
}

// WithTx returns a new [Queries] that executes all subsequent calls within
// the given transaction. The caller is responsible for committing or rolling
// back tx.
func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{db: tx}
}

// Database splits queries across primary and replica connections. Reads go to
// the replica and writes go to the primary. Both [Queries] instances expose the
// full Querier interface; the split is enforced by the caller picking RW() or
// RO() at the call site.
type Database struct {
	rw *Queries
	ro *Queries
}

// New wraps two [DBTX] connections (primary for writes, replica for reads)
// into a [Database]. The caller owns the lifetime of the underlying
// connections; this package never closes them.
func New(rw, ro DBTX) *Database {
	return &Database{
		rw: &Queries{db: rw},
		ro: &Queries{db: ro},
	}
}

// RW returns the [Querier] backed by the primary connection.
func (d *Database) RW() Querier { return d.rw }

// RO returns the [Querier] backed by the read replica.
func (d *Database) RO() Querier { return d.ro }
