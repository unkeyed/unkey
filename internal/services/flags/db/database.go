package db

import (
	"context"
	"database/sql"
)

// DBTX is the database interface required by generated query methods. Either a
// plain connection or a transaction satisfies it.
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

// Queries is the stateless receiver for sqlc-generated query methods. Each
// method takes a [DBTX] because emit_methods_with_db_argument is true.
type Queries struct{}

// Query is the package-level entry point for calling generated query methods.
var Query Querier = &Queries{}
