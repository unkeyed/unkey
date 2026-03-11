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

// Queries implements [Querier] by executing sqlc-generated SQL against a [DBTX].
type Queries struct {
	db DBTX
}

// WithTx returns a new [Queries] that executes all subsequent calls within the
// given transaction. The caller is responsible for committing or rolling back tx.
func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{db: tx}
}

// New opens a writable MySQL connection and returns a [Querier] ready for use.
// The connection is created with mode "rw" so tracing and metrics are labeled
// consistently with primary-DB traffic.
func New(url string) (Querier, func() error, error) {
	primary, err := mysql.NewReplica(url, "rw")
	if err != nil {
		return nil, nil, err
	}

	return &Queries{db: primary}, primary.Close, nil
}
