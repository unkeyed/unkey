package db

import (
	"context"
	"database/sql"
)

// DBTX is the subset of [sql.DB] and [sql.Tx] that the generated queries use,
// so a query runs the same on a connection, a replica, or in a transaction.
type DBTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Queries holds the generated single-row queries. It has no state; the caller
// passes the connection to every call.
type Queries struct{}

// BulkQueries holds the generated multi-row upserts.
type BulkQueries struct{}

// Query runs the generated queries of the ratelimit_state table.
var Query Querier = &Queries{}

// BulkQuery runs the generated multi-row inserts of the ratelimit_state table.
var BulkQuery BulkQuerier = &BulkQueries{}
