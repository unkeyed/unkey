package db

import (
	"context"
	"database/sql"
)

// DBTX accepts a database connection or transaction for generated queries.
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

// Queries is the stateless receiver for generated query methods.
type Queries struct{}

// Query provides the package's generated read queries.
var Query Querier = &Queries{}
