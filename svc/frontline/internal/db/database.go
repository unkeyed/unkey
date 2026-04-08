package db

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/pkg/mysql"
	mysqlmetrics "github.com/unkeyed/unkey/pkg/mysql/metrics"
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

// Queries implements [Querier] by executing sqlc-generated SQL against a [DBTX].
type Queries struct {
	db DBTX
}

// WithTx returns a new [Queries] that executes all subsequent calls within the
// given transaction. The caller is responsible for committing or rolling back tx.
func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db: tx,
	}
}

// New opens a read-only MySQL replica connection and returns a [Querier] ready
// for use. The connection is created with mode "ro" so that tracing and metrics
// labels distinguish replica traffic from primary traffic.
//
// The second return value is a close function that releases the underlying
// connection pool. Callers must invoke it during shutdown to avoid leaking
// connections. On error, both the Querier and close function are nil.
func New(url string, metrics *mysqlmetrics.Metrics) (Querier, func() error, error) {
	replica, err := mysql.NewReplica(url, "ro", metrics)
	if err != nil {
		return nil, nil, err
	}

	return &Queries{db: replica}, replica.Close, nil
}
