package db

import (
	"context"
	"database/sql"
)

// DBTX is an interface that abstracts database operations for both
// direct connections and transactions.
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

// Queries provides all query methods.
type Queries struct{}

// Query is the global query instance.
var Query = &Queries{}
