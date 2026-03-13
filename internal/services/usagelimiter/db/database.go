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

type Queries struct{}

var Query Querier = &Queries{}

type Database = mysql.Database
type Config = mysql.Config

var (
	IsNotFound = mysql.IsNotFound
)

func New(config Config) (mysql.Database, error) {
	return mysql.New(config)

}
