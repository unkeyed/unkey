package db

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/pkg/mysql"
)

// DBTX is the database interface required by generated query methods. It mirrors
// the subset of [database/sql.DB] and [database/sql.Tx] that sqlc-generated code
// calls, so either a plain connection or a transaction satisfies it.
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

// Queries is the receiver for sqlc-generated query methods. Because
// emit_methods_with_db_argument is true, each method takes a DBTX
// parameter and the struct itself is stateless.
type Queries struct{}

// Query is the package-level entry point for calling generated query methods.
var Query Querier = &Queries{}

// Database holds read and write replicas for the keys service.
type Database struct {
	ro DBTX
	rw DBTX
}

// New creates a Database from a [mysql.MySQL] instance.
func New(db mysql.MySQL) *Database {
	return &Database{ro: db.RO(), rw: db.RW()}
}

// RO returns the read replica.
func (d *Database) RO() DBTX { return d.ro }

// RW returns the write replica.
func (d *Database) RW() DBTX { return d.rw }
